// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package runner implements a component to generate metadata payload at the right interval.
package def

import (
	"context"
	"time"

	compdef "github.com/DataDog/datadog-agent/comp/def"
)

// team: agent-shared-components

// Component is the component type.
type Component interface{}

// MetadataProvider is the provider for metadata
type MetadataProvider func(context.Context) time.Duration

// Provider represents the callback from a metada provider. This is returned by 'NewProvider' helper.
type Provider struct {
	compdef.Out

	Callback MetadataProvider `group:"metadata_provider"`
}

// NewProvider registers a new metadata provider by adding a callback to the runner.
func NewProvider(callback MetadataProvider) Provider {
	return Provider{
		Callback: callback,
	}
}
