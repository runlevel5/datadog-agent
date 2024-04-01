// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver

package model

import (
	datadoghq "github.com/DataDog/datadog-operator/apis/datadoghq/v1alpha1"
)

// AutoscalingSettingsList holds a list of AutoscalingSettings
type AutoscalingSettingsList struct {
	// Settings is a list of .Spec
	Settings []AutoscalingSettings `json:"settings"`
}

// AutoscalingSettings is the .Spec of a PodAutoscaler retrieved through remote config
type AutoscalingSettings struct {
	// Namespace is the namespace of the PodAutoscaler
	Namespace string `json:"namespace"`

	// Name is the name of the PodAutoscaler
	Name string `json:"name"`

	// Spec is the full spec of the PodAutoscaler
	Spec *datadoghq.DatadogPodAutoscalerSpec `json:"spec"`
}

// ScalingValues represents the scaling values (horizontal and vertical) for a target
type ScalingValues struct {
	Horizontal *HorizontalScalingValues `json:"horizontal,omitempty"`
	Vertical   *VerticalScalingValues   `json:"vertical,omitempty"`
}

// HorizontalScalingValues holds the horizontal scaling values for a target
type HorizontalScalingValues struct {
	// Source is the source of the value
	Source datadoghq.DatadogPodAutoscalerValueSource `json:"source"`

	// Replicas is the desired number of replicas for the target
	Replicas *int32 `json:"replicas,omitempty"`
}

// VerticalScalingValues holds the vertical scaling values for a target
type VerticalScalingValues struct {
	// Source is the source of the value
	Source datadoghq.DatadogPodAutoscalerValueSource `json:"source"`

	// ContainerResources holds the resources for a container
	ContainerResources []datadoghq.DatadogPodAutoscalerContainerResources `json:"containerResources,omitempty"`
}
