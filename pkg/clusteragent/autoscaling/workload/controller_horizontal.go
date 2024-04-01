// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver

package workload

import (
	"context"
	"fmt"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	scaleclient "k8s.io/client-go/scale"

	"github.com/DataDog/datadog-agent/pkg/clusteragent/autoscaling"
	"github.com/DataDog/datadog-agent/pkg/clusteragent/autoscaling/workload/model"
	"github.com/DataDog/datadog-agent/pkg/util/pointer"
)

type horizontalController struct {
	scaler scaler
}

func newHorizontalReconciler(restMapper apimeta.RESTMapper, scaleGetter scaleclient.ScalesGetter) *horizontalController {
	return &horizontalController{
		scaler: newScaler(restMapper, scaleGetter),
	}
}

func (hr *horizontalController) sync(ctx context.Context, autoscalerInternal *model.PodAutoscalerInternal) (processResult, error) {
	// Get the current scale of the target resource
	scale, gr, err := hr.scaler.get(ctx, autoscalerInternal.Namespace, autoscalerInternal.Name, autoscalerInternal.TargetGVK)
	if err != nil {
		return withStatusUpdate(true, autoscaling.Requeue), fmt.Errorf("failed to get scale subresource for autoscaler %s, err: %w", autoscalerInternal.ID(), err)
	}

	// Update the current number of replicas from the scaling values
	statusUpdateRequired := false
	if autoscalerInternal.CurrentReplicas == nil || *autoscalerInternal.CurrentReplicas != scale.Spec.Replicas {
		statusUpdateRequired = true
		autoscalerInternal.CurrentReplicas = pointer.Ptr(scale.Spec.Replicas)
	}

	// Perform the scaling operation
	if autoscalerInternal.ScalingValues.Replicas != nil && *autoscalerInternal.ScalingValues.Replicas != scale.Spec.Replicas {
		statusUpdateRequired = true
		scale.Spec.Replicas = *autoscalerInternal.ScalingValues.Replicas

		if _, err := hr.scaler.update(ctx, gr, scale); err != nil {
			return withStatusUpdate(true, autoscaling.Requeue), fmt.Errorf("failed to update scale subresource for autoscaler %s, err: %w", autoscalerInternal.ID(), err)
		}
	}

	return withStatusUpdate(statusUpdateRequired, autoscaling.NoRequeue), nil
}
