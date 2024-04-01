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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	kubeAutoscaling "github.com/DataDog/agent-payload/v5/autoscaling/kubernetes"
	datadoghq "github.com/DataDog/datadog-operator/apis/datadoghq/v1alpha1"
	"github.com/hashicorp/go-multierror"

	"github.com/DataDog/datadog-agent/pkg/clusteragent/autoscaling/workload/model"
	"github.com/DataDog/datadog-agent/pkg/remoteconfig/state"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

func (cr *configRetriever) processAutoscalingValues(receivedTimestamp time.Time, configKey string, rawConfig state.RawConfig) error {
	valuesList := &kubeAutoscaling.WorkloadValuesList{}
	err := json.Unmarshal(rawConfig.Config, &valuesList)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config id:%s, version: %d, config key: %s, err: %v", rawConfig.Metadata.ID, rawConfig.Metadata.Version, configKey, err)
	}

	for _, values := range valuesList.Values {
		processErr := cr.processAutoscalingValue(values, rawConfig.Metadata.Version, receivedTimestamp)
		if processErr != nil {
			err = multierror.Append(err, processErr)
		}
	}

	return err
}

func (cr *configRetriever) processAutoscalingValue(values *kubeAutoscaling.WorkloadValues, version uint64, timestamp time.Time) error {
	if values == nil || values.Id == "" {
		// Should never happen, but protecting the code from invalid inputs
		return nil
	}

	podAutoscaler, podAutoscalerFound := cr.store.LockRead(values.Id, false)
	// If the PodAutoscaler is not found, it must be created through the controller
	// discarding the values received here.
	// The store is not locked as we call LockRead with lockOnMissing = false
	if !podAutoscalerFound {
		return nil
	}

	// Update PodAutoscaler values with received values
	// Even on error, the PodAutoscaler can be partially updated, always setting it
	defer func() {
		cr.store.UnlockSet(values.Id, podAutoscaler, configRetrieverStoreID)
	}()
	scalingValues, err := parseAutoscalingValues(values)
	if err != nil {
		return fmt.Errorf("failed to parse scaling values for PodAutoscaler %s: %w", values.Id, err)
	}

	err = podAutoscaler.UpdateFromValues(scalingValues, version, timestamp)
	if err != nil {
		return fmt.Errorf("failed to update scaling values for PodAutoscaler %s: %w", values.Id, err)
	}

	return nil
}

func parseAutoscalingValues(values *kubeAutoscaling.WorkloadValues) (model.ScalingValues, error) {
	scalingValues := model.ScalingValues{}

	if values.Horizontal != nil {
		scalingValues.Horizontal = &model.HorizontalScalingValues{
			Source:   parseValueSource(values.Horizontal.Source),
			Replicas: values.Horizontal.Replicas,
		}
	}

	if values.Vertical != nil {
		scalingValues.Vertical = &model.VerticalScalingValues{
			Source: parseValueSource(values.Vertical.Source),
		}

		if containersNum := len(values.Vertical.Resources); containersNum > 0 {
			scalingValues.Vertical.ContainerResources = make([]datadoghq.DatadogPodAutoscalerContainerResources, 0, containersNum)

			for _, containerResources := range values.Vertical.Resources {
				convertedResources := datadoghq.DatadogPodAutoscalerContainerResources{
					Name: containerResources.ContainerName,
				}

				if limits, err := parseResourceList(containerResources.Limits); err == nil {
					convertedResources.Limits = limits
				} else {
					return model.ScalingValues{}, err
				}

				if requests, err := parseResourceList(containerResources.Requests); err == nil {
					convertedResources.Requests = requests
				} else {
					return model.ScalingValues{}, err
				}

				scalingValues.Vertical.ContainerResources = append(scalingValues.Vertical.ContainerResources, convertedResources)
			}
		}
	}

	return scalingValues, nil
}

func parseResourceList(resourceList []*kubeAutoscaling.ContainerResources_ResourceList) (corev1.ResourceList, error) {
	if resourceList == nil {
		return nil, nil
	}

	corev1ResourceList := make(corev1.ResourceList, len(resourceList))
	for _, containerResource := range resourceList {
		if _, found := corev1ResourceList[corev1.ResourceName(containerResource.Name)]; found {
			return nil, fmt.Errorf("resource %s is duplicated", containerResource.Name)
		}

		qty, err := resource.ParseQuantity(containerResource.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to parse resource %s value %s: %w", containerResource.Name, containerResource.Value, err)
		}

		corev1ResourceList[corev1.ResourceName(containerResource.Name)] = qty
	}

	return corev1ResourceList, nil
}

func parseValueSource(source kubeAutoscaling.ValueSource) datadoghq.DatadogPodAutoscalerValueSource {
	switch source {
	case kubeAutoscaling.ValueSource_autoscaling:
		return datadoghq.Autoscaling
	case kubeAutoscaling.ValueSource_manual:
		return datadoghq.Manual
	default:
		// Should never happen, but protecting the code from invalid inputs
		log.Errorf("Unknown value source %s, defaulting to autoscaling", source)
		return datadoghq.Autoscaling
	}
}
