// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package run implements 'updater run'.
package run

import (
	"context"

	"github.com/DataDog/datadog-agent/cmd/updater/command"
	"github.com/DataDog/datadog-agent/comp/core"
	"github.com/DataDog/datadog-agent/comp/core/config"
	"github.com/DataDog/datadog-agent/comp/core/log/logimpl"
	"github.com/DataDog/datadog-agent/comp/core/secrets"
	"github.com/DataDog/datadog-agent/comp/core/sysprobeconfig/sysprobeconfigimpl"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
	"go.uber.org/fx"

	"github.com/DataDog/datadog-agent/comp/updater/localapi"
	"github.com/DataDog/datadog-agent/comp/updater/localapi/localapiimpl"
	"github.com/DataDog/datadog-agent/comp/updater/rc/rcimpl"
	"github.com/DataDog/datadog-agent/comp/updater"
	"github.com/DataDog/datadog-agent/comp/updater/updater/updaterimpl"
	pkgconfig "github.com/DataDog/datadog-agent/pkg/config"

	"github.com/spf13/cobra"
)

type cliParams struct {
	command.GlobalParams
}

// Commands returns the run command
func Commands(global *command.GlobalParams) []*cobra.Command {
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Runs the updater",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFxWrapper(&cliParams{
				GlobalParams: *global,
			})
		},
	}
	return []*cobra.Command{runCmd}
}

func runFxWrapper(params *cliParams) error {
	ctx := context.Background()
	return fxutil.Run(
		fx.Provide(func() context.Context { return ctx }),
		fx.Supply(core.BundleParams{
			ConfigParams:         config.NewAgentParams(params.GlobalParams.ConfFilePath),
			SecretParams:         secrets.NewEnabledParams(),
			SysprobeConfigParams: sysprobeconfigimpl.NewParams(),
			LogParams:            logimpl.ForDaemon("UPDATER", "updater.log_file", pkgconfig.DefaultUpdaterLogFile),
		}),
		core.Bundle(),
		fx.Supply(updaterimpl.Options{
			Package: params.Package,
		}),
		rcimpl.Module(),
		updaterimpl.Module(),
		localapiimpl.Module(),
		fx.Invoke(run),
	)
}

func run(updater updater.Component, localAPI localapi.Component) error {
	updater.Start()
	defer updater.Stop()

	return localAPI.Serve()
}
