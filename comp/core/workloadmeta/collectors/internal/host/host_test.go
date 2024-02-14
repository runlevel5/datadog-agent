// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package host implements the host tag Workloadmeta collector.
package host

import (
	"context"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"

	"github.com/DataDog/datadog-agent/comp/core"
	"github.com/DataDog/datadog-agent/comp/core/config"
	"github.com/DataDog/datadog-agent/comp/core/workloadmeta"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"

	"go.uber.org/fx"
)

type testDeps struct {
	fx.In

	Config config.Component
	Wml    workloadmeta.Mock
}

func TestHostCollector(t *testing.T) {
	expectedTags := []string{"tag1:value1", "tag2", "tag3"}

	overrides := map[string]interface{}{
		"tags":                   expectedTags,
		"expected_tags_duration": "10m",
	}

	deps := fxutil.Test[testDeps](t, fx.Options(
		fx.Replace(config.MockParams{Overrides: overrides}),
		core.MockBundle(),
		fx.Supply(workloadmeta.NewParams()),
		fx.Supply(context.Background()),
		workloadmeta.MockModule(),
	))

	mockClock := clock.NewMock()
	c := collector{
		config: deps.Config,
		clock:  mockClock,
	}

	c.Start(context.TODO(), deps.Wml)

	assert.Equal(t, len(deps.Wml.GetNotifiedEvents()), 1)
	assertTags(t, deps.Wml.GetNotifiedEvents()[0].Entity, expectedTags)

	// Advance the clock by 11 minutes so prune will expire the tags.
	mockClock.Add(11 * time.Minute)

	assert.Eventually(t, func() bool {
		return len(deps.Wml.GetNotifiedEvents()) == 2
	}, 2*time.Second, 100*time.Millisecond)

	assertTags(t, deps.Wml.GetNotifiedEvents()[1].Entity, []string{})
}

func assertTags(t *testing.T, entity workloadmeta.Entity, expectedTags []string) {
	e := entity.(*workloadmeta.HostTags)
	assert.ElementsMatch(t, e.HostTags, expectedTags)
}
