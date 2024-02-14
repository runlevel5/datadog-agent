// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver

package agentsidecar

import (
	"encoding/json"
	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"os"
)

const (
	agentSidecarContainerName = "datadog-agent-injected"

	providerFargate = "fargate"
)

func getDefaultSidecarTemplate() *corev1.Container {
	ddSite := os.Getenv("DD_SITE")
	if ddSite == "" {
		ddSite = config.DefaultSite
	}

	agentContainer := &corev1.Container{
		Env: []corev1.EnvVar{
			{
				Name: "DD_API_KEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						Key: "api-key",
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "datadog-secret",
						},
					},
				},
			},
			{
				Name:  "DD_SITE",
				Value: ddSite,
			},
			{
				Name:  "DD_CLUSTER_NAME",
				Value: config.Datadog.GetString("cluster_name"),
			},
			{
				Name: "DD_KUBERNETES_KUBELET_NODENAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "spec.nodeName",
					},
				},
			},
		},
		Image:           "datadog/agent",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Name:            agentSidecarContainerName,
		Resources: corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				"memory": resource.MustParse("256Mi"),
				"cpu":    resource.MustParse("200m"),
			},
			Limits: map[corev1.ResourceName]resource.Quantity{
				"memory": resource.MustParse("256Mi"),
				"cpu":    resource.MustParse("200m"),
			},
		},
	}

	return agentContainer
}

////////////////////////////////
//                            //
//     Provider Overrides     //
//                            //
////////////////////////////////

// ProviderIsSupported indicates whether the provider is supported by agent sidecar injection
func ProviderIsSupported(provider string) bool {
	switch provider {
	case providerFargate:
		return true
	default:
		return false
	}
}

func applyProviderOverrides(container *corev1.Container) {
	provider := config.Datadog.GetString("admission_controller.agent_sidecar.provider")

	if !ProviderIsSupported(provider) {
		log.Errorf("unsupported provider: %v", provider)
		return
	}

	switch provider {
	case providerFargate:
		applyFargateOverrides(container)
	}
}

func applyFargateOverrides(container *corev1.Container) {
	if container == nil {
		return
	}

	container.Env = append(container.Env, corev1.EnvVar{
		Name:  "DD_EKS_FARGATE",
		Value: "true",
	})
}

////////////////////////////////
//                            //
//     Profiles Overrides     //
//                            //
////////////////////////////////

// ProfileOverride represents environment variables and resource requirements overrides
type ProfileOverride struct {
	EnvVars              []corev1.EnvVar             `json:"env,omitempty"`
	ResourceRequirements corev1.ResourceRequirements `json:"resources,omitempty"`
}

func applyProfileOverrides(container *corev1.Container) {
	if container == nil {
		return
	}

	// Read and parse profiles
	profilesJSON := config.Datadog.GetString("admission_controller.agent_sidecar.profiles")

	var profiles []ProfileOverride

	err := json.Unmarshal([]byte(profilesJSON), &profiles)
	if err != nil {
		log.Errorf("failed to parse profiles for admission controller agent sidecar injection: %s", err)
		return
	}

	if len(profiles) > 1 {
		log.Errorf("only 1 profile is supported")
		return
	}

	if len(profiles) == 0 {
		return
	}

	overrides := profiles[0]

	// Apply environment variable overrides
	for _, envVarOverride := range overrides.EnvVars {
		// Check if the environment variable already exists in the container
		var found bool
		for i, envVar := range container.Env {
			if envVar.Name == envVarOverride.Name {
				// Override the existing environment variable value
				container.Env[i] = envVarOverride
				found = true
				break
			}
		}
		// If the environment variable doesn't exist, add it to the container
		if !found {
			container.Env = append(container.Env, envVarOverride)
		}
	}

	// Apply resource requirement overrides
	if overrides.ResourceRequirements.Limits != nil {
		if container.Resources.Limits == nil {
			container.Resources.Limits = make(corev1.ResourceList)
		}
		for k, v := range overrides.ResourceRequirements.Limits {
			container.Resources.Limits[k] = v
		}
	}
	if overrides.ResourceRequirements.Requests != nil {
		if container.Resources.Requests == nil {
			container.Resources.Requests = make(corev1.ResourceList)
		}
		for k, v := range overrides.ResourceRequirements.Requests {
			container.Resources.Requests[k] = v
		}
	}
}
