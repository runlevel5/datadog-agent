// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

// Package hostname exposes hostname.Get() as a component.
package hostname

import (
	"context"
)

// team: agent-shared-components

const (
	ConfigProvider  = "configuration"
	FargateProvider = "fargate"
)

// Data contains hostname and the hostname provider
type Data struct {
	Hostname string
	Provider string
}

// FromConfiguration returns true if the hostname was found through the configuration file
func (h Data) FromConfiguration() bool {
	return h.Provider == ConfigProvider
}

// FromFargate returns true if the hostname was found through Fargate
func (h Data) FromFargate() bool {
	return h.Provider == FargateProvider
}

// Component is the component type.
type Component interface {
	// Get returns the host name for the agent.
	Get(context.Context) (string, error)
	// GetWithProvider returns the hostname for the Agent and the provider that was use to retrieve it.
	GetWithProvider(ctx context.Context) (Data, error)
	// GetSafe is Get(), but it returns 'unknown host' if anything goes wrong.
	GetSafe(context.Context) string
}
