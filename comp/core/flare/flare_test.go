// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package flare

import (
	"testing"

	"github.com/DataDog/datadog-agent/comp/aggregator/diagnosesendermanager"
	"github.com/DataDog/datadog-agent/comp/collector/collector"
	"github.com/DataDog/datadog-agent/comp/core/config"
	"github.com/DataDog/datadog-agent/comp/core/flare/types"
	"github.com/DataDog/datadog-agent/comp/core/log/logimpl"
	"github.com/DataDog/datadog-agent/comp/metadata/inventoryagent/inventoryagentimpl"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestFlareCreation(t *testing.T) {
	realProvider := func(fb types.FlareBuilder) error { return nil }

	f, _, err := newFlare(
		fxutil.Test[dependencies](
			t,
			logimpl.MockModule(),
			config.MockModule(),
			fx.Provide(func() diagnosesendermanager.Component { return nil }),
			inventoryagentimpl.MockModule(),
			fx.Provide(func() Params { return Params{} }),
			collector.NoneModule(),

			// provider a nil FlareCallback
			fx.Provide(fx.Annotate(
				func() types.FlareCallback { return nil },
				fx.ResultTags(`group:"flare"`),
			)),
			// provider a real FlareCallback
			fx.Provide(fx.Annotate(
				func() types.FlareCallback { return realProvider },
				fx.ResultTags(`group:"flare"`),
			)),
		),
	)

	require.NoError(t, err)
	assert.Len(t, f.(*flare).providers, 1)
	assert.NotNil(t, f.(*flare).providers[0])
}