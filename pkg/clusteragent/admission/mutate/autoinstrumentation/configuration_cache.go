// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver

package autoinstrumentation

import (
	"fmt"
	"sync"

	"github.com/DataDog/datadog-agent/pkg/remoteconfig/state"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

type instrumentationConfiguration struct {
	enabled            bool
	enabledNamespaces  []string
	disabledNamespaces []string
}

type enablementConfig struct {
	configID          string
	rcVersion         int
	rcAction          string
	env               *string
	enabled           *bool
	enabledNamespaces *[]string
}

type instrumentationConfigurationCache struct {
	localConfiguration     *instrumentationConfiguration
	currentConfiguration   *instrumentationConfiguration
	clusterName            string
	namespaceToConfigIDMap map[string]string // maps the namespace with enabled instrumentation to Remote Enablement rule
	namespaceToEnvMap      map[string]string

	mu               sync.RWMutex
	orderedRevisions []int64
	enabledRevisions map[int64]enablementConfig
}

func newInstrumentationConfigurationCache(
	localEnabled bool,
	localEnabledNamespaces []string,
	localDisabledNamespaces []string,
	clusterName string,
) *instrumentationConfigurationCache {
	localConfig := newInstrumentationConfiguration(localEnabled, localEnabledNamespaces, localDisabledNamespaces)
	currentConfig := newInstrumentationConfiguration(localEnabled, localEnabledNamespaces, localDisabledNamespaces)

	nsToRules := make(map[string]string)
	if localEnabled {
		for _, ns := range localEnabledNamespaces {
			nsToRules[ns] = "local"
		}
	}

	return &instrumentationConfigurationCache{
		localConfiguration:     localConfig,
		currentConfiguration:   currentConfig,
		clusterName:            clusterName,
		namespaceToConfigIDMap: nsToRules,
		namespaceToEnvMap:      make(map[string]string),

		orderedRevisions: make([]int64, 0),
		enabledRevisions: map[int64]enablementConfig{},
	}
}

func (c *instrumentationConfigurationCache) getConfigID(ns string) string {
	if rcID, ok := c.namespaceToConfigIDMap[ns]; ok {
		return rcID
	} else if rcIDCluster, ok := c.namespaceToConfigIDMap["cluster"]; ok {
		return rcIDCluster
	}
	return ""
}

func (c *instrumentationConfigurationCache) getEnv(ns string) string {
	if env, ok := c.namespaceToEnvMap[ns]; ok {
		return env
	} else if env, ok := c.namespaceToEnvMap["cluster"]; ok {
		return env
	}
	return ""
}

func (c *instrumentationConfigurationCache) update(req Request) Response {

	k8sClusterTargets := req.K8sTargetV2.ClusterTargets
	var resp Response

	switch req.Action {
	case EnableConfig:
		for _, target := range k8sClusterTargets {
			clusterName := target.ClusterName

			if c.clusterName != clusterName {
				continue
			}
			log.Infof("Current configuration: %v, %v, %v",
				c.currentConfiguration.enabled, c.currentConfiguration.enabledNamespaces, c.currentConfiguration.disabledNamespaces)
			newEnabled := target.Enabled
			newEnabledNamespaces := target.EnabledNamespaces

			c.mu.Lock()
			defer c.mu.Unlock()
			var env string
			if req.LibConfig.Env == nil {
				env = ""
			} else {
				env = *req.LibConfig.Env
			}
			resp = c.updateConfiguration(*newEnabled, newEnabledNamespaces, req.ID, int(req.RcVersion), env)
			log.Infof("Updated configuration: %v, %v, %v",
				c.currentConfiguration.enabled, c.currentConfiguration.enabledNamespaces, c.currentConfiguration.disabledNamespaces)

			c.orderedRevisions = append(c.orderedRevisions, req.Revision)
			c.enabledRevisions[req.Revision] = enablementConfig{
				configID:          req.ID,
				rcVersion:         int(req.RcVersion),
				rcAction:          string(req.Action),
				env:               req.LibConfig.Env,
				enabled:           target.Enabled,
				enabledNamespaces: target.EnabledNamespaces,
			}
		}
	default:
		log.Errorf("unknown action %q", req.Action)
	}

	return resp
}

func (c *instrumentationConfigurationCache) delete(rcConfigID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, rev := range c.orderedRevisions {
		confID, ok := c.enabledRevisions[rev]
		if !ok {
			log.Error("Revision was not found")
		}
		if confID.configID == rcConfigID {
			delete(c.enabledRevisions, rev)
			c.orderedRevisions = append(c.orderedRevisions[:i], c.orderedRevisions[i+1:]...)
			if confID.enabledNamespaces != nil && len(*confID.enabledNamespaces) > 0 {
				for _, ns := range *confID.enabledNamespaces {
					delete(c.namespaceToConfigIDMap, ns)
					delete(c.namespaceToEnvMap, ns)
				}
				log.Infof("LILIYAB11")
			} else {
				delete(c.namespaceToConfigIDMap, "cluster")
				delete(c.namespaceToEnvMap, "cluster")
				log.Infof("LILIYAB22")
			}
			break
		}
	}
	c.resetConfiguration()
	return nil
}

func (c *instrumentationConfigurationCache) resetConfiguration() {
	log.Infof("Reseting current configuration %+v:", c.currentConfiguration)
	c.currentConfiguration.enabled = c.localConfiguration.enabled
	c.currentConfiguration.enabledNamespaces = c.localConfiguration.enabledNamespaces
	c.currentConfiguration.disabledNamespaces = c.localConfiguration.disabledNamespaces
	log.Infof("New base configuration %+v:", c.currentConfiguration)
	for _, rev := range c.orderedRevisions {
		conf := c.enabledRevisions[rev]
		var env string
		if conf.env == nil {
			env = ""
		} else {
			env = *conf.env
		}
		log.Infof("Updating with  %s, %v, %v:", conf.configID, conf.enabled, conf.enabledNamespaces)
		c.updateConfiguration(*conf.enabled, conf.enabledNamespaces, conf.configID, conf.rcVersion, env)
	}
}

func (c *instrumentationConfigurationCache) updateConfiguration(enabled bool, enabledNamespaces *[]string, rcID string, rcVersion int, env string) Response {
	log.Infof("Updating current APM Instrumentation configuration. Old APM Instrumentation configuration [enabled=%t enabledNamespaces=%v disabledNamespaces=%v]",
		c.currentConfiguration.enabled,
		c.currentConfiguration.enabledNamespaces,
		c.currentConfiguration.disabledNamespaces,
	)
	resp := Response{
		ID:        rcID,
		RcVersion: uint64(rcVersion),
		Status:    state.ApplyStatus{State: state.ApplyStateAcknowledged},
	}

	if c.currentConfiguration.enabled && !enabled {
		log.Errorf("Remote Enablement: disabling APM instrumentation is not supported")
		resp.Status.State = state.ApplyStateError
		resp.Status.Error = "Disabling APM instrumentation is not supported"
		return resp
	}

	if c.currentConfiguration.enabled {
		if len(c.currentConfiguration.enabledNamespaces) == 0 &&
			len(c.currentConfiguration.disabledNamespaces) == 0 &&
			enabledNamespaces != nil && len(*enabledNamespaces) > 0 {
			// current configuration - APM Instrumentation enabled in the whole cluster
			// remote configuration - APM Instrumentation enabled in specific namespaces
			// result - error
			log.Errorf("Remote Enablement: APM Insrtumentation is enabled in the whole cluster via agent configuration")
			resp.Status.State = state.ApplyStateError
			resp.Status.Error = "Remote Enablement: APM Insrtumentation is enabled in the whole cluster via agent configuration"
			return resp
		} else if len(c.currentConfiguration.enabledNamespaces) > 0 && (enabledNamespaces == nil || len(*enabledNamespaces) == 0) {
			// current configuration - APM Instrumentation enabled in specific namespaces
			// remote configuration - APM Instrumentation enabled in the whole cluster
			// result - enable APM instrumentation in the whole cluster
			c.currentConfiguration.enabledNamespaces = []string{}
			c.namespaceToConfigIDMap["cluster"] = fmt.Sprintf("%s-%d", rcID, rcVersion)
			if env != "" {
				c.namespaceToEnvMap["cluster"] = env
			}
		} else if len(c.currentConfiguration.enabledNamespaces) > 0 {
			// current configuration - APM Instrumentation enabled in specific namespaces
			// remote configuration - APM Instrumentation enabled in specific namespaces
			// result - enable APM instrumentation in namespaces specified by current + remote configuration
			alreadyEnabledNamespaces := []string{}
			for _, ns := range *enabledNamespaces {
				if _, ok := c.namespaceToConfigIDMap[ns]; ok {
					alreadyEnabledNamespaces = append(alreadyEnabledNamespaces, ns)
				} else {
					c.currentConfiguration.enabledNamespaces = append(c.currentConfiguration.enabledNamespaces, ns)
					c.namespaceToConfigIDMap[ns] = fmt.Sprintf("%s-%d", rcID, rcVersion)
					if env != "" {
						c.namespaceToEnvMap[ns] = env
					}
				}
			}
			if len(alreadyEnabledNamespaces) > 0 {
				resp.Status.State = state.ApplyStateError
				resp.Status.Error = fmt.Sprintf("Remote Enablement: failing instrumentation because APM is already enabled in namespaces %v", alreadyEnabledNamespaces)
				return resp
			}
		} else if len(c.currentConfiguration.disabledNamespaces) > 0 && (enabledNamespaces == nil || len(*enabledNamespaces) == 0) {
			// current configuration - APM Instrumentation disabled in specific namespaces
			// remote configuration - APM Instrumentation enabled in the whole cluster
			// result - enable APM instrumentation in the whole cluster
			c.currentConfiguration.disabledNamespaces = []string{}
			c.namespaceToConfigIDMap["cluster"] = fmt.Sprintf("%s-%d", rcID, rcVersion)
			if env != "" {
				c.namespaceToEnvMap["cluster"] = env
			}
		} else if len(c.currentConfiguration.disabledNamespaces) > 0 {
			// current configuration - APM Instrumentation disabled in specific namespaces
			// remote configuration - APM Instrumentation enabled in specific namespaces
			// result - enable APM instrumentation in disabled namespaces
			disabledNsMap := make(map[string]struct{})
			for _, ns := range c.currentConfiguration.disabledNamespaces {
				disabledNsMap[ns] = struct{}{}
			}
			for _, ns := range *enabledNamespaces {
				delete(disabledNsMap, ns)
				c.namespaceToConfigIDMap[ns] = fmt.Sprintf("%s-%d", rcID, rcVersion)
				if env != "" {
					c.namespaceToEnvMap[ns] = env
				}
			}
			disabledNs := make([]string, 0, len(disabledNsMap))
			for k := range disabledNsMap {
				disabledNs = append(disabledNs, k)
			}
			c.currentConfiguration.disabledNamespaces = disabledNs
		}
	} else {
		if enabled {
			c.currentConfiguration.enabled = enabled
			if enabledNamespaces != nil && len(*enabledNamespaces) > 0 {
				for _, ns := range *enabledNamespaces {
					c.currentConfiguration.enabledNamespaces = append(c.currentConfiguration.enabledNamespaces, ns)
					c.namespaceToConfigIDMap[ns] = fmt.Sprintf("%s-%d", rcID, rcVersion)
					if env != "" {
						c.namespaceToEnvMap[ns] = env
					}
				}
			} else {
				c.namespaceToConfigIDMap["cluster"] = fmt.Sprintf("%s-%d", rcID, rcVersion)
				if env != "" {
					c.namespaceToEnvMap["cluster"] = env
				}
			}
		} else {
			log.Errorf("Noop: APM Instrumentation is off")
			resp.Status.State = state.ApplyStateError
			resp.Status.Error = "Noop: APM Instrumentation is off"
			return resp
		}
	}

	log.Infof("New APM Instrumentation configuration [enabled=%t enabledNamespaces=%v disabledNamespaces=%v]",
		c.currentConfiguration.enabled,
		c.currentConfiguration.enabledNamespaces,
		c.currentConfiguration.disabledNamespaces,
	)
	return resp
}

func newInstrumentationConfiguration(
	enabled bool,
	enabledNamespaces []string,
	disabledNamespaces []string,
) *instrumentationConfiguration {
	return &instrumentationConfiguration{
		enabled:            enabled,
		enabledNamespaces:  enabledNamespaces,
		disabledNamespaces: disabledNamespaces,
	}
}
