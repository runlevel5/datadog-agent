// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package agent

import (
	"context"

	"go.uber.org/fx"

	"github.com/DataDog/datadog-agent/pkg/trace/watchdog"

	"github.com/DataDog/datadog-go/v5/statsd"
)

func setupShutdown(ctx context.Context, shutdowner fx.Shutdowner, statsd statsd.ClientInterface) {
	// Handle stops properly
	go handleSignal(shutdowner, statsd)

	// Support context cancellation approach (required for Windows service, as it doesn't use signals)
	go func() {
		defer watchdog.LogOnPanic(statsd)
		<-ctx.Done()
		_ = shutdowner.Shutdown()
	}()
}
