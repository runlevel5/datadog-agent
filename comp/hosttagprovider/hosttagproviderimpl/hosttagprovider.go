// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024-present Datadog, Inc.

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
		fx.Provide(newhosttagprovider),
	)
}

type hosttagproviderimpl struct {
}

func newhosttagprovider() hosttagprovider.Component {
	return &hosttagproviderimpl{}
}

func (h *hosttagproviderimpl) HostTags() []string {
	return hostMetadataUtils.GetHostTags(context.TODO(), false, config.Datadog).System
}
