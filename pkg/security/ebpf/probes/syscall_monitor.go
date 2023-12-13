// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

// Package probes holds probes related files
package probes

import (
	manager "github.com/DataDog/ebpf-manager"
	"github.com/cilium/ebpf"

	"github.com/DataDog/datadog-agent/pkg/security/secl/model/syscalls"
)

// syscallMonitorProbes holds the list of probes used to track syscall events

func getSyscallMonitorProbes() []*manager.Probe {
	return []*manager.Probe{
		{
			ProbeIdentificationPair: manager.ProbeIdentificationPair{
				UID:          SecurityAgentUID,
				EBPFFuncName: "sys_enter",
			},
		},
	}
}

func getSyscallTableMap() *manager.Map {
	m := &manager.Map{
		Name: "syscall_table",
	}

	// initialize the content of the map with the syscalls ID of the current architecture
	type syscallTableKey struct {
		id  uint64
		key uint64
	}

	m.Contents = []ebpf.MapKV{
		{
			Key: syscallTableKey{
				id:  uint64(syscalls.SysExit),
				key: 1,
			},
			Value: uint8(1),
		},
		{
			Key: syscallTableKey{
				id:  uint64(syscalls.SysExitGroup),
				key: 1,
			},
			Value: uint8(1),
		},
		{
			Key: syscallTableKey{
				id:  uint64(syscalls.SysExecve),
				key: 2,
			},
			Value: uint8(1),
		},
		{
			Key: syscallTableKey{
				id:  uint64(syscalls.SysExecveat),
				key: 2,
			},
			Value: uint8(1),
		},
	}
	return m
}
