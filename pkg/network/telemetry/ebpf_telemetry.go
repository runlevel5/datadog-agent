// under the Apache License Version 2.0. This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf

package telemetry

import (
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
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

type eBPFPatchCall int

const (
	ebpfEntryTrampolinePatchCall eBPFPatchCall = -1 // patch constant for trampoline instruction
	ebpfTelemetryMapErrors       eBPFPatchCall = -2
	ebpfTelemetryHelperErrors    eBPFPatchCall = -3
)

type eBPFInstrumentation struct {
	callSite    eBPFPatchCall
	programName string
}

var instrumentation = []eBPFInstrumentation{
	{ebpfEntryTrampolinePatchCall, "ebpf_instrumentation__trampoline_handler"},
	{ebpfTelemetryPatchCall, "ebpf_instrumentation__map_error_telemetry"},
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
	mapErrMap       *ebpf.Map
	helperErrMap    *ebpf.Map
	bpfTelemetryMap *ebpf.Map
	mapKeys         map[string]uint64
	probeKeys       map[string]uint64
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
	if b.mapErrMap == nil {
		b.mapErrMap, _, _ = m.GetMap(probes.MapErrTelemetryMap)
	}
	if b.helperErrMap == nil {
		b.helperErrMap, _, _ = m.GetMap(probes.HelperErrTelemetryMap)
	}
	if b.bpfTelemetryMap == nil {
		b.bpfTelemetryMap, _, _ = m.GetMap(probes.EBPFTelemetryMap)
	}

	if err := b.initializeMapErrTelemetryMap(m.Maps); err != nil {
		return err
	}
	if err := b.initializeHelperErrTelemetryMap(); err != nil {
		return err
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

	if b.helperErrMap != nil {
		var hval HelperErrTelemetry
		for probeName, k := range b.probeKeys {
			err := b.helperErrMap.Lookup(unsafe.Pointer(&k), unsafe.Pointer(&hval))
			if err != nil {
				log.Debugf("failed to get telemetry for probe:key %s:%d\n", probeName, k)
				continue
			}
			for indx, helperName := range helperNames {
				base := maxErrno * indx
				if count := getErrCount(hval.Count[base : base+maxErrno]); len(count) > 0 {
					for errStr, errCount := range count {
						ch <- prometheus.MustNewConstMetric(ebpfHelperErrorsGauge, prometheus.GaugeValue, float64(errCount), helperName, probeName, errStr)
					}
				}
			}
		}
	}

	if b.mapErrMap != nil {
		var val MapErrTelemetry
		for m, k := range b.mapKeys {
			err := b.mapErrMap.Lookup(unsafe.Pointer(&k), unsafe.Pointer(&val))
			if err != nil {
				log.Debugf("failed to get telemetry for map:key %s:%d\n", m, k)
				continue
			}
			if count := getErrCount(val.Count[:]); len(count) > 0 {
				for errStr, errCount := range count {
					ch <- prometheus.MustNewConstMetric(ebpfMapOpsErrorsGauge, prometheus.GaugeValue, float64(errCount), m, errStr)
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

func buildMapErrTelemetryConstants(mgr *manager.Manager) []manager.ConstantEditor {
	var keys []manager.ConstantEditor
	h := keyHash()
	for _, m := range mgr.Maps {
		keys = append(keys, manager.ConstantEditor{
			Name:  m.Name + "_telemetry_key",
			Value: mapKey(h, m),
		})
	}
	return keys
}

func keyHash() hash.Hash64 {
	return fnv.New64a()
}

func mapKey(h hash.Hash64, m *manager.Map) uint64 {
	h.Reset()
	_, _ = h.Write([]byte(m.Name))
	return h.Sum64()
}

func probeKey(h hash.Hash64, funcName string) uint64 {
	h.Reset()
	_, _ = h.Write([]byte(funcName))
	return h.Sum64()
}

func (b *EBPFTelemetry) initializeMapErrTelemetryMap(maps []*manager.Map) error {
	if b.mapErrMap == nil {
		return nil
	}

	z := new(MapErrTelemetry)
	h := keyHash()
	for _, m := range maps {
		// Some maps, such as the telemetry maps, are
		// redefined in multiple programs.
		if _, ok := b.mapKeys[m.Name]; ok {
			continue
		}

		key := mapKey(h, m)
		err := b.mapErrMap.Update(unsafe.Pointer(&key), unsafe.Pointer(z), ebpf.UpdateNoExist)
		if err != nil && !errors.Is(err, ebpf.ErrKeyExist) {
			return fmt.Errorf("failed to initialize telemetry struct for map %s", m.Name)
		}
		b.mapKeys[m.Name] = key
	}
	return nil
}

func (b *EBPFTelemetry) initializeHelperErrTelemetryMap() error {
	if b.helperErrMap == nil {
		return nil
	}

	// the `probeKeys` get added during instruction patching, so we just try to insert entries for any that don't exist
	z := new(HelperErrTelemetry)
	for p, key := range b.probeKeys {
		err := b.helperErrMap.Update(unsafe.Pointer(&key), unsafe.Pointer(z), ebpf.UpdateNoExist)
		if err != nil && !errors.Is(err, ebpf.ErrKeyExist) {
			return fmt.Errorf("failed to initialize telemetry struct for probe %s", p)
		}
	}
	return nil
}

func countRawBPFIns(ins *asm.Instruction) uint64 {
	if ins.OpCode.IsDWordLoad() {
		return 2
	}

	return 1
}

type patchSite struct {
	ins      *asm.Instruction
	callsite uint64
	insIndex int
}

func patchEBPFTelemetry(m *manager.Manager, enable bool, bpfTelemetry *EBPFTelemetry, bpfDir string, bytecode io.ReaderAt) error {
	const symbol = "telemetry_program_id_key"
	newIns := asm.Mov.Reg(asm.R1, asm.R1)
	if enable {
		newIns = asm.StoreXAdd(asm.R1, asm.R2, asm.Word)
	}
	ldDWImm := asm.LoadImmOp(asm.DWord)
	h := keyHash()

	progs, err := m.GetProgramSpecs()
	if err != nil {
		return err
	}

	// The instrumentation code relies on there being 8 bytes of space available on the stack which we can steal to
	// cache the pointer to the telemetry data.
	// In EBPF there cannot be any dynamic allocations on the stack. We utilize this guarantee to get the stack usage
	// of each function; to determine whether there are 8 bytes free for us to use. If not available, we cannot have telemetry
	// and so patch nops at our trampolines.
	//
	// The stack size of each function is obtained from the `.stack_sizes` section in the ELF file. This section is emitted
	// when the `-stack-size-section` argument is passed to llc. Details of this section are documented here:
	// - https://releases.llvm.org/9.0.0/docs/CodeGenerator.html#emitting-function-stack-size-information
	//
	// Confusingly the docs mention:
	// "The stack size values only include the space allocated in the function prologue. Functions with dynamic stack allocations are not included."
	// This seems to suggest that this option relies on an explicit prologue to get the stack space. There is no prologue present in EBPF bytecode
	// since the frame pointer is read-only.
	// According to the LLVM source code however, there is no explicit requirement of the prologue. This is deduced in two way:
	// 1. Looking at the LLVM source code which tracks BPF stack size and ensures it is less than 512 bytes, and the source code
	//    for calculating stack sizes. As can be seen in the following links, both functions track fixed size allocations on the
	//    stack to determine the size. The ebpf code checks each allocation to make sure it does not cross the 512 byte limit,
	//    and the function calculating the `StackSize` loops over all stack allocations to calculate the stack size.
	//    - BPF stack size check: https://github.com/llvm/llvm-project/blob/llvmorg-12.0.1/llvm/lib/Target/BPF/BPFRegisterInfo.cpp#L96-L102
	//    - `StackSize` calculation: https://github.com/llvm/llvm-project/blob/llvmorg-12.0.1/llvm/lib/CodeGen/PrologEpilogInserter.cpp#L1104
	//    - Emit stack_sizes section: https://github.com/llvm/llvm-project/blob/llvmorg-12.0.1/llvm/lib/CodeGen/AsmPrinter/AsmPrinter.cpp#L1137
	//
	// 2. By parsing the `.stack_sizes` sections in the object files and validating that it reports the correct stack usage.
	sizes, err := parseStackSizesSections(bytecode, progs)
	if err != nil {
		return fmt.Errorf("failed to parse '.stack_sizes' section in file: %w", err)
	}

	for fn, p := range progs {
		space := sizes.stackHas8BytesFree(fn)
		if !space {
			log.Warnf("Function %s does not have enough free stack space for instrumentation", fn)
		}

		turnOn := enable && bpfTelemetry != nil && space

		// do constant editing of programs for helper errors post-init
		ins := p.Instructions
		offsets := ins.ReferenceOffsets()
		if turnOn {
			indices := offsets[symbol]
			if len(indices) > 0 {
				for _, index := range indices {
					load := &ins[index]
					if load.OpCode != ldDWImm {
						return fmt.Errorf("symbol %v: load: found %v instead of %v", symbol, load.OpCode, ldDWImm)
					}
					key := probeKey(h, fn)
					load.Constant = int64(key)
					bpfTelemetry.probeKeys[fn] = key
				}
			}
		}

		// patch telemetry helper calls
		const retpolineArg = "retpoline_jump_addr"

		iter := ins.Iterate()
		var insCount uint64
		var telemetryPatchSite *asm.Instruction
		patchSites := make(map[eBPFPatchCall][]patchSite)
		for iter.Next() {
			ins := iter.Ins
			insCount += countRawBPFIns(ins)

			// If the compiler argument '-pg' is passed, then the compiler will instrument a `call -1` instruction
			// somewhere (see [2] below) in the beginning of the bytecode. We will patch this instruction as a trampoline to
			// -------------------------
			// jump to our instrumentation code.
			//
			// Checks:
			// [1] We cannot use the helper `IsBuiltinCall()` as below since the compiler does not correctly generate
			//     the instrumentation instruction. The loader expects the source and destination register to be set to R0
			//     along with the correct opcode, for it to be recognized as a built-in call. However, the `call -1` does
			//     not satisfy this requirement. Therefore, we use a looser check relying on the constant to be `-1` to correctly
			//     identify the patch point.
			//
			// [2] The instrumentation instruction may not be the first instruction in the bytecode.
			//     Since R1 is allowed to be clobbered, the compiler adds the instrumentation instruction after `RX = R1`, when
			//     R1, i.e. the context pointer, is used in the code. `RX` will be a callee saved register.
			//     If R1, i.e. the context pointer, is not used then the instrumented instruction will be the first instruction.
			if patchCall := ebpfEntryTrampolinePatchCall; ins.OpCode.JumpOp() == asm.Call && ins.Constant == ebpfEntryTrampolinePatchCall && iter.Offset <= maxTrampolineOffset {
				if site, ok := patchSites[patchCall]; ok {
					patchSites[patchCall] = append(patchSites[patchCall], patchSite{ins, uint64(iter.Offset), iter.Index})
				} else {
					patchSites[patchCall] = []patchSites{{ins, uint64(iter.Offset), iter.Index}}
				}
			}

			if patchCall := ebpfTelemetryPatchCall; ins.IsBuiltinCall() && ins.Constant == ebpfTelemetryPatchCall {
				if site, ok := patchSites[patchCall]; ok {
					patchSites[patchCall] = append(patchSites[patchCall], patchSite{ins, uint64(iter.Offset), iter.Index})
				} else {
					patchSites[patchCall] = []patchSites{{ins, uint64(iter.Offset), iter.Index}}
				}
			}
		}

		bpfAsset, err := netbpf.ReadEBPFTelemetryModule(bpfDir, "ebpf_instrumentation")
		if err != nil {
			return fmt.Errorf("failed to read %s bytecode file: %w", eBPFIns.filename, err)
		}
		collectionSpec, err := ebpf.LoadCollectionSpecFromReader(bpfAsset)
		if err != nil {
			return fmt.Errorf("failed to load collection spec from reader: %w", err)
		}

		var instrumentationBlock []*asm.Instruction
		for _, instr := range instrumentation {
			sites := patchSites[instr.callSite]

			blockCount := 0
			for ip, ins := range collectionSpec.Programs[instr.programName].Instructions {
				// The final instruction in the instrumentation block is `exit`, which we
				// do not want.
				if ins.OpCode.JumpOp == asm.Exit {
					break
				}
				blockCount += countRawBPFIns(ins)

				// The first instruction has associated func_info btf information. Since
				// the instrumentation is not a function, the verifier will complain that the number of
				// `func_info` objects in the BTF do not match the number of loaded programs:
				// https://elixir.bootlin.com/linux/latest/source/kernel/bpf/verifier.c#L15035
				// To workaround this we create a new instruction object and give it empty metadata.
				if ip == 0 {
					instrumentationBlock = append(
						instrumentationBlock,
						&asm.Instruction{
							OpCode:   ins.OpCode,
							Dst:      ins.Dst,
							Src:      ins.Src,
							Offset:   ins.Offset,
							Constant: ins.Constant,
						}.WithMetadata(asm.Metadata{}),
					)

					continue
				}

				instrumentationBlock = append(instrumentationBlock, ins)
			}

			// for each callsite to this instrumentation block in the original function, append a retpoline
			// to jump back to the correct ip after the instrumentation is complete
			for callType, site := range sites {
				retJumpOffset := site.callsite - (insCount + blockCount) - 1 // the final -1 because the jump offset is computed from ip-1
				if callType == ebpfEntryTrampolinePatchCall {
					instrumentationBlock = append(
						instrumentationBlock,
						asm.Instruction{OpCode: asm.OpCode(asm.JumpClass).SetJumpOp(asm.Ja), Offset: int16(retJumpOffset)},
					)

				} else {
					instrumentationBlock = append(
						instrumentationBlock,
						// if r1 == callsite goto callsite+1
						&asm.Instruction{
							OpCode:   asm.OpCode(asm.JumpClass).SetJumpOp(asm.JEq).SetSource(asm.ImmSource),
							Dst:      asm.R1,
							Offset:   -1,
							Constant: retJumpOffset,
						},
					)
				}

				blockCount++
			}

			// patch the original callsites to jump to instrumentation block
			for callType, site := range sites {
				if callType != ebpfEntryTrampolinePatchCall {
					for idx := site.insIndex; idx > site.insIndex-5 && idx >= 0; idx-- {
						load := p.Instructions[idx]
						// The load has to be there before the patch call
						if load.OpCode.JumpOp() == asm.Call {
							return fmt.Errorf("failed to discover load instruction to patch trampoline return address")
						}

						// keep looking until we find the load instruction to patch the return address in
						if !(load.OpCode == ldDWImm && load.Reference() == retpolineArg) {
							continue
						}

						load.Constant = int64(site.callsite)
					}
				}

				// The trampoline instruction is an unconditional jump to the start of this instrumentation block.
				//
				// Point the instrumented instruction to the metadata of the original instruction. If not provided,
				// some programs can end up with corrupted BTF. This happens for programs where the instrumented instruction
				// is the first instruction. In that case the loader expects this instruction to point to the `func_info` BTF.
				*(site.ins) = asm.Instruction{
					OpCode: asm.OpCode(asm.JumpClass).SetupJumpOp(asm.Ja),
					Offset: int16(insCount - site.classite - 1),
				}.WithMetadata(ins.Metadata)
			}

			insCount += blockCount
		}
	}

	return nil
}

// setupForTelemetry sets up the manager to handle eBPF telemetry.
// It will patch the instructions of all the manager probes and `undefinedProbes` provided.
// Constants are replaced for map error and helper error keys with their respective values.
// This must be called before ebpf-manager.Manager.Init/InitWithOptions
func setupForTelemetry(m *manager.Manager, options *manager.Options, bpfTelemetry *EBPFTelemetry, bpfDir string, bytecode io.ReaderAt) error {
	activateBPFTelemetry, err := ebpfTelemetrySupported()
	if err != nil {
		return err
	}
	m.InstructionPatcher = func(m *manager.Manager) error {
		return patchEBPFTelemetry(m, activateBPFTelemetry, bpfTelemetry, bpfDir, bytecode)
	}

	if activateBPFTelemetry {
		// add telemetry maps to list of maps, if not present
		if !slices.ContainsFunc(m.Maps, func(x *manager.Map) bool { return x.Name == probes.MapErrTelemetryMap }) {
			m.Maps = append(m.Maps, &manager.Map{Name: probes.MapErrTelemetryMap})
		}
		if !slices.ContainsFunc(m.Maps, func(x *manager.Map) bool { return x.Name == probes.HelperErrTelemetryMap }) {
			m.Maps = append(m.Maps, &manager.Map{Name: probes.HelperErrTelemetryMap})
		}

		if bpfTelemetry != nil {
			bpfTelemetry.setupMapEditors(options)
		}

		options.ConstantEditors = append(options.ConstantEditors, buildMapErrTelemetryConstants(m)...)
	}
	// we cannot exclude the telemetry maps because on some kernels, deadcode elimination hasn't removed references
	// if telemetry not enabled: leave key constants as zero, and deadcode elimination should reduce number of instructions

	return nil
}

func (b *EBPFTelemetry) setupMapEditors(opts *manager.Options) {
	if (b.mapErrMap != nil) || (b.helperErrMap != nil) {
		if opts.MapEditors == nil {
			opts.MapEditors = make(map[string]*ebpf.Map)
		}
	}
	// if the maps have already been loaded, setup editors to point to them
	if b.mapErrMap != nil {
		opts.MapEditors[probes.MapErrTelemetryMap] = b.mapErrMap
	}
	if b.helperErrMap != nil {
		opts.MapEditors[probes.HelperErrTelemetryMap] = b.helperErrMap
	}
	if b.bpfTelemetryMap != nil {
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
