// under the Apache License Version 2.0. This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf

package telemetry

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	manager "github.com/DataDog/ebpf-manager"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/slices"
	"golang.org/x/sys/unix"

	"github.com/DataDog/datadog-agent/pkg/network/config"
	netbpf "github.com/DataDog/datadog-agent/pkg/network/ebpf"
	"github.com/DataDog/datadog-agent/pkg/network/ebpf/probes"
	"github.com/DataDog/datadog-agent/pkg/util/kernel"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	maxErrno    = 64
	maxErrnoStr = "other"

	ebpfMapTelemetryNS    = "ebpf_maps"
	ebpfHelperTelemetryNS = "ebpf_helpers"

	maxTrampolineOffset = 2
)

const (
	ebpfEntryTrampolinePatchCall int64 = -1 // patch constant for trampoline instruction
	ebpfTelemetryMapErrors             = -2
	ebpfTelemetryHelperErrors          = -3
)

type eBPFInstrumentation struct {
	patchType   int64
	programName string
}

var instrumentation = []eBPFInstrumentation{
	{ebpfEntryTrampolinePatchCall, "ebpf_instrumentation__trampoline_handler"},
	{ebpfTelemetryMapErrors, "ebpf_instrumentation__map_error_telemetry"},
	{ebpfTelemetryHelperErrors, "ebpf_instrumentation__helper_error_telemetry"},
}

const (
	readIndx int = iota
	readUserIndx
	readKernelIndx
	skbLoadBytes
	perfEventOutput
)

var ebpfMapOpsErrorsGauge = prometheus.NewDesc(fmt.Sprintf("%s__errors", ebpfMapTelemetryNS), "Failures of map operations for a specific ebpf map reported per error.", []string{"map_name", "error"}, nil)
var ebpfHelperErrorsGauge = prometheus.NewDesc(fmt.Sprintf("%s__errors", ebpfHelperTelemetryNS), "Failures of bpf helper operations reported per helper per error for each probe.", []string{"helper", "probe_name", "error"}, nil)

var helperNames = map[int]string{
	readIndx:        "bpf_probe_read",
	readUserIndx:    "bpf_probe_read_user",
	readKernelIndx:  "bpf_probe_read_kernel",
	skbLoadBytes:    "bpf_skb_load_bytes",
	perfEventOutput: "bpf_perf_event_output",
}

// EBPFTelemetry struct contains all the maps that
// are registered to have their telemetry collected.
type EBPFTelemetry struct {
	mtx             sync.Mutex
	bpfTelemetryMap *ebpf.Map
	mapKeys         map[string]uint64
	mapIndex        uint64
	probeKeys       map[string]uint64
	programIndex    uint64
	Config          *config.Config
}

// NewEBPFTelemetry initializes a new EBPFTelemetry object
func NewEBPFTelemetry() *EBPFTelemetry {
	if supported, _ := ebpfTelemetrySupported(); !supported {
		return nil
	}
	return &EBPFTelemetry{
		mapKeys:   make(map[string]uint64),
		probeKeys: make(map[string]uint64),
	}
}

// populateMapsWithKeys initializes the maps for holding telemetry info.
// It must be called after the manager is initialized
func (b *EBPFTelemetry) populateMapsWithKeys(m *manager.Manager) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	// first manager to call will populate the maps
	if b.bpfTelemetryMap != nil {
		return nil
	}
	b.bpfTelemetryMap, _, _ = m.GetMap(probes.EBPFTelemetryMap)

	key := 0
	z := new(InstrumentationBlob)
	err := b.bpfTelemetryMap.Update(unsafe.Pointer(&key), unsafe.Pointer(z), ebpf.UpdateNoExist)
	if err != nil && !errors.Is(err, ebpf.ErrKeyExist) {
		return fmt.Errorf("failed to initialize telemetry struct")
	}

	return nil
}

// Describe returns all descriptions of the collector
func (b *EBPFTelemetry) Describe(ch chan<- *prometheus.Desc) {
	ch <- ebpfMapOpsErrorsGauge
	ch <- ebpfHelperErrorsGauge
}

