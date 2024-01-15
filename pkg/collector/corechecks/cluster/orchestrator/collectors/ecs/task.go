// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build orchestrator

// Package ecs defines a collector to collect ECS task
package ecs

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"

	"github.com/DataDog/datadog-agent/comp/core/workloadmeta"
	"github.com/DataDog/datadog-agent/pkg/collector/corechecks/cluster/orchestrator/collectors"
	"github.com/DataDog/datadog-agent/pkg/collector/corechecks/cluster/orchestrator/processors"
	"github.com/DataDog/datadog-agent/pkg/collector/corechecks/cluster/orchestrator/processors/ecs"
	transformers "github.com/DataDog/datadog-agent/pkg/collector/corechecks/cluster/orchestrator/transformers/ecs"
	"github.com/DataDog/datadog-agent/pkg/orchestrator"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// TaskCollector is a collector for ECS tasks.
type TaskCollector struct {
	metadata  *collectors.CollectorMetadata
	processor *processors.Processor
}

// NewTaskCollector creates a new collector for the ECS Task resource.
func NewTaskCollector() *TaskCollector {
	return &TaskCollector{
		metadata: &collectors.CollectorMetadata{
			IsStable:           false,
			IsMetadataProducer: true,
			IsManifestProducer: false,
			Name:               "ecstasks",
			NodeType:           orchestrator.ECSTask,
		},
		processor: processors.NewProcessor(new(ecs.TaskHandlers)),
	}
}

// Metadata is used to access information about the collector.
func (t *TaskCollector) Metadata() *collectors.CollectorMetadata {
	return t.metadata
}

// Init is used to initialize the collector.
func (t *TaskCollector) Init(_ *collectors.CollectorRunConfig) {}

// Run triggers the collection process.
func (t *TaskCollector) Run(rcfg *collectors.CollectorRunConfig) (*collectors.CollectorRunResult, error) {
	list := rcfg.WorkloadmetaStore.ListECSTasks()
	tasks := make([]transformers.TaskWithContainers, 0, len(list))
	for _, task := range list {
		tasks = append(tasks, t.FetchContainers(rcfg, task))
	}

	ctx := &processors.ECSProcessorContext{
		BaseProcessorContext: processors.BaseProcessorContext{
			Cfg:              rcfg.Config,
			MsgGroupID:       rcfg.MsgGroupRef.Inc(),
			NodeType:         t.metadata.NodeType,
			ManifestProducer: t.metadata.IsManifestProducer,
		},
	}
	if len(list) > 0 {
		ctx.AWSAccountID = list[0].AWSAccountID
		ctx.ClusterName = list[0].ClusterName
		ctx.Region = list[0].Region

		// If the cluster ID is not set, we generate it from the first task
		if rcfg.ClusterID == "" {
			rcfg.ClusterID = initClusterID(list[0].AWSAccountID, list[0].Region, list[0].ClusterName)
		}
		ctx.ClusterID = rcfg.ClusterID
	}

	processResult, processed := t.processor.Process(ctx, tasks)

	if processed == -1 {
		return nil, fmt.Errorf("unable to process resources: a panic occurred")
	}

	result := &collectors.CollectorRunResult{
		Result:             processResult,
		ResourcesListed:    len(list),
		ResourcesProcessed: processed,
	}

	return result, nil
}

// FetchContainers fetches the containers from workloadmeta store for a given task.
func (t *TaskCollector) FetchContainers(rcfg *collectors.CollectorRunConfig, task *workloadmeta.ECSTask) transformers.TaskWithContainers {
	ecsTask := transformers.TaskWithContainers{
		Task:       task,
		Containers: make([]*workloadmeta.Container, 0, len(task.Containers)),
	}

	for _, container := range task.Containers {
		c, err := rcfg.WorkloadmetaStore.GetContainer(container.ID)
		if err != nil {
			log.Errorc(err.Error(), orchestrator.ExtraLogContext...)
			continue
		}
		ecsTask.Containers = append(ecsTask.Containers, c)
	}

	return ecsTask
}

// initClusterID generates a cluster ID from the AWS account ID, region and cluster name.
func initClusterID(awsAccountID int, region, clusterName string) string {
	cluster := fmt.Sprintf("%d/%s/%s", awsAccountID, region, clusterName)

	hash := md5.New()
	hash.Write([]byte(cluster))
	hashString := hex.EncodeToString(hash.Sum(nil))
	uuid, err := uuid.FromBytes([]byte(hashString[0:16]))
	if err != nil {
		log.Errorc(err.Error(), orchestrator.ExtraLogContext...)
		return ""
	}
	return uuid.String()
}
