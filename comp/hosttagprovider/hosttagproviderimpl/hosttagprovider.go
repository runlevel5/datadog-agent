// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024-present Datadog, Inc.

// Package hosttagproviderimpl is an implementation of the hosttagprovider.Component interface.
package hosttagproviderimpl

import (
	"context"

	"go.uber.org/fx"

	"github.com/DataDog/datadog-agent/comp/hosttagprovider"
	hostMetadataUtils "github.com/DataDog/datadog-agent/comp/metadata/host/hostimpl/utils"
	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
)

// Module defines the fx options for this component.
func Module() fxutil.Module {
	return fxutil.Component(
		fx.Provide(newHostTagProviderImpl),
	)
}

type hostTagProviderImpl struct{}

func newHostTagProviderImpl() hosttagprovider.Component {
	return &hostTagProviderImpl{}
}

func (h *hostTagProviderImpl) HostTags() []string {
	return hostMetadataUtils.GetHostTags(context.TODO(), false, config.Datadog).System
}