// Collect returns the current state of all metrics of the collector
func (b *EBPFTelemetry) Collect(ch chan<- prometheus.Metric) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.bpfTelemetryMap != nil {
		var val InstrumentationBlob
		key := 0
		err := b.bpfTelemetryMap.Lookup(unsafe.Pointer(&key), unsafe.Pointer(&val))
		if err != nil {
			log.Debugf("failed to get instrumentation blob")
		}

		for mapName, mapIndx := range b.mapKeys {
			if count := getErrCount(val.Map_err_telemetry[mapIndx].Count[:]); len(count) > 0 {
				for errStr, errCount := range count {
					ch <- prometheus.MustNewConstMetric(ebpfMapOpsErrorsGauge, prometheus.GaugeValue, float64(errCount), mapName, errStr)
				}
			}
		}

		for programName, programIndex := range b.probeKeys {
			for index, helperName := range helperNames {
				base := maxErrno * index
				if count := getErrCount(val.Helper_err_telemetry[programIndex].Count[base : base+maxErrno]); len(count) > 0 {
					for errStr, errCount := range count {
						ch <- prometheus.MustNewConstMetric(ebpfHelperErrorsGauge, prometheus.GaugeValue, float64(errCount), helperName, programName, errStr)
					}
				}
			}
		}
	}
}

func getErrCount(v []uint64) map[string]uint64 {
	errCount := make(map[string]uint64)
	for i, count := range v {
		if count == 0 {
			continue
		}

		if (i + 1) == maxErrno {
			errCount[maxErrnoStr] = count
		} else if name := unix.ErrnoName(syscall.Errno(i)); name != "" {
			errCount[name] = count
		} else {
			errCount[syscall.Errno(i).Error()] = count
		}
	}
	return errCount
}

func countRawBPFIns(ins *asm.Instruction) int64 {
	if ins.OpCode.IsDWordLoad() {
		return 2
	}

	return 1
}

type patchSite struct {
	ins       *asm.Instruction
	callsite  int64
	patchType int64
	insIndex  int
}

func initializeProbeKeys(m *manager.Manager, bpfTelemetry *EBPFTelemetry) error {
	bpfTelemetry.mtx.Lock()
	defer bpfTelemetry.mtx.Unlock()

	progs, err := m.GetProgramSpecs()
	if err != nil {
		return fmt.Errorf("failed to get program specs: %w", err)
	}

	for fn, _ := range progs {
		bpfTelemetry.probeKeys[fn] = bpfTelemetry.programIndex
		bpfTelemetry.programIndex++
	}

	return nil
}

