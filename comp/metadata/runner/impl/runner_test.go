// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package runnerimpl implements a component to generate metadata payload at the right interval.
package impl

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DataDog/datadog-agent/comp/core/config"
	"github.com/DataDog/datadog-agent/comp/core/log/logimpl"
	compdef "github.com/DataDog/datadog-agent/comp/def"
	"github.com/DataDog/datadog-agent/comp/metadata/runner/def"
	"github.com/DataDog/datadog-agent/pkg/config/model"
)

func TestRunner(t *testing.T) {
	wg := sync.WaitGroup{}

	provider := func(context.Context) time.Duration {
		wg.Done()
		return 1 * time.Minute // Long timeout to block
	}

	wg.Add(1)

	lc := compdef.NewLifecycle()

	comp := NewRunner(
		Requires{
			Log:       logimpl.NewMock(t),
			Config:    config.NewMock(t),
			Lc:        lc,
			Providers: []def.MetadataProvider{provider},
		},
	)
	assert.NotNil(t, comp)

	hooks := lc.Hooks()
	require.Len(t, hooks, 1)

	// either the provider call wg.Done() or the test will fail as a timeout
	hooks[0].OnStart(context.Background())
	wg.Wait()
	assert.NoError(t, hooks[0].OnStop(context.Background()))
}

func TestDisabledMetadataCollection(t *testing.T) {
	conf := config.NewMock(t)
	conf.Set("enable_metadata_collection", false, model.SourceUnknown)

	lc := compdef.NewLifecycle()
	comp := NewRunner(
		Requires{
			Log:       logimpl.NewMock(t),
			Config:    conf,
			Lc:        lc,
			Providers: []def.MetadataProvider{},
		},
	)
	assert.NotNil(t, comp)

	hooks := lc.Hooks()
	require.Len(t, hooks, 0)
}
