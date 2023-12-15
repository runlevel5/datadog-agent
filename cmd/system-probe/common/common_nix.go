// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

package common

import (
	ebpfkernel "github.com/DataDog/datadog-agent/pkg/security/ebpf/kernel"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	//nolint:revive // TODO(EBPF) Fix revive linter
	DefaultLogFile = "/var/log/datadog/system-probe.log"
)

func DisableUnsupportedKernel(isEnabled bool) bool {
	if !isEnabled {
		return false
	}
	kernelVersion, err := ebpfkernel.NewKernelVersion()
	if err != nil {
		log.Error("unable to detect the kernel version: %w", err)
	} else if !kernelVersion.IsRH7Kernel() && !kernelVersion.IsRH8Kernel() && kernelVersion.Code < ebpfkernel.Kernel4_15 {
		log.Warn("network_process is not supported for this kernel version. Disabling network_process")
		return true
	}

	return false
}