func patchEBPFTelemetry(m *manager.Manager, enable bool, bpfTelemetry *EBPFTelemetry, bpfDir string, bytecode io.ReaderAt) error {
	ldDWImm := asm.LoadImmOp(asm.DWord)

	if err := initializeProbeKeys(m, bpfTelemetry); err != nil {
		return err
	}

	progs, err := m.GetProgramSpecs()
	if err != nil {
		return err
	}

	sizes, err := parseStackSizesSections(bytecode, progs)
	if err != nil {
		return fmt.Errorf("failed to parse '.stack_sizes' section in file: %w", err)
	}

	for fn, p := range progs {
		space := sizes.stackHas8BytesFree(fn)
		if !space {
			log.Warnf("Function %s does not have enough free stack space for instrumentation", fn)
			continue
		}

		//turnOn := enable && bpfTelemetry != nil && space

		// build map of patch sites. These are the sites where we patch the jumps to our instrumnetation blocks.
		iter := p.Instructions.Iterate()
		patchSites := make(map[int64][]patchSite)
		var insCount int64
		for iter.Next() {
			ins := iter.Ins
			insCount += countRawBPFIns(ins)

			if (ins.OpCode.JumpOp() == asm.Call && ins.Constant == ebpfEntryTrampolinePatchCall) ||
				(ins.IsBuiltinCall() && (ins.Constant == ebpfTelemetryMapErrors || ins.Constant == ebpfTelemetryHelperErrors)) {
				if ins.Constant == ebpfEntryTrampolinePatchCall && iter.Offset > maxTrampolineOffset {
					return fmt.Errorf("trampoline instruction found at disallowed offset %d\n", iter.Offset)
				}

				if _, ok := patchSites[ins.Constant]; ok {
					patchSites[ins.Constant] = append(patchSites[ins.Constant], patchSite{ins, int64(iter.Offset), ins.Constant, iter.Index})
				} else {
					patchSites[ins.Constant] = []patchSite{{ins, int64(iter.Offset), ins.Constant, iter.Index}}
				}
			}
		}
		programInsCount := insCount

		// setup instrumentation block
		bpfAsset, err := netbpf.ReadEBPFTelemetryModule(bpfDir, "ebpf_instrumentation")
		if err != nil {
			return fmt.Errorf("failed to read ebpf_instrumentation.o bytecode file: %w", err)
		}
		collectionSpec, err := ebpf.LoadCollectionSpecFromReader(bpfAsset)
		if err != nil {
			return fmt.Errorf("failed to load collection spec from reader: %w", err)
		}

		var instrumentationBlock []*asm.Instruction
		for _, instr := range instrumentation {
			if _, ok := patchSites[instr.patchType]; !ok {
				// This instrumentation is not used
				continue
			}

			iter := collectionSpec.Programs[instr.programName].Instructions.Iterate()
			var blockCount int64
			for iter.Next() {
				ins := iter.Ins

				// The final instruction in the instrumentation block is `exit`, which we
				// do not want.
				if ins.OpCode.JumpOp() == asm.Exit {
					break
				}
				blockCount += countRawBPFIns(ins)

				// The first instruction has associated func_info btf information. Since
				// the instrumentation is not a function, the verifier will complain that the number of
				// `func_info` objects in the BTF do not match the number of loaded programs:
				// https://elixir.bootlin.com/linux/latest/source/kernel/bpf/verifier.c#L15035
				// To workaround this we create a new instruction object and give it empty metadata.
				if iter.Index == 0 {
					newIns := asm.Instruction{
						OpCode:   ins.OpCode,
						Dst:      ins.Dst,
						Src:      ins.Src,
						Offset:   ins.Offset,
						Constant: ins.Constant,
					}.WithMetadata(asm.Metadata{})

					instrumentationBlock = append(
						instrumentationBlock,
						&newIns,
					)
					continue
				}

				instrumentationBlock = append(instrumentationBlock, ins)
			}

			// for each callsite to this instrumentation block in the original function, append a jump
			// back to the correct ip after the instrumentation is complete
			sites := patchSites[instr.patchType]
			fmt.Printf("Program %s Block %s sites: %v\n", fn, instr.programName, sites)
			for _, site := range sites {
				retJumpOffset := site.callsite - (insCount + blockCount)

				// for the entry trampoline insert an unconditional jump since there can be only 1 call site.
				if site.patchType == ebpfEntryTrampolinePatchCall {
					//					newIns := asm.Instruction{OpCode: asm.OpCode(asm.JumpClass).SetJumpOp(asm.Ja), Offset: int16(retJumpOffset)}
					fmt.Printf("trampoline return address: %d\n", retJumpOffset)
					newIns := asm.Instruction{
						OpCode:   asm.OpCode(asm.JumpClass).SetJumpOp(asm.JEq).SetSource(asm.ImmSource),
						Dst:      asm.R0,
						Offset:   int16(retJumpOffset),
						Constant: int64(1),
					}
					instrumentationBlock = append(instrumentationBlock, &newIns)
				} else {
					// for all other callsites the instrumentation code is generated such that R0 is an unsigned long constant.
					// patch the desired return address into r0. This way the appropriate jump will be taken.
					newIns := asm.Instruction{
						OpCode:   asm.OpCode(asm.JumpClass).SetJumpOp(asm.JEq).SetSource(asm.ImmSource),
						Dst:      asm.R0,
						Offset:   int16(retJumpOffset),
						Constant: int64(retJumpOffset),
					}
					instrumentationBlock = append(
						instrumentationBlock,
						// if r1 == callsite goto callsite+1
						&newIns,
					)
				}

				blockCount++
			}

			// the verifier requires the last instruction to be an unconditional jump or exit
			// https://elixir.bootlin.com/linux/latest/source/kernel/bpf/verifier.c#L2877
			//
			// Moreover, we need an exit or unconditional jump after each instrumentation block
			// so the verifier does not fallthrough and analyze the next block.
			//if instr.patchType != ebpfEntryTrampolinePatchCall {
			//	returnIns := asm.Return()
			//	instrumentationBlock = append(instrumentationBlock, &returnIns)
			//	blockCount++
			//}

			fmt.Printf("Program: %s\n", fn)
			// patch the original callsites to jump to instrumentation block
			const retpolineArg = "retpoline_jump_addr"
			for _, site := range sites {
				if site.patchType != ebpfEntryTrampolinePatchCall {
					for idx := site.insIndex - 1; idx > site.insIndex-5 && idx >= 0; idx-- {
						load := &p.Instructions[idx]
						// The load has to be there before the patch call
						if load.OpCode.JumpOp() == asm.Call {
							return fmt.Errorf("failed to discover load instruction to patch trampoline return address")
						}

						// keep looking until we find the load instruction to patch the return address in
						if load.OpCode == ldDWImm && load.Reference() == retpolineArg {
							load.Constant = site.callsite
						}

						// patch map index
						if site.patchType == ebpfTelemetryMapErrors {
							if index, ok := bpfTelemetry.mapKeys[strings.TrimSuffix(load.Reference(), "_telemetry_key")]; load.OpCode == ldDWImm && ok {
								load.Constant = int64(index)
							}
						}
					}
				}

				// The trampoline instruction is an unconditional jump to the start of this instrumentation block.
				//
				// Point the instrumented instruction to the metadata of the original instruction. If not provided,
				// some programs can end up with corrupted BTF. This happens for programs where the instrumented instruction
				// is the first instruction. In that case the loader expects this instruction to point to the `func_info` BTF.
				if site.patchType == ebpfEntryTrampolinePatchCall {
					*(site.ins) = asm.Instruction{
						OpCode: asm.OpCode(asm.JumpClass).SetJumpOp(asm.Ja),
						Offset: int16(insCount - site.callsite - 1),
					}.WithMetadata(site.ins.Metadata)
				} else {
					*(site.ins) = asm.Instruction{
						OpCode:   asm.OpCode(asm.JumpClass).SetJumpOp(asm.JEq).SetSource(asm.ImmSource),
						Dst:      asm.R1,
						Offset:   int16(insCount - site.callsite - 1),
						Constant: int64(site.callsite),
					}.WithMetadata(site.ins.Metadata)
				}

				fmt.Printf("[%d] %v\n", site.insIndex, site.ins)
			}

			// patch the program index in helper telemetry block
			const symbol = "telemetry_program_id_key"
			if instr.patchType == ebpfTelemetryHelperErrors {
				ins := collectionSpec.Programs[instr.programName].Instructions
				offsets := ins.ReferenceOffsets()
				indices := offsets[symbol]
				if len(indices) > 0 {
					for _, index := range indices {
						load := &ins[index]
						if load.OpCode != ldDWImm {
							return fmt.Errorf("symbol %v: load: found %v instead of %v", symbol, load.OpCode, ldDWImm)
						}
						load.Constant = int64(bpfTelemetry.probeKeys[fn])
					}
				}
			}

			insCount += blockCount

			fmt.Printf("Instrumented %s for program %s\n", instr.programName, fn)
		}

		returnIns := asm.Return()
		instrumentationBlock = append(instrumentationBlock, &returnIns)

		// append the instrumentation code to the instructions
		for _, ins := range instrumentationBlock {
			programInsCount += countRawBPFIns(ins)
			fmt.Printf("[%d] %v\n", programInsCount-1, ins)
			p.Instructions = append(p.Instructions, *ins)
		}
	}

	return nil
}

