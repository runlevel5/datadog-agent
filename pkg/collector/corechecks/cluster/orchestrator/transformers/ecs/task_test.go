// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build orchestrator

package ecs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	model "github.com/DataDog/agent-payload/v5/process"
	"github.com/DataDog/datadog-agent/comp/core/workloadmeta"
)

func TestExtractECSTask(t *testing.T) {
	now := time.Now()

	actual := ExtractECSTask(TaskWithContainers{
		Task: &workloadmeta.ECSTask{
			EntityID: workloadmeta.EntityID{
				Kind: workloadmeta.KindECSTask,
				ID:   "arn:aws:ecs:us-east-1:123456789012:task/12345678-1234-1234-1234-123456789012",
			},
			EntityMeta: workloadmeta.EntityMeta{
				Name: "12345678-1234-1234-1234-123456789012",
			},
			ClusterName:   "ecs-cluster",
			AWSAccountID:  123456789012,
			Region:        "us-east-1",
			LaunchType:    workloadmeta.ECSLaunchTypeEC2,
			Family:        "redis",
			Version:       "1",
			DesiredStatus: "RUNNING",
			KnownStatus:   "RUNNING",
			VPCID:         "vpc-12345678",
			ServiceName:   "redis",
			PullStartedAt: &now,
			Limits:        map[string]float64{"CPU": 1, "Memory": 2048},
			Containers: []workloadmeta.OrchestratorContainer{
				{
					ID: "938f6d263c464aa5985dc67ab7f38a7e-1714341083",
				},
				{
					ID: "938f6d263c464aa5985dc67ab7f38a7e-1714341084",
				},
			},
			Tags: workloadmeta.ECSTaskTags{
				"ecs.cluster": "ecs-cluster",
				"region":      "us-east-1",
			},
			ContainerInstanceTags: workloadmeta.ContainerInstanceTags{
				"instance": "instance-1",
				"region":   "us-east-1",
			},
		},
		Containers: []*workloadmeta.Container{
			{
				EntityID: workloadmeta.EntityID{
					Kind: workloadmeta.KindContainer,
					ID:   "938f6d263c464aa5985dc67ab7f38a7e-1714341083",
				},
				EntityMeta: workloadmeta.EntityMeta{
					Name: "log_router",
					Labels: map[string]string{
						"com.amazonaws.ecs.cluster":        "ecs-cluster",
						"com.amazonaws.ecs.container-name": "log_router",
					},
				},
				ContainerName: "log_router_container",
				Image: workloadmeta.ContainerImage{
					RawName: "amazon/aws-for-fluent-bit:latest",
					Name:    "amazon/aws-for-fluent-bit",
				},
				Ports: []workloadmeta.ContainerPort{
					{
						Port:     80,
						HostPort: 80,
					},
				},
				Health: &workloadmeta.ContainerHealthStatus{
					Status: "HEALTHY",
					Since:  &now,
					ExitCode: func(i uint32) *uint32 {
						return &i
					}(2),
				},
				Type: "NORMAL",
			},
			{
				EntityID: workloadmeta.EntityID{
					Kind: workloadmeta.KindContainer,
					ID:   "938f6d263c464aa5985dc67ab7f38a7e-1714341084",
				},
				EntityMeta: workloadmeta.EntityMeta{
					Name: "redis",
				},
				Image: workloadmeta.ContainerImage{
					RawName: "redis/redis:latest",
					Name:    "redis/redis",
				},
				ContainerName: "redis",
				Ports: []workloadmeta.ContainerPort{
					{
						Port:     90,
						HostPort: 90,
					},
					{
						Port:     81,
						HostPort: 8080,
					},
				},
				Type: "NORMAL",
			},
		},
	})

	expected := &model.ECSTask{
		Arn:           "arn:aws:ecs:us-east-1:123456789012:task/12345678-1234-1234-1234-123456789012",
		LaunchType:    "ec2",
		DesiredStatus: "RUNNING",
		KnownStatus:   "RUNNING",
		Family:        "redis",
		Version:       "1",
		VpcId:         "vpc-12345678",
		ServiceName:   "redis",
		PullStartedAt: now.Unix(),
		Limits:        map[string]float64{"CPU": 1, "Memory": 2048},
		EcsTags: []string{
			"ecs.cluster:ecs-cluster",
			"region:us-east-1",
		},
		ContainerInstanceTags: []string{
			"instance:instance-1",
			"region:us-east-1",
		},
		Containers: []*model.ECSContainer{
			{
				DockerID:   "938f6d263c464aa5985dc67ab7f38a7e-1714341083",
				DockerName: "log_router",
				Name:       "log_router_container",
				Image:      "amazon/aws-for-fluent-bit:latest",
				Type:       "NORMAL",
				Ports: []*model.ECSContainerPort{
					{
						ContainerPort: 80,
						HostPort:      80,
					},
				},
				Health: &model.ECSContainerHealth{
					Status: "HEALTHY",
					Since:  now.Unix(),
					ExitCode: &model.ECSContainerExitCode{
						ExitCode: 2,
					},
				},
				Labels: []string{
					"com.amazonaws.ecs.cluster:ecs-cluster",
					"com.amazonaws.ecs.container-name:log_router",
				},
			},
			{
				DockerID:   "938f6d263c464aa5985dc67ab7f38a7e-1714341084",
				DockerName: "redis",
				Name:       "redis",
				Image:      "redis/redis:latest",
				Type:       "NORMAL",
				Ports: []*model.ECSContainerPort{
					{
						ContainerPort: 90,
						HostPort:      90,
					},
					{
						ContainerPort: 81,
						HostPort:      8080,
					},
				},
			},
		},
	}

	require.Equal(t, expected, actual)
}
