// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf

package config

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	sysconfig "github.com/DataDog/datadog-agent/cmd/system-probe/config"
	aconfig "github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/eventmonitor"
	emconfig "github.com/DataDog/datadog-agent/pkg/eventmonitor/config"
	secconfig "github.com/DataDog/datadog-agent/pkg/security/config"
)

func newSystemProbeConfig(t *testing.T) {
	originalConfig := aconfig.SystemProbe
	t.Cleanup(func() {
		aconfig.SystemProbe = originalConfig
	})
	aconfig.SystemProbe = aconfig.NewConfig("system-probe", "DD", strings.NewReplacer(".", "_"))
	aconfig.InitSystemProbeConfig(aconfig.SystemProbe)
}

func TestEventStreamEnabledForSupportedKernelsLinux(t *testing.T) {
	t.Setenv("DD_SYSTEM_PROBE_EVENT_MONITORING_NETWORK_PROCESS_ENABLED", strconv.FormatBool(true))
	newSystemProbeConfig(t)

	sysProbeConfig, err := sysconfig.New("")
	require.NoError(t, err)

	emconfig := emconfig.NewConfig(sysProbeConfig)
	secconfig, err := secconfig.NewConfig()
	require.NoError(t, err)

	opts := eventmonitor.Opts{}
	evm, err := eventmonitor.NewEventMonitor(emconfig, secconfig, opts)
	require.NoError(t, err)
	require.NoError(t, evm.Init())
}
