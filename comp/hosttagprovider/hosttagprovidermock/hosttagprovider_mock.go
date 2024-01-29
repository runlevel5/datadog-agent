// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024-present Datadog, Inc.

//go:build test

// Package hosttagprovidermock is an implementation of the hosttagprovider.Component interface.
package hosttagprovidermock

import (
	"go.uber.org/fx"

	"github.com/DataDog/datadog-agent/comp/hosttagprovider"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
)

// MockModule defines the fx options for the mock component.
func MockModule() fxutil.Module {
	return fxutil.Component(
		fx.Provide(newMockHostTagProvider),
	)
}

type mockHostTagProviderImpl struct{}

func newMockHostTagProvider() hosttagprovider.Component {
	return &mockHostTagProviderImpl{}
}

func (h *mockHostTagProviderImpl) HostTags() []string {
	return []string{}
}
