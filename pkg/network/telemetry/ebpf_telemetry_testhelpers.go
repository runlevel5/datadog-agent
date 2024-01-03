// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf && test

package telemetry

import (
	"fmt"
	"unsafe"

	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// GetHelpersTelemetry returns a map of error telemetry for each ebpf program
func (b *EBPFTelemetry) GetHelpersTelemetry() map[string]interface{} {
	helperTelemMap := make(map[string]interface{})
	if b.bpfTelemetryMap == nil {
		return helperTelemMap
	}

	var val InstrumentationBlob
	key := 0
	err := b.bpfTelemetryMap.Lookup(unsafe.Pointer(&key), unsafe.Pointer(&key))
	if err != nil {
		log.Debugf("failed to get instrumentation blob")
		return helperTelemMap
	}

	fmt.Printf("Active value: %d\n", val.Telemetry_active)
	for probeName, probeIndex := range b.probeKeys {
		t := make(map[string]interface{})
		for indx, helperName := range helperNames {
			base := maxErrno * indx
			if count := getErrCount(val.Helper_err_telemetry[probeIndex].Count[base : base+maxErrno]); len(count) > 0 {
				t[helperName] = count
			}
		}
		if len(t) > 0 {
			helperTelemMap[probeName] = t
		}
	}

	return helperTelemMap
}

// GetMapsTelemetry returns a map of error telemetry for each ebpf map
func (b *EBPFTelemetry) GetMapsTelemetry() map[string]interface{} {
	t := make(map[string]interface{})
	if b.bpfTelemetryMap == nil {
		return t
	}
	var val InstrumentationBlob
	key := 0
	err := b.bpfTelemetryMap.Lookup(unsafe.Pointer(&key), unsafe.Pointer(&val))
	if err != nil {
		log.Debugf("failed to get instrumentation blob")
		return t
	}

	for mapName, mapIndx := range b.mapKeys {
		if count := getErrCount(val.Map_err_telemetry[mapIndx].Count[:]); len(count) > 0 {
			t[mapName] = count
		}
	}

	return t
}
