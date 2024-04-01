// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver

package workload

import (
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/clock"

	"github.com/DataDog/datadog-agent/pkg/clusteragent/autoscaling/workload/model"
	"github.com/DataDog/datadog-agent/pkg/config/remote/data"
	"github.com/DataDog/datadog-agent/pkg/remoteconfig/state"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/datadog-agent/pkg/util/pointer"
)

const (
	configRetrieverStoreID string = "cr"
)

// Subinterface of rcclient.Component to allow mocking
type rcClient interface {
	Subscribe(product string, fn func(update map[string]state.RawConfig, applyStateCallback func(string, state.ApplyStatus)))
}

// configRetriever is responsible for retrieving remote objects (Autoscaling .Spec and values)
type configRetriever struct {
	store    *store
	isLeader func() bool
	clock    clock.Clock
}

func newConfigRetriever(store *store, isLeader func() bool, rcClient rcClient) (*configRetriever, error) {
	cr := &configRetriever{
		store:    store,
		isLeader: isLeader,
		clock:    clock.RealClock{},
	}

	rcClient.Subscribe(data.ProductContainerAutoscalingSettings, func(update map[string]state.RawConfig, applyStateCallback func(string, state.ApplyStatus)) {
		cr.autoscalerUpdateCallback(cr.clock.Now(), update, applyStateCallback, cr.processAutoscalingSettings)
	})

	rcClient.Subscribe(data.ProductContainerAutoscalingValues, func(update map[string]state.RawConfig, applyStateCallback func(string, state.ApplyStatus)) {
		cr.autoscalerUpdateCallback(cr.clock.Now(), update, applyStateCallback, cr.processAutoscalingValues)
	})
	return cr, nil
}

func (cr *configRetriever) autoscalerUpdateCallback(timestamp time.Time, update map[string]state.RawConfig, applyStateCallback func(string, state.ApplyStatus), process func(time.Time, string, state.RawConfig) error) {
	// configKey and configValues.Metadata.{ID,Name} are opaque identifiers, we don't use them
	// we're just keeping configKey for logging purposes
	for configKey, rawConfig := range update {
		if !cr.isLeader() {
			applyStateCallback(configKey, state.ApplyStatus{
				State: state.ApplyStateUnacknowledged,
				Error: "",
			})
			continue
		}

		err := process(timestamp, configKey, rawConfig)
		if err != nil {
			applyStateCallback(configKey, state.ApplyStatus{
				State: state.ApplyStateError,
				Error: err.Error(),
			})
		} else {
			applyStateCallback(configKey, state.ApplyStatus{
				State: state.ApplyStateAcknowledged,
				Error: "",
			})
		}
	}
}

func (cr *configRetriever) processAutoscalingSettings(receivedTimestamp time.Time, configKey string, rawConfig state.RawConfig) error {
	settingsList := &model.AutoscalingSettingsList{}
	err := json.Unmarshal(rawConfig.Config, &settingsList)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config id:%s, version: %d, config key: %s, err: %v", rawConfig.Metadata.ID, rawConfig.Metadata.Version, configKey, err)
	}

	// Creating/Updating received PodAutoscalers
	for _, settings := range settingsList.Settings {
		ns, name, err := cache.SplitMetaNamespaceKey(settings.ID)
		if err != nil || ns == "" {
			log.Errorf("Received invalid PodAutoscaler ID from config id:%s, version: %d, config key: %s, ID was: %s, discarding", rawConfig.Metadata.ID, rawConfig.Metadata.Version, configKey, settings.ID)
		}

		podAutoscaler, podAutoscalerFound := cr.store.LockRead(settings.ID, true)
		// If the PodAutoscaler is not found, we need to create it
		if !podAutoscalerFound {
			podAutoscaler = model.PodAutoscalerInternal{
				Namespace: ns,
				Name:      name,
			}
		}

		podAutoscaler.UpdateFromSettings(&settings.Spec, rawConfig.Metadata.Version, receivedTimestamp)
		cr.store.UnlockSet(settings.ID, podAutoscaler, configRetrieverStoreID)
	}

	return nil
}

func (cr *configRetriever) processAutoscalingValues(receivedTimestamp time.Time, configKey string, rawConfig state.RawConfig) error {
	valuesList := &model.AutoscalingValuesList{}
	err := json.Unmarshal(rawConfig.Config, &valuesList)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config id:%s, version: %d, config key: %s, err: %v", rawConfig.Metadata.ID, rawConfig.Metadata.Version, configKey, err)
	}

	for _, values := range valuesList.Values {
		podAutoscaler, podAutoscalerFound := cr.store.LockRead(values.ID, false)

		// If the PodAutoscaler is not found, it must be created through the controller
		// discarding the values received here.
		// The store is not locked as we call LockRead with lockOnMissing = false
		if !podAutoscalerFound {
			continue
		}

		// Update PodAutoscaler values with received values
		podAutoscaler.ScalingValues = values.ScalingValues
		podAutoscaler.ScalingValuesVersion = pointer.Ptr(rawConfig.Metadata.Version)
		podAutoscaler.ScalingValuesTimestamp = receivedTimestamp

		cr.store.UnlockSet(values.ID, podAutoscaler, configRetrieverStoreID)
	}

	return nil
}
