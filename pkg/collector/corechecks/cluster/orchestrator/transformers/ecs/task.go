// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build orchestrator

// Package ecs provides methods for converting ECS resources to protobuf model.
package ecs

import (
	"fmt"
	"sort"
	"time"

	model "github.com/DataDog/agent-payload/v5/process"
	"github.com/DataDog/datadog-agent/comp/core/workloadmeta"
	"github.com/DataDog/datadog-agent/pkg/tagger"
	"github.com/DataDog/datadog-agent/pkg/tagger/collectors"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// TaskWithContainers represents an ECS task with its containers fetched from the workloadmeta store
type TaskWithContainers struct {
	Task       *workloadmeta.ECSTask
	Containers []*workloadmeta.Container
}

// ExtractECSTask returns the protobuf model corresponding to an ECS Task resource.
func ExtractECSTask(task TaskWithContainers) *model.ECSTask {
	if task.Task == nil {
		return nil
	}
	taskModel := &model.ECSTask{
		Arn:                     task.Task.EntityID.ID,
		LaunchType:              string(task.Task.LaunchType),
		DesiredStatus:           task.Task.DesiredStatus,
		KnownStatus:             task.Task.KnownStatus,
		Family:                  task.Task.Family,
		Version:                 task.Task.Version,
		AvailabilityZone:        task.Task.AvailabilityZone,
		Limits:                  task.Task.Limits,
		EphemeralStorageMetrics: task.Task.EphemeralStorageMetrics,
		ServiceName:             task.Task.ServiceName,
		VpcId:                   task.Task.VPCID,
		PullStartedAt:           extractTimestampPtr(task.Task.PullStartedAt),
		PullStoppedAt:           extractTimestampPtr(task.Task.PullStoppedAt),
		ExecutionStoppedAt:      extractTimestampPtr(task.Task.ExecutionStoppedAt),
		Containers:              extractECSContainer(task.Containers),
	}

	tags, err := tagger.Tag(fmt.Sprintf("ecs_task://%s", task.Task.EntityID.ID), collectors.HighCardinality)
	if err != nil {
		log.Debugf("Could not retrieve tags for task: %s", err.Error())
	}

	taskModel.Tags = tags
	taskModel.EcsTags = toTags(task.Task.Tags)
	taskModel.ContainerInstanceTags = toTags(task.Task.ContainerInstanceTags)

	// Enforce order consistency on slices
	sort.Strings(taskModel.Tags)
	sort.Strings(taskModel.EcsTags)
	sort.Strings(taskModel.ContainerInstanceTags)

	return taskModel
}

func extractECSContainer(containers []*workloadmeta.Container) []*model.ECSContainer {
	ecsContainers := make([]*model.ECSContainer, 0, len(containers))
	for _, container := range containers {
		if container == nil {
			continue
		}
		ecsContainer := &model.ECSContainer{
			DockerID:      container.EntityID.ID,
			DockerName:    container.EntityMeta.Name,
			Name:          container.ContainerName,
			Image:         container.Image.RawName,
			ImageID:       container.Image.ID,
			DesiredStatus: container.DesiredStatus,
			KnownStatus:   container.KnownStatus,
			Type:          container.Type,
			LogDriver:     container.LogDriver,
			LogOptions:    container.LogOptions,
			ContainerArn:  container.ContainerARN,
			CreatedAt:     extractTimestamp(container.State.CreatedAt),
			StartedAt:     extractTimestamp(container.State.StartedAt),
			FinishedAt:    extractTimestamp(container.State.FinishedAt),
			Labels:        toTags(container.EntityMeta.Labels),
			Ports:         extractECSContainerPort(container),
			Networks:      extractECSContainerNetworks(container),
			Volumes:       extractECSContainerVolume(container),
			Health:        extractECSContainerHealth(container),
			ExitCode:      extractExitCode(container.State.ExitCode),
		}
		sort.Strings(ecsContainer.Labels)
		ecsContainers = append(ecsContainers, ecsContainer)
	}
	return ecsContainers
}

func extractTimestampPtr(t *time.Time) int64 {
	if t == nil || t.IsZero() {
		return 0
	}
	return t.Unix()
}

func extractTimestamp(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func extractExitCode(exitCode *uint32) *model.ECSContainerExitCode {
	if exitCode == nil {
		return nil
	}
	return &model.ECSContainerExitCode{
		ExitCode: int32(*exitCode),
	}
}

func extractECSContainerPort(container *workloadmeta.Container) []*model.ECSContainerPort {
	if len(container.Ports) == 0 {
		return nil
	}

	ports := make([]*model.ECSContainerPort, 0, len(container.Ports))
	for _, port := range container.Ports {
		ports = append(ports, &model.ECSContainerPort{
			ContainerPort: int32(port.Port),
			HostPort:      int32(port.HostPort),
			Protocol:      port.Protocol,
			HostIp:        port.HostIP,
		})
	}
	return ports
}

func extractECSContainerNetworks(container *workloadmeta.Container) []*model.ECSContainerNetwork {
	if len(container.Networks) == 0 {
		return nil
	}

	networks := make([]*model.ECSContainerNetwork, 0, len(container.Networks))
	for _, network := range container.Networks {
		networks = append(networks, &model.ECSContainerNetwork{
			NetworkMode:   network.NetworkMode,
			Ipv4Addresses: network.IPv4Addresses,
			Ipv6Addresses: network.IPv6Addresses,
		})
	}
	return networks
}

func extractECSContainerVolume(container *workloadmeta.Container) []*model.ECSContainerVolume {
	if len(container.Volumes) == 0 {
		return nil
	}

	volumes := make([]*model.ECSContainerVolume, 0, len(container.Volumes))
	for _, volume := range container.Volumes {
		volumes = append(volumes, &model.ECSContainerVolume{
			DockerName:  volume.DockerName,
			Source:      volume.Source,
			Destination: volume.Destination,
		})
	}
	return volumes
}

func extractECSContainerHealth(container *workloadmeta.Container) *model.ECSContainerHealth {
	if container.Health == nil {
		return nil
	}

	health := &model.ECSContainerHealth{
		Status:   container.Health.Status,
		Output:   container.Health.Output,
		ExitCode: extractExitCode(container.Health.ExitCode),
		Since:    extractTimestampPtr(container.Health.Since),
	}

	return health
}

func toTags(tags map[string]string) []string {
	var result []string
	for k, v := range tags {
		result = append(result, fmt.Sprintf("%s:%s", k, v))
	}
	return result
}
