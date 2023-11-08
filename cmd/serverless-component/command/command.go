// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package command

import (
	_ "expvar"
	_ "net/http/pprof"

	"github.com/spf13/cobra"

	"github.com/DataDog/datadog-agent/cmd/serverless-component/subcommands/start"
	"github.com/DataDog/datadog-agent/pkg/cli/subcommands/version"

	// register all workloadmeta collectors
	_ "github.com/DataDog/datadog-agent/pkg/workloadmeta/collectors"
)

func MakeRootCommand(defaultLogFile string) *cobra.Command {
	// dogstatsdCmd is the root command
	logsCmd := &cobra.Command{
		Use:   "logs-agent [command]",
		Short: "Datadog logs-agent at your service.",
		Long:  `TODO`,
	}

	for _, cmd := range makeCommands(defaultLogFile) {
		logsCmd.AddCommand(cmd)
	}

	return logsCmd
}

func makeCommands(defaultLogFile string) []*cobra.Command {
	return []*cobra.Command{start.MakeCommand(defaultLogFile), version.MakeCommand("logs-agent")}
}
