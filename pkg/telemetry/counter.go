// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build !serverless
// +build !serverless

package telemetry

import (
	"github.com/DataDog/datadog-agent/comp/core/telemetry"
	telemetryComponent "github.com/DataDog/datadog-agent/comp/core/telemetry/telemetryimpl"
)

// Counter tracks how many times something is happening.
type Counter interface {
	telemetry.Counter
}

// NewCounter creates a Counter with default options for telemetry purpose.
// Current implementation used: Prometheus Counter
func NewCounter(subsystem, name string, tags []string, help string) Counter {
	return NewCounterWithOpts(subsystem, name, tags, help, DefaultOptions)
}

// NewCounterWithOpts creates a Counter with the given options for telemetry purpose.
// See NewCounter()
func NewCounterWithOpts(subsystem, name string, tags []string, help string, opts Options) Counter {
	return telemetryComponent.GetCompatComponent().NewCounterWithOpts(subsystem, name, tags, help, telemetry.Options(opts))
}
