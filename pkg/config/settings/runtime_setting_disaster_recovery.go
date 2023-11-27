// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package settings

import (
	"fmt"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/config/model"
)

// DisasterRecoveryRuntimeSetting wraps operations to start logging aggregator payload at runtime.
type DisasterRecoveryRuntimeSetting struct {
	ConfigKey string
}

// NewDisasterRecoveryRuntimeSetting returns a new DisasterRecoveryRuntimeSetting
func NewDisasterRecoveryRuntimeSetting() *DisasterRecoveryRuntimeSetting {
	return &DisasterRecoveryRuntimeSetting{ConfigKey: "disaster_recovery_enabled"}
}

// Description returns the runtime setting's description
func (l *DisasterRecoveryRuntimeSetting) Description() string {
	return "Enables disaster recovery mode at runtime."
}

// Hidden returns whether or not this setting is hidden from the list of runtime settings
func (l *DisasterRecoveryRuntimeSetting) Hidden() bool {
	return true // Should be false
}

// Name returns the name of the runtime setting
func (l *DisasterRecoveryRuntimeSetting) Name() string {
	return l.ConfigKey
}

// Get returns the current value of the runtime setting
func (l *DisasterRecoveryRuntimeSetting) Get() (interface{}, error) {
	return config.Datadog.GetBool("disaster_recovery.enabled"), nil
}

// Set changes the value of the runtime setting
func (l *DisasterRecoveryRuntimeSetting) Set(v interface{}, source model.Source) error {
	var newValue bool
	var err error

	if newValue, err = GetBool(v); err != nil {
		return fmt.Errorf("DisasterRecoveryRuntimeSetting: %v", err)
	}

	config.Datadog.Set("disaster_recovery.enabled", newValue, source)
	return nil
}
