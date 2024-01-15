// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package workloadmeta

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewContainerImage(t *testing.T) {
	tests := []struct {
		name                      string
		imageName                 string
		expectedWorkloadmetaImage ContainerImage
		expectsErr                bool
	}{
		{
			name:      "image with tag",
			imageName: "datadog/agent:7",
			expectedWorkloadmetaImage: ContainerImage{
				RawName:   "datadog/agent:7",
				Name:      "datadog/agent",
				ShortName: "agent",
				Tag:       "7",
				ID:        "0",
			},
		}, {
			name:      "image without tag",
			imageName: "datadog/agent",
			expectedWorkloadmetaImage: ContainerImage{
				RawName:   "datadog/agent",
				Name:      "datadog/agent",
				ShortName: "agent",
				Tag:       "latest", // Default to latest when there's no tag
				ID:        "1",
			},
		},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			image, err := NewContainerImage(strconv.Itoa(i), test.imageName)
			assert.NoError(t, err)
			assert.Equal(t, test.expectedWorkloadmetaImage, image)
		})
	}
}

func TestContainerString(t *testing.T) {
	date, err := time.Parse("2006-01-02 15:04", "2023-01-15 15:04")
	assert.NoError(t, err)

	container := Container{
		EntityID: EntityID{
			Kind: KindContainer,
			ID:   "container-1-id",
		},
		EntityMeta: EntityMeta{
			Name:      "container-1",
			Namespace: "default",
			Annotations: map[string]string{
				"annotation-1": "value-1",
			},
			Labels: map[string]string{
				"label-1": "value-1",
			},
		},
		Image: ContainerImage{
			Name:     "image",
			Tag:      "tag",
			ID:       "image-id",
			Registry: "registry",
			RawName:  "registry/image:tag",
		},
		Owner: &EntityID{
			Kind: KindECSTask,
			ID:   "task-id",
		},
		Runtime:       ContainerRuntimeDocker,
		RuntimeFlavor: "docker",
		Hostname:      "host-1",
		State: ContainerState{
			Running:   true,
			Health:    ContainerHealthHealthy,
			Status:    ContainerStatusRunning,
			CreatedAt: date,
			StartedAt: date,
		},
		KnownStatus: "RUNNING",
		Health: &ContainerHealthStatus{
			Status: "Healthy",
			Output: "custom-health-check-output",
		},
		NetworkIPs: map[string]string{
			"awsvpc": "ip-1",
		},
		Volumes: []ContainerVolume{
			{
				DockerName:  "volume-1",
				Source:      "/source/1",
				Destination: "/container/1",
			},
			{
				DockerName:  "volume-2",
				Source:      "/source/2",
				Destination: "/container/2",
			},
		},
		Ports: []ContainerPort{
			{
				Name:     "port-1",
				Port:     8080,
				HostPort: 80,
				Protocol: "tcp",
				HostIP:   "host-ip-1",
			},
			{
				Name:     "port-2",
				Port:     8081,
				HostPort: 81,
				Protocol: "tcp",
				HostIP:   "host-ip-2",
			},
		},
		Type: "NORMAL",
		Limits: map[string]uint64{
			"cpu": 100,
		},
		LogDriver: "json-file",
		LogOptions: map[string]string{
			"max-size": "10m",
		},
		Snapshotter: "snapshotter",
	}
	expected := `----------- Entity ID -----------
Kind: container ID: container-1-id
----------- Entity Meta -----------
Name: container-1
Namespace: default
Annotations: annotation-1:value-1
Labels: label-1:value-1
----------- Image -----------
Name: image
Tag: tag
ID: image-id
Raw Name: registry/image:tag
Short Name:
----------- Container Info -----------
Runtime: docker
RuntimeFlavor: docker
Running: true
Status: running
Health: healthy
Created At: 2023-01-15 15:04:00 +0000 UTC
Started At: 2023-01-15 15:04:00 +0000 UTC
Finished At: 0001-01-01 00:00:00 +0000 UTC
----------- Resources -----------
Allowed env variables:
Hostname: host-1
Network IPs: awsvpc:ip-1
PID: 0
----------- Ports -----------
Port: 8080
Name: port-1
Protocol: tcp
Host Port: 80
Host IP: host-ip-1
Port: 8081
Name: port-2
Protocol: tcp
Host Port: 81
Host IP: host-ip-2
----------- Volumes -----------
Name: volume-1
Source: /source/1
Destination: /container/1
Name: volume-2
Source: /source/2
Destination: /container/2
----------- ECS Container Health -----------
Status: Healthy
Since: <nil>
ExitCode: <nil>
Output: custom-health-check-output
----------- Container Priorities On ECS -----------
CreatedAt: 2023-01-15 15:04:00 +0000 UTC
StartedAt: 2023-01-15 15:04:00 +0000 UTC
DesiredStatus:
KnownStatus: RUNNING
Type: NORMAL
Limits: map[cpu:100]
LogDriver: json-file
LogOptions: map[max-size:10m]
Snapshotter: snapshotter
`
	compareTestOutput(t, expected, container.String(true))
}

func TestECSTaskString(t *testing.T) {
	task := ECSTask{
		EntityID: EntityID{
			Kind: KindECSTask,
			ID:   "task-1-id",
		},
		EntityMeta: EntityMeta{
			Name: "task-1",
		},
		Containers: []OrchestratorContainer{
			{
				ID:   "container-1-id",
				Name: "container-1",
				Image: ContainerImage{
					RawName:   "datadog/agent:7",
					Name:      "datadog/agent",
					ShortName: "agent",
					Tag:       "7",
					ID:        "0",
				},
			},
		},
		Family:  "family-1",
		Version: "revision-1",
		EphemeralStorageMetrics: map[string]int64{
			"memory": 100,
			"cpu":    200,
		},
	}
	expected := `----------- Entity ID -----------
Kind: ecs_task ID: task-1-id
----------- Entity Meta -----------
Name: task-1
Namespace:
Annotations:
Labels:
----------- Containers -----------
Name: container-1 ID: container-1-id
----------- Task Info -----------
Tags:
Container Instance Tags:
Cluster Name:
Region:
Availability Zone:
Family: family-1
Version: revision-1
Launch Type:
AWS Account ID: 0
Desired Status:
Known Status:
VPC ID:
Ephemeral Storage Metrics: map[cpu:200 memory:100]
Limits: map[]
`
	compareTestOutput(t, expected, task.String(true))
}
func compareTestOutput(t *testing.T, expected, actual string) {
	assert.Equal(t, strings.ReplaceAll(expected, " ", ""), strings.ReplaceAll(actual, " ", ""))
}
