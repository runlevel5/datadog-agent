package telemetry

import (
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/DataDog/datadog-agent/pkg/util/native"
	"github.com/cilium/ebpf"
)

const lowerOrder7Mask = 0x7f
const higherOrderBit = 0x80

// https://en.wikipedia.org/wiki/LEB128
func decodeULEB128(in io.Reader) (uint64, error) {
	var result, shift uint64
	for {
		b := []byte{0}
		if _, err := in.Read(b); err != nil {
			return 0, fmt.Errorf("failed to decode ULEB128: %w", err)
		}
		result |= uint64(b[0]&lowerOrder7Mask) << shift
		if b[0]&higherOrderBit == 0 {
			return result, nil
		}
		shift += 7
	}
}

type stackSizes map[string]uint64

type symbolKey struct {
	index int
	value uint64
}

func parseStackSizesSections(bytecode io.ReaderAt, programSpecs map[string]*ebpf.ProgramSpec) (stackSizes, error) {
	objFile, err := elf.NewFile(bytecode)
	if err != nil {
		return nil, fmt.Errorf("failed to open bytecode: %w", err)
	}

	syms, err := objFile.Symbols()
	if err != nil {
		return nil, fmt.Errorf("failed to read elf symbols: %w", err)
	}

	symbols := make(map[symbolKey]string, len(programSpecs))
	for _, sym := range syms {
		if _, ok := programSpecs[sym.Name]; ok {
			fmt.Printf("Insert symbol %v -> %s. Size: %d\n", symbolKey{int(sym.Section), sym.Value}, sym.Name, sym.Size)
			symbols[symbolKey{int(sym.Section), sym.Value}] = sym.Name
		}
	}

	sizes := make(stackSizes, len(programSpecs))
	for _, section := range objFile.Sections {
		if section.Name != ".stack_sizes" {
			continue
		}

		sectionReader := section.Open()
		for {
			var s uint64
			if err := binary.Read(sectionReader, native.Endian, &s); err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("error reading '.stack_sizes' section: %w", err)
			}

			size, err := decodeULEB128(sectionReader)
			if err != nil {
				return nil, err
			}

			if _, ok := symbols[symbolKey{int(section.Link), s}]; ok {
				name := symbols[symbolKey{int(section.Link), s}]
				fmt.Printf("Stack size for program %s is %d\n", name, size)
				sizes[name] = size
			}
		}
	}

	if len(sizes) != len(programSpecs) {
		for _, p := range programSpecs {
			if _, ok := sizes[p.Name]; !ok {
				fmt.Printf("%s program not found in sizes\n", p.Name)
			}
		}
		fmt.Println(symbols)
		return nil, errors.New("failed to find stack sizes of all programs")
	}

	return sizes, nil
}
