// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build orchestrator

package ecs

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"

	"github.com/DataDog/agent-payload/v5/process"
	"github.com/DataDog/datadog-agent/comp/core/workloadmeta"
	"github.com/DataDog/datadog-agent/pkg/aggregator/mocksender"
	"github.com/DataDog/datadog-agent/pkg/orchestrator"
	oconfig "github.com/DataDog/datadog-agent/pkg/orchestrator/config"
	"github.com/DataDog/datadog-agent/pkg/serializer/types"
)

type fakeWorkloadmetaStore struct {
	workloadmeta.Component
	notifiedEvents []*workloadmeta.ECSTask
}

func (store *fakeWorkloadmetaStore) AddECSTasks(task ...*workloadmeta.ECSTask) {
	store.notifiedEvents = append(store.notifiedEvents, task...)
}

func (store *fakeWorkloadmetaStore) ListECSTasks() (events []*workloadmeta.ECSTask) {
	return store.notifiedEvents
}

func (store *fakeWorkloadmetaStore) GetContainer(id string) (*workloadmeta.Container, error) {
	if id == "938f6d263c464aa5985dc67ab7f38a7e-1714341083" {
		return container1(), nil
	}
	if id == "938f6d263c464aa5985dc67ab7f38a7e-1714341084" {
		return container2(), nil
	}
	return nil, fmt.Errorf("container not found")
}

type fakeSender struct {
	mocksender.MockSender
	messages   []process.MessageBody
	clusterIDs []string
	nodeTypes  []int
}

func (s *fakeSender) OrchestratorMetadata(msgs []types.ProcessMessageBody, clusterID string, nodeType int) {
	s.messages = append(s.messages, msgs...)
	s.clusterIDs = append(s.clusterIDs, clusterID)
	s.nodeTypes = append(s.nodeTypes, nodeType)
}

func (s *fakeSender) Flush() []process.MessageBody {
	messages := s.messages
	s.messages = s.messages[:0]
	return messages
}

func TestNotECS(t *testing.T) {
	check, _, sender := prepareTest("notECS")
	err := check.Run()
	require.NoError(t, err)
	require.Len(t, sender.messages, 0)
}

func TestECS(t *testing.T) {
	check, store, sender := prepareTest("ecs")

	// add one task to fake WorkloadmetaStore
	task1Id := "123"
	store.AddECSTasks(task(task1Id))

	err := check.Run()
	require.NoError(t, err)

	// should receive one message
	messages := sender.Flush()
	require.Len(t, messages, 1)
	require.Equal(t, expected(task1Id), messages[0])
	require.Equal(t, expected(task1Id).ClusterId, sender.clusterIDs[0])
	require.Equal(t, int(orchestrator.ECSTask), sender.nodeTypes[0])

	// add another task with different id to fake WorkloadmetaStore
	task2Id := "124"
	store.AddECSTasks(task(task2Id))

	err = check.Run()
	require.NoError(t, err)

	messages = sender.Flush()
	require.Len(t, messages, 1)
	require.Equal(t, expected(task2Id), messages[0])
	require.Equal(t, sender.clusterIDs[0], sender.clusterIDs[1])
	require.Equal(t, sender.nodeTypes[0], sender.nodeTypes[1])

	// 0 message should be received as tasks are skipped by cache
	err = check.Run()
	require.NoError(t, err)
	messages = sender.Flush()
	require.Len(t, messages, 0)
}

// prepareTest returns a check, a fake workloadmeta store and a fake sender
func prepareTest(env string) (*Check, *fakeWorkloadmetaStore, *fakeSender) {
	orchConfig := oconfig.NewDefaultOrchestratorConfig()
	orchConfig.OrchestrationCollectionEnabled = true
	orchConfig.OrchestrationECSCollectionEnabled = true

	store := &fakeWorkloadmetaStore{}
	sender := &fakeSender{}

	c := &Check{
		sender:            sender,
		WorkloadmetaStore: store,
		config:            orchConfig,
		groupID:           atomic.NewInt32(0),
	}

	if env == "ecs" {
		c.isECSCollectionEnabledFunc = func() bool { return true }
	}

	return c, store, sender
}

func task(id string) *workloadmeta.ECSTask {
	return &workloadmeta.ECSTask{
		EntityID: workloadmeta.EntityID{
			Kind: workloadmeta.KindECSTask,
			ID:   fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:task/%s", id),
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
	}
}

func container1() *workloadmeta.Container {
	return &workloadmeta.Container{
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
			ExitCode: func(i uint32) *uint32 {
				return &i
			}(2),
		},
		Type: "NORMAL",
	}

}

func container2() *workloadmeta.Container {
	return &workloadmeta.Container{
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
	}
}

func expected(id string) *process.CollectorECSTask {
	// version is determined by hashing the json
	version := "3719516614568364302"
	groupID := int32(1)
	if id != "123" {
		version = "10786000042049282301"
		groupID = 2
	}

	return &process.CollectorECSTask{
		AwsAccountID: 123456789012,
		ClusterName:  "ecs-cluster",
		ClusterId:    "63306530-3932-3664-3664-376566306132",
		Region:       "us-east-1",
		GroupId:      groupID,
		GroupSize:    1,
		Tasks: []*process.ECSTask{
			{
				Arn:           fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:task/%s", id),
				TaskVersion:   version,
				LaunchType:    "ec2",
				DesiredStatus: "RUNNING",
				KnownStatus:   "RUNNING",
				Family:        "redis",
				Version:       "1",
				VpcId:         "vpc-12345678",
				ServiceName:   "redis",
				Limits:        map[string]float64{"CPU": 1, "Memory": 2048},
				EcsTags: []string{
					"ecs.cluster:ecs-cluster",
					"region:us-east-1",
				},
				ContainerInstanceTags: []string{
					"instance:instance-1",
					"region:us-east-1",
				},
				Containers: []*process.ECSContainer{
					{
						DockerID:   "938f6d263c464aa5985dc67ab7f38a7e-1714341083",
						DockerName: "log_router",
						Name:       "log_router_container",
						Image:      "amazon/aws-for-fluent-bit:latest",
						Type:       "NORMAL",
						Ports: []*process.ECSContainerPort{
							{
								ContainerPort: 80,
								HostPort:      80,
							},
						},
						Health: &process.ECSContainerHealth{
							Status: "HEALTHY",
							ExitCode: &process.ECSContainerExitCode{
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
						Ports: []*process.ECSContainerPort{
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
			},
		},
	}

}
