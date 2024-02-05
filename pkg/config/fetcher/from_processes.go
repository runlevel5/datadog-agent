// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package fetcher is a collection of high level helpers to pull the configuration from other agent processes
package fetcher

import (
	"fmt"

	"github.com/DataDog/datadog-agent/comp/core/config"
	"github.com/DataDog/datadog-agent/pkg/api/util"
	settingshttp "github.com/DataDog/datadog-agent/pkg/config/settings/http"
)

// TraceAgentConfig fetch the configuration from the trace-agent process by querying its HTTPS API
func TraceAgentConfig(config config.Reader) (string, error) {
	err := util.SetAuthToken()
	if err != nil {
		return "", err
	}

	c := util.GetClient(false)

	port := config.GetInt("apm_config.debug.port")
	if port <= 0 {
		return "", fmt.Errorf("invalid apm_config.debug.port -- %d", port)
	}

	ipcAddressWithPort := fmt.Sprintf("http://127.0.0.1:%d/config", port)

	client := settingshttp.NewClient(c, ipcAddressWithPort, "trace-agent")
	return client.FullConfig()
}
