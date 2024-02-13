// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package host implements the host tag Workloadmeta collector.
package host

import (
	"context"
	"time"

	"github.com/DataDog/datadog-agent/comp/core/config"
	"github.com/DataDog/datadog-agent/comp/core/workloadmeta"
	hostMetadataUtils "github.com/DataDog/datadog-agent/comp/metadata/host/hostimpl/utils"
	"github.com/DataDog/datadog-agent/pkg/util/log"

	"go.uber.org/fx"
)

const (
	collectorID   = "host"
	componentName = "host tag collector"
)

type dependencies struct {
	fx.In

	Config config.Component
}

type collector struct {
	id      string
	catalog workloadmeta.AgentType
	config  config.Component
}

// NewCollector returns a new host collector provider and an error
func NewCollector(deps dependencies) (workloadmeta.CollectorProvider, error) {
	return workloadmeta.CollectorProvider{
		Collector: &collector{
			id:      collectorID,
			catalog: workloadmeta.NodeAgent | workloadmeta.ProcessAgent,
			config:  deps.Config,
		},
	}, nil
}

// GetFxOptions returns the FX framework options for the collector
func GetFxOptions() fx.Option {
	return fx.Provide(NewCollector)
}

func (c *collector) Start(ctx context.Context, store workloadmeta.Component) error {

	duration := c.config.GetDuration("expected_tags_duration")
	if duration <= 0 {
		return nil
	}
	if duration <= time.Minute {
		log.Debugf("Tags are checked for expiration once per minute. expected_tags_duration should be at least one minute and in minute intervals.")
	}
	tags := hostMetadataUtils.GetHostTags(ctx, false, c.config).System
	log.Debugf("Adding host tags to metrics for %v : %v", duration, tags)

	store.Notify([]workloadmeta.CollectorEvent{
		{
			Type:   workloadmeta.EventTypeSet,
			Source: workloadmeta.SourceNodeOrchestrator,
			Entity: makeEntity(tags),
		},
	})

	go func() {
		timer := time.NewTimer(duration)
		select {
		case <-ctx.Done():
			return

		case <-timer.C:
			store.Notify([]workloadmeta.CollectorEvent{
				{
					Type:   workloadmeta.EventTypeSet,
					Source: workloadmeta.SourceNodeOrchestrator,
					Entity: makeEntity([]string{}),
				},
			})
		}
	}()
	return nil
}

func makeEntity(tags []string) *workloadmeta.HostTags {
	return &workloadmeta.HostTags{
		EntityID: workloadmeta.EntityID{
			Kind: workloadmeta.KindHost,
			ID:   "host",
		},
		HostTags: tags,
	}
}

func (c *collector) Pull(_ context.Context) error {
	return nil
}

func (c *collector) GetID() string {
	return c.id
}

func (c *collector) GetTargetCatalog() workloadmeta.AgentType {
	return c.catalog
}
