// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package start

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/DataDog/datadog-agent/cmd/security-agent/command"
	"github.com/DataDog/datadog-agent/comp/core"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
)

func TestCommand(t *testing.T) {
	tests := []struct {
		name     string
		cliInput []string
		check    func(cliParams *cliParams, params core.BundleParams)
	}{
		{
			name:     "start",
			cliInput: []string{"start"},
			check: func(cliParams *cliParams, params core.BundleParams) {
				// Verify logger defaults
				require.Equal(t, command.LoggerName, params.LoggerName(), "logger name not matching")
			},
		},
		{
			name:     "pidfile",
			cliInput: []string{"start", "--pidfile", "/pid/file"},
			check: func(cliParams *cliParams, params core.BundleParams) {
				// Verify logger defaults
				require.Equal(t, command.LoggerName, params.LoggerName(), "logger name not matching")
				require.Equal(t, "/pid/file", cliParams.pidfilePath, "PID file path not matching")
			},
		},
	}

	for _, test := range tests {
		fxutil.TestOneShotSubcommand(t,
			Commands(newGlobalParamsTest(t)),
			test.cliInput,
			start,
			test.check,
		)
	}
}

func newGlobalParamsTest(t *testing.T) *command.GlobalParams {
	// the config needs an existing config file when initializing
	config := path.Join(t.TempDir(), "datadog.yaml")
	err := os.WriteFile(config, []byte("hostname: test"), 0644)
	require.NoError(t, err)

	return &command.GlobalParams{
		ConfigFilePaths: []string{config},
	}
}
