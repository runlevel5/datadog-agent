// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver

package model

import (
	"errors"
	"time"

	"github.com/DataDog/datadog-agent/pkg/util/pointer"
	datadoghq "github.com/DataDog/datadog-operator/apis/datadoghq/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// UnsetGeneration is the value used to represent a .Spec value for which we have no generation (not created or not updated in-cluster yet)
const UnsetGeneration = -1

// PodAutoscalerInternal hols the necessary data to work with the `DatadogPodAutoscaler` CRD.
type PodAutoscalerInternal struct {
	// Namespace is the namespace of the PodAutoscaler
	Namespace string

	// Name is the name of the PodAutoscaler
	Name string

	// Generation is the received generation of the PodAutoscaler
	Generation int64

	// Keeping track of .Spec (configuration of the Autoscaling)
	Spec datadoghq.DatadogPodAutoscalerSpec

	// TargetGVK is the GroupVersionKind of the target resource
	// Parsed once from the .Spec.TargetRef
	TargetGVK schema.GroupVersionKind

	// SettingsVersion is the version of the settings from RC
	// (only if owner == remote)
	SettingsVersion *uint64

	// SettingsTimestamp is the time when the settings were last updated
	// (only if owner == remote)
	SettingsTimestamp time.Time

	// Target represents the desired state
	ScalingValues ScalingValues

	// ScalingValuesVersion is the version of the values from RC
	ScalingValuesVersion *uint64

	// ScalingValuesTimestamp is the time when the values were last updated
	ScalingValuesTimestamp time.Time

	// HorizontalLastAction is the last horizontal action successfully taken
	HorizontalLastAction *datadoghq.DatadogPodAutoscalerHorizontalAction

	// HorizontalLastActionError is the last error encountered on horizontal scaling
	HorizontalLastActionError error

	// VerticalLastAction is the last action taken by the Vertical Pod Autoscaler
	VerticalLastAction *datadoghq.DatadogPodAutoscalerVerticalAction

	// VerticalRolloutError is the last error encountered on vertical scaling
	VerticalRolloutError error

	// CurrentReplicas is the current number of PODs for the targetRef
	CurrentReplicas *int32

	// ScaledReplicas is the current number of PODs for the targetRef matching the resources recommendations
	ScaledReplicas *int32

	// GlobalError is the an error encountered by the controller not specific to a scaling action
	GlobalError error

	// Deleted flags the PodAutoscaler as deleted (removal to be handled by the controller)
	Deleted bool
}

// NewPodAutoscalerInternal creates a new PodAutoscalerInternal from a Kubernetes CR
func NewPodAutoscalerInternal(_ string, podAutoscaler *datadoghq.DatadogPodAutoscaler) PodAutoscalerInternal {
	pai := PodAutoscalerInternal{
		Namespace:  podAutoscaler.Namespace,
		Name:       podAutoscaler.Name,
		Generation: podAutoscaler.Generation,
		Spec:       podAutoscaler.Spec,
	}
	pai.UpdateFromStatus(podAutoscaler)
	return pai
}

// ID returns the functional identifier of the PodAutoscaler
func (p *PodAutoscalerInternal) ID() string {
	return p.Namespace + "/" + p.Name
}

// UpdateFromSpec updates the PodAutoscalerInternal from a new spec
func (p *PodAutoscalerInternal) UpdateFromSpec(podAutoscalerSpec *datadoghq.DatadogPodAutoscalerSpec) {
	podAutoscalerSpec.DeepCopyInto(&p.Spec)
	// Reset the target GVK as it might have changed
	p.TargetGVK = schema.GroupVersionKind{}
}

// UpdateFromSettings updates the PodAutoscalerInternal from a new settings
func (p *PodAutoscalerInternal) UpdateFromSettings(podAutoscalerSpec *datadoghq.DatadogPodAutoscalerSpec, settingsVersion uint64, settingsTimestamp time.Time) {
	p.SettingsVersion = pointer.Ptr(settingsVersion)
	p.SettingsTimestamp = settingsTimestamp

	podAutoscalerSpec.DeepCopyInto(&p.Spec)
	podAutoscalerSpec.RemoteVersion = settingsVersion

	// Reset the target GVK as it might have changed
	p.TargetGVK = schema.GroupVersionKind{}
}

// UpdateFromStatus updates the PodAutoscalerInternal from an existing status.
func (p *PodAutoscalerInternal) UpdateFromStatus(podAutoscaler *datadoghq.DatadogPodAutoscaler) {
	if podAutoscaler.Status.RecommendationsVersion != nil {
		p.ScalingValuesVersion = pointer.Ptr(*podAutoscaler.Status.RecommendationsVersion)
		p.ScalingValuesTimestamp = podAutoscaler.Status.UpdateTime.Time
	}

	if podAutoscaler.Status.CurrentReplicas != nil {
		p.CurrentReplicas = podAutoscaler.Status.CurrentReplicas
	}

	if podAutoscaler.Status.Horizontal != nil {
		p.ScalingValues.Replicas = &podAutoscaler.Status.Horizontal.DesiredReplicas
		p.HorizontalLastAction = podAutoscaler.Status.Horizontal.LastAction
	}

	if podAutoscaler.Status.Vertical != nil {
		p.ScalingValues.ContainerResources = podAutoscaler.Status.Vertical.DesiredResources
		p.VerticalLastAction = podAutoscaler.Status.Vertical.LastAction
	}

	// Reading potential errors from conditions.
	// We're only keeping error string, loosing type, but it's not important for what we do.
	for _, cond := range podAutoscaler.Status.Conditions {
		if cond.Type == datadoghq.DatadogPodAutoscalerHorizontalAbleToScaleCondition && cond.Status == corev1.ConditionFalse {
			p.HorizontalLastActionError = errors.New(cond.Reason)
		} else if cond.Type == datadoghq.DatadogPodAutoscalerVerticalAbleToRollout && cond.Status == corev1.ConditionFalse {
			p.VerticalRolloutError = errors.New(cond.Reason)
		}
	}
}

// BuildStatus builds the status of the PodAutoscaler from the internal state
func (p *PodAutoscalerInternal) BuildStatus(currentTime metav1.Time, currentStatus *datadoghq.DatadogPodAutoscalerStatus) datadoghq.DatadogPodAutoscalerStatus {
	status := datadoghq.DatadogPodAutoscalerStatus{}
	if p.ScalingValuesVersion != nil {
		status.RecommendationsVersion = p.ScalingValuesVersion
		status.UpdateTime = pointer.Ptr(metav1.NewTime(p.ScalingValuesTimestamp))
	}

	if p.CurrentReplicas != nil {
		status.CurrentReplicas = p.CurrentReplicas
	}

	// Produce Horizontal status only if we have a desired number of replicas
	if p.ScalingValues.Replicas != nil {
		status.Horizontal = &datadoghq.DatadogPodAutoscalerHorizontalStatus{
			DesiredReplicas: *p.ScalingValues.Replicas,
			LastAction:      p.HorizontalLastAction,
		}
	}

	// Produce Vertical status only if we have a desired container resources
	if len(p.ScalingValues.ContainerResources) > 0 {
		status.Vertical = &datadoghq.DatadogPodAutoscalerVerticalStatus{
			LastAction:       p.VerticalLastAction,
			DesiredResources: p.ScalingValues.ContainerResources,
		}
	}

	// Building conditions
	existingConditions := map[datadoghq.DatadogPodAutoscalerConditionType]*datadoghq.DatadogPodAutoscalerCondition{
		datadoghq.DatadogPodAutoscalerHorizontalAbleToScaleCondition: nil,
		datadoghq.DatadogPodAutoscalerVerticalAbleToRollout:          nil,
		datadoghq.DatadogPodAutoscalerErrorCondition:                 nil,
	}

	if currentStatus != nil {
		for i := range currentStatus.Conditions {
			condition := &currentStatus.Conditions[i]
			if _, ok := existingConditions[condition.Type]; ok {
				existingConditions[condition.Type] = condition
			}
		}
	}

	var errorReason string
	errorStatus := corev1.ConditionFalse
	if p.GlobalError != nil {
		errorStatus = corev1.ConditionTrue
		errorReason = p.GlobalError.Error()
	}

	var horizontalReason string
	horizontalStatus := corev1.ConditionUnknown
	if p.HorizontalLastActionError != nil {
		horizontalStatus = corev1.ConditionFalse
		horizontalReason = p.HorizontalLastActionError.Error()
	} else if p.HorizontalLastAction != nil {
		horizontalStatus = corev1.ConditionTrue
	}

	var verticalReason string
	rolloutStatus := corev1.ConditionUnknown
	if p.VerticalRolloutError != nil {
		rolloutStatus = corev1.ConditionFalse
		verticalReason = p.VerticalRolloutError.Error()
	} else if p.VerticalLastAction != nil {
		rolloutStatus = corev1.ConditionTrue
	}

	errorCondition := p.newCondition(errorStatus, errorReason, currentTime, datadoghq.DatadogPodAutoscalerErrorCondition, existingConditions[datadoghq.DatadogPodAutoscalerErrorCondition])
	ableToScaleCondition := p.newCondition(horizontalStatus, horizontalReason, currentTime, datadoghq.DatadogPodAutoscalerHorizontalAbleToScaleCondition, existingConditions[datadoghq.DatadogPodAutoscalerHorizontalAbleToScaleCondition])
	ableToRolloutCondition := p.newCondition(rolloutStatus, verticalReason, currentTime, datadoghq.DatadogPodAutoscalerVerticalAbleToRollout, existingConditions[datadoghq.DatadogPodAutoscalerVerticalAbleToRollout])
	status.Conditions = []datadoghq.DatadogPodAutoscalerCondition{errorCondition, ableToScaleCondition, ableToRolloutCondition}

	return status
}

func (p *PodAutoscalerInternal) newCondition(status corev1.ConditionStatus, reason string, currentTime metav1.Time, conditionType datadoghq.DatadogPodAutoscalerConditionType, prevCondition *datadoghq.DatadogPodAutoscalerCondition) datadoghq.DatadogPodAutoscalerCondition {
	condition := datadoghq.DatadogPodAutoscalerCondition{
		Type:   conditionType,
		Status: status,
		Reason: reason,
	}

	if prevCondition == nil || (prevCondition.Status != condition.Status) {
		condition.LastTransitionTime = currentTime
	} else {
		condition.LastTransitionTime = prevCondition.LastTransitionTime
	}

	return condition
}