// setupForTelemetry sets up the manager to handle eBPF telemetry.
// It will patch the instructions of all the manager probes and `undefinedProbes` provided.
// Constants are replaced for map error and helper error keys with their respective values.
// This must be called before ebpf-manager.Manager.Init/InitWithOptions
func setupForTelemetry(m *manager.Manager, options *manager.Options, bpfTelemetry *EBPFTelemetry, bpfDir string, bytecode io.ReaderAt) error {
	bpfTelemetry.mtx.Lock()
	defer bpfTelemetry.mtx.Unlock()

	activateBPFTelemetry, err := ebpfTelemetrySupported()
	if err != nil {
		return err
	}
	m.InstructionPatcher = func(m *manager.Manager) error {
		return patchEBPFTelemetry(m, activateBPFTelemetry, bpfTelemetry, bpfDir, bytecode)
	}

	if activateBPFTelemetry {
		// add telemetry maps to list of maps, if not present
		if !slices.ContainsFunc(m.Maps, func(x *manager.Map) bool { return x.Name == probes.EBPFTelemetryMap }) {
			m.Maps = append(m.Maps, &manager.Map{Name: probes.EBPFTelemetryMap})
		}

		if bpfTelemetry != nil {
			bpfTelemetry.setupMapEditors(options)

			for _, m := range m.Maps {
				bpfTelemetry.mapKeys[m.Name] = bpfTelemetry.mapIndex
				bpfTelemetry.mapIndex++
			}
		}
	}
	// we cannot exclude the telemetry maps because on some kernels, deadcode elimination hasn't removed references
	// if telemetry not enabled: leave key constants as zero, and deadcode elimination should reduce number of instructions

	return nil
}

func (b *EBPFTelemetry) setupMapEditors(opts *manager.Options) {
	if b.bpfTelemetryMap != nil && opts.MapEditors == nil {
		opts.MapEditors = make(map[string]*ebpf.Map)
		opts.MapEditors[probes.EBPFTelemetryMap] = b.bpfTelemetryMap
	}
}

// ebpfTelemetrySupported returns whether eBPF telemetry is supported, which depends on the verifier in 4.14+
func ebpfTelemetrySupported() (bool, error) {
	kversion, err := kernel.HostVersion()
	if err != nil {
		return false, err
	}
	return kversion >= kernel.VersionCode(4, 14, 0), nil
}
