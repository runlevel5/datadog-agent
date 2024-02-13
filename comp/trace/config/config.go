// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package config

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"

	"go.uber.org/fx"
	"gopkg.in/yaml.v2"

	coreconfig "github.com/DataDog/datadog-agent/comp/core/config"
	apiutil "github.com/DataDog/datadog-agent/pkg/api/util"
	pkgconfig "github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/config/model"
	traceconfig "github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/datadog-agent/pkg/util/scrubber"
)

// team: agent-apm

type dependencies struct {
	fx.In
	Params Params
	Config coreconfig.Component
}

// cfg implements the Component.
type cfg struct {
	// this component is currently implementing a thin wrapper around pkg/trace/config,
	// and uses globals in that package.
	*traceconfig.AgentConfig

	// coreConfig relates to the main agent config component
	coreConfig coreconfig.Component

	// warnings are the warnings generated during setup
	warnings *pkgconfig.Warnings
}

func newConfig(deps dependencies) (Component, error) {
	tracecfg, err := setupConfig(deps, "")

	if err != nil {
		// Allow main Agent to start with missing API key
		if !(err == traceconfig.ErrMissingAPIKey && !deps.Params.FailIfAPIKeyMissing) {
			return nil, err
		}
	}

	c := cfg{
		AgentConfig: tracecfg,
		coreConfig:  deps.Config,
	}
	c.SetMaxMemCPU(pkgconfig.IsContainerized())

	return &c, nil
}

func (c *cfg) Warnings() *pkgconfig.Warnings {
	return c.warnings
}

func (c *cfg) Object() *traceconfig.AgentConfig {
	return c.AgentConfig
}

// SetHandler returns a handler to change the runtime configuration.
func (c *cfg) SetHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			httpError(w, http.StatusMethodNotAllowed, fmt.Errorf("%s method not allowed, only %s", req.Method, http.MethodPost))
			return
		}
		for key, values := range req.URL.Query() {
			if len(values) == 0 {
				continue
			}
			value := html.UnescapeString(values[len(values)-1])
			switch key {
			case "log_level":
				lvl := strings.ToLower(value)
				if lvl == "warning" {
					lvl = "warn"
				}
				if err := pkgconfig.ChangeLogLevel(lvl); err != nil {
					httpError(w, http.StatusInternalServerError, err)
					return
				}
				pkgconfig.Datadog.Set("log_level", lvl, model.SourceAgentRuntime)
				log.Infof("Switched log level to %s", lvl)
			default:
				log.Infof("Unsupported config change requested (key: %q).", key)
			}
		}
	})
}

// GetConfigHandler returns handler to get the runtime configuration.
func (c *cfg) GetConfigHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			httpError(w,
				http.StatusMethodNotAllowed,
				fmt.Errorf("%s method not allowed, only %s", req.Method, http.MethodGet),
			)
			return
		}

		if apiutil.Validate(w, req) != nil {
			return
		}

		runtimeConfig, err := yaml.Marshal(c.coreConfig.AllSettings())
		if err != nil {
			log.Errorf("Unable to marshal runtime config response: %s", err)
			body, _ := json.Marshal(map[string]string{"error": err.Error()})
			http.Error(w, string(body), http.StatusInternalServerError)
			return
		}

		scrubbed, err := scrubber.ScrubYaml(runtimeConfig)
		if err != nil {
			log.Errorf("Unable to get the core config: %s", err)
			body, _ := json.Marshal(map[string]string{"error": err.Error()})
			http.Error(w, string(body), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(scrubbed)
	})
}

// SetMaxMemCPU sets watchdog's max_memory and max_cpu_percent parameters.
// If the agent is containerized, max_memory and max_cpu_percent are disabled by default.
// Resource limits are better handled by container runtimes and orchestrators.
func (c *cfg) SetMaxMemCPU(isContainerized bool) {
	if c.coreConfig.Object().IsSet("apm_config.max_cpu_percent") {
		c.MaxCPU = c.coreConfig.Object().GetFloat64("apm_config.max_cpu_percent") / 100
	} else if isContainerized {
		log.Debug("Running in a container and apm_config.max_cpu_percent is not set, setting it to 0")
		c.MaxCPU = 0
	}

	if c.coreConfig.Object().IsSet("apm_config.max_memory") {
		c.MaxMemory = c.coreConfig.Object().GetFloat64("apm_config.max_memory")
	} else if isContainerized {
		log.Debug("Running in a container and apm_config.max_memory is not set, setting it to 0")
		c.MaxMemory = 0
	}
}
