// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver

package workload

import (
	"context"
	"fmt"
	"math"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	scaleclient "k8s.io/client-go/scale"

	"github.com/DataDog/datadog-agent/pkg/clusteragent/autoscaling"
	"github.com/DataDog/datadog-agent/pkg/clusteragent/autoscaling/workload/model"
	"github.com/DataDog/datadog-agent/pkg/util/pointer"
)

type scaleDirection int

const (
	scaleUp   scaleDirection = 0
	scaleDown scaleDirection = 1

	defaultMinReplicas int32 = 1
	defaultMaxReplicas int32 = math.MaxInt32
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
	gvk, err := autoscalerInternal.GetTargetGVK()
	if err != nil {
		return withStatusUpdate(true, autoscaling.NoRequeue), fmt.Errorf("failed to parse API version %s: %w", autoscalerInternal.Spec.TargetRef.APIVersion, err)
	}

	// Get the current scale of the target resource
	scale, gr, err := hr.scaler.get(ctx, autoscalerInternal.Namespace, autoscalerInternal.Spec.TargetRef.Name, gvk)
	if err != nil {
		return withStatusUpdate(true, autoscaling.Requeue), fmt.Errorf("failed to get scale subresource for autoscaler %s, err: %w", autoscalerInternal.ID(), err)
	}

	// Update the current number of replicas from the scaling values
	statusUpdateRequired := false
	if autoscalerInternal.CurrentReplicas == nil || *autoscalerInternal.CurrentReplicas != scale.Spec.Replicas {
		statusUpdateRequired = true
		autoscalerInternal.CurrentReplicas = pointer.Ptr(scale.Spec.Replicas)
	}

	result, err := hr.performScaling(ctx, autoscalerInternal, gr, scale)
	result.updateStatus = result.updateStatus || statusUpdateRequired
	return result, err
}

func (hr *horizontalController) performScaling(ctx context.Context, autoscalerInternal *model.PodAutoscalerInternal, gr schema.GroupResource, scale *autoscalingv1.Scale) (processResult, error) {
	// No update required, except perhaps status, bailing out
	if autoscalerInternal.ScalingValues.Horizontal == nil ||
		autoscalerInternal.ScalingValues.Horizontal.Replicas == nil ||
		*autoscalerInternal.ScalingValues.Horizontal.Replicas == scale.Spec.Replicas {
		return withStatusUpdate(false, autoscaling.NoRequeue), nil
	}

	currentDesiredReplicas := scale.Spec.Replicas
	targetDesiredReplicas := *autoscalerInternal.ScalingValues.Horizontal.Replicas

	// Handling min/max replicas
	minReplicas := defaultMinReplicas
	if autoscalerInternal.Spec.Constraints != nil && autoscalerInternal.Spec.Constraints.MinReplicas != nil {
		minReplicas = *autoscalerInternal.Spec.Constraints.MinReplicas
	}

	maxReplicas := defaultMaxReplicas
	if autoscalerInternal.Spec.Constraints != nil && autoscalerInternal.Spec.Constraints.MaxReplicas >= minReplicas {
		maxReplicas = autoscalerInternal.Spec.Constraints.MaxReplicas
	}

	var scaleDirection scaleDirection
	if targetDesiredReplicas > currentDesiredReplicas {
		scaleDirection = scaleUp
	} else {
		scaleDirection = scaleDown
	}

	scale.Spec.Replicas = hr.computeScaleAction(currentDesiredReplicas, targetDesiredReplicas, minReplicas, maxReplicas, scaleDirection)
	_, err := hr.scaler.update(ctx, gr, scale)
	if err != nil {
		return withStatusUpdate(true, autoscaling.Requeue), fmt.Errorf("failed to update scale subresource for autoscaler %s, err: %w", autoscalerInternal.ID(), err)
	}

	return withStatusUpdate(true, autoscaling.NoRequeue), nil
}

func (hr *horizontalController) computeScaleAction(
	_, targetDesiredReplicas int32,
	minReplicas, maxReplicas int32,
	_ scaleDirection,
) int32 {
	// TODO: Implement the logic to compute the new number of replicas based on Policies
	if targetDesiredReplicas > maxReplicas {
		targetDesiredReplicas = maxReplicas
	} else if targetDesiredReplicas < minReplicas {
		targetDesiredReplicas = minReplicas
	}

	return targetDesiredReplicas
}
