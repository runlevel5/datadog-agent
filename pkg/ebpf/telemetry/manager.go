// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf

package telemetry

import (
	"github.com/DataDog/datadog-agent/pkg/ebpf/manager"
)

type telemetryManagerModifier struct {
}

func (t *telemetryManagerModifier) Name() string {
	return "telemetry"
}

func (t *telemetryManagerModifier) BeforeInit(m *manager.Manager, opts *manager.Options) error {
	return setupForTelemetry(m.Manager, &opts.Options)
}

func (t *telemetryManagerModifier) AfterInit(m *manager.Manager, _ *manager.Options) error {
	if bpfTelemetry != nil {
		return bpfTelemetry.populateMapsWithKeys(m.Manager)
	}
	return nil
}

func (t *telemetryManagerModifier) OnStop(_ *manager.Manager) error {
	return nil
}

func init() {
	manager.RegisterModifier(&telemetryManagerModifier{})
}
