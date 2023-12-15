// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package common

const (
	//nolint:revive // TODO(EBPF) Fix revive linter
	DefaultLogFile = "c:\\programdata\\datadog\\logs\\system-probe.log"
)

// Returns true if network_process needs to be disabled due to unsupported kernel version
func DisablePESUnsupportedKernel(isEnabled bool) bool {
	return false
}
