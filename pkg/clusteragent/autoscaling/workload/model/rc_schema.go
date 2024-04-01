// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver

package model

import (
	datadoghq "github.com/DataDog/datadog-operator/apis/datadoghq/v1alpha1"
)

// AutoscalingValuesList holds a list of AutoscalingValues
type AutoscalingValuesList struct {
	// ClusterID is the ID of the cluster
	CluterID string `json:"cluster_id"`

	// Values is the list of AutoscalingValues
	Values []AutoscalingValues `json:"values"`
}

// AutoscalingValues holds the scaling values for a PodAutoscaler (horizontal and vertical)
type AutoscalingValues struct {
	// ID is the ID of the PodAutoscaler object in the format <namespace>/<name>
	ID string `json:"id"`

	// ScalingValues holds the scaling values
	ScalingValues `json:",inline"`
}

// ScalingValues represents the scaling values (horizontal and vertical) for a target
type ScalingValues struct {
	// Replicas is the desired number of replicas for the target
	Replicas *int32 `json:"replicas,omitempty"`

	// ContainerResources holds the resources for a container
	ContainerResources []datadoghq.DatadogPodAutoscalerContainerResources `json:"containerResources,omitempty"`
}

// AutoscalingSettingsList holds a list of AutoscalingSettings
type AutoscalingSettingsList struct {
	// ClusterID is the ID of the cluster
	CluterID string `json:"cluster_id"`

	// Settings is a list of .Spec
	Settings []AutoscalingSettings `json:"settings"`
}

// AutoscalingSettings is the .Spec of a PodAutoscaler retrieved through remote config
type AutoscalingSettings struct {
	// ID is the ID of the PodAutoscaler object in the format <namespace>/<name>
	ID string `json:"id"`

	// Spec is the full spec of the PodAutoscaler
	Spec datadoghq.DatadogPodAutoscalerSpec `json:"spec"`
}
