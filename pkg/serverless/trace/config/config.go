// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package config

import (
	"fmt"

	compcorecfg "github.com/DataDog/datadog-agent/comp/core/config"
	comptracecfg "github.com/DataDog/datadog-agent/comp/trace/config"
	"github.com/DataDog/datadog-agent/pkg/trace/config"
)

// LoadConfig is implementing Load to retrieve the config
type LoadConfig struct {
	Path string
}

// Load loads the config from a file path
func (l *LoadConfig) Load() (*config.AgentConfig, error) {
	c, err := compcorecfg.NewServerlessConfig(l.Path)
	if err != nil {
		return nil, err
	} else if c == nil {
		return nil, fmt.Errorf("No error, but no configuration component was produced - bailing out")
	}
	return comptracecfg.LoadConfigFile(l.Path, c)
}
