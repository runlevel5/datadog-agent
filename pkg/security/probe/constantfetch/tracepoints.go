// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

// Package constantfetch holds constantfetch related files
package constantfetch

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/DataDog/datadog-agent/pkg/security/seclog"
)

func ReadTracepointFieldOffsetWithFallback(tracepoint string, field string, fallback uint64) uint64 {
	offset, err := ReadTracepointFieldOffset(tracepoint, field)
	if err != nil {
		seclog.Errorf("failed to read tracepoint format %s.%s: %v", tracepoint, field, err)
		return fallback
	}
	return offset
}

// ReadTracepointFieldOffset reads a field offset from a tracepoint format definition
func ReadTracepointFieldOffset(tracepoint string, field string) (uint64, error) {
	format, err := os.Open(fmt.Sprintf("/sys/kernel/debug/tracing/events/%s/format", tracepoint))
	if err != nil {
		return 0, err
	}
	defer format.Close()

	spaceField := fmt.Sprintf(" %s", field)

	scanner := bufio.NewScanner(format)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "field") {
			continue
		}

		parts := strings.Split(line, ";")
		var (
			name   string
			offset uint64
		)

		for _, part := range parts {
			part = strings.TrimSpace(part)

			if field_name, ok := strings.CutPrefix(part, "field:"); ok {
				name = field_name
			} else if value, ok := strings.CutPrefix(part, "offset:"); ok {
				offset, err = strconv.ParseUint(value, 10, 64)
				if err != nil {
					return 0, err
				}
			}
		}

		if strings.HasSuffix(name, spaceField) {
			return offset, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return 0, nil
}
