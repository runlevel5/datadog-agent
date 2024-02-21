// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build !otlp

// Package collector implements the OTLP Collector component for non-OTLP builds.
package collector

import (
	"go.uber.org/fx"

	"github.com/DataDog/datadog-agent/comp/core/status"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
)

// Component represents the no-op Component interface.
type Component interface {
	Start() error
	Stop()
}

// Module specifies the fx module for non-OTLP builds.
func Module() fxutil.Module {
	return fxutil.Component(
		fx.Provide(newPipeline))
}

type provides struct {
	fx.Out

	Comp           Component
	StatusProvider status.InformationProvider
}

func newPipeline() provides {
	return provides{
		Comp:           noOpComp{},
		StatusProvider: status.NoopInformationProvider(),
	}
}

type noOpComp struct{}

// Start is a no-op.
func (noOpComp) Start() error { return nil }

// Stop is a no-op.
func (noOpComp) Stop() {}
