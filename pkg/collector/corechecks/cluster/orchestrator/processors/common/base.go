// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build orchestrator

// Package common provides basic handlers used by orchestrator processor
package common

import (
	model "github.com/DataDog/agent-payload/v5/process"
	"github.com/DataDog/datadog-agent/pkg/collector/corechecks/cluster/orchestrator/processors"
)

// BaseHandlers implements basic Handlers
type BaseHandlers struct{}

// BeforeCacheCheck is a handler called before cache check.
func (BaseHandlers) BeforeCacheCheck(_ processors.ProcessorContext, _, _ interface{}) (skip bool) {
	return
}

// BeforeMarshalling is a handler called before marshalling.
func (BaseHandlers) BeforeMarshalling(_ processors.ProcessorContext, _, _ interface{}) (skip bool) {
	return
}

// AfterMarshalling is a handler called after resource marshalling.
func (BaseHandlers) AfterMarshalling(_ processors.ProcessorContext, _, _ interface{}, _ []byte) (skip bool) {
	return
}

// ScrubBeforeMarshalling is a handler called to scrub data before marshalling.
func (BaseHandlers) ScrubBeforeMarshalling(_ processors.ProcessorContext, _ interface{}) {}

// ScrubBeforeExtraction is a handler called to scrub data before extraction.
func (BaseHandlers) ScrubBeforeExtraction(_ processors.ProcessorContext, _ interface{}) {}

// BuildManifestMessageBody is a handler called to build a message body out of a list of extracted resources.
func (BaseHandlers) BuildManifestMessageBody(ctx processors.ProcessorContext, resourceManifests []interface{}, groupSize int) model.MessageBody {
	return ExtractModelManifests(ctx, resourceManifests, groupSize)
}

// ExtractModelManifests creates the model manifest from the given manifests
func ExtractModelManifests(ctx processors.ProcessorContext, resourceManifests []interface{}, groupSize int) *model.CollectorManifest {
	pctx := ctx.(*processors.K8sProcessorContext)
	manifests := make([]*model.Manifest, 0, len(resourceManifests))

	for _, m := range resourceManifests {
		manifests = append(manifests, m.(*model.Manifest))
	}

	cm := &model.CollectorManifest{
		ClusterName: pctx.Cfg.KubeClusterName,
		ClusterId:   pctx.ClusterID,
		Manifests:   manifests,
		GroupId:     pctx.MsgGroupID,
		GroupSize:   int32(groupSize),
	}
	return cm
}
