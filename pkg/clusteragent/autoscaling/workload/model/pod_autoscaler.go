// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver

package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/DataDog/datadog-agent/pkg/clusteragent/autoscaling"
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
	Spec *datadoghq.DatadogPodAutoscalerSpec

	// SettingsTimestamp is the time when the settings were last updated
	// Version is stored in .Spec.RemoteVersion
	// (only if owner == remote)
	SettingsTimestamp time.Time

	// ScalingValues represents the current target scaling values (retrieved from RC)
	ScalingValues ScalingValues

	// ScalingValuesHash is the hash of the values from RC
	ScalingValuesHash string

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
	// (only if owner == remote)
	Deleted bool

	//
	// Private fields
	//
	// targetGVK is the GroupVersionKind of the target resource
	// Parsed once from the .Spec.TargetRef
	targetGVK schema.GroupVersionKind
}

// NewPodAutoscalerInternal creates a new PodAutoscalerInternal from a Kubernetes CR
func NewPodAutoscalerInternal(podAutoscaler *datadoghq.DatadogPodAutoscaler) PodAutoscalerInternal {
	pai := PodAutoscalerInternal{
		Namespace: podAutoscaler.Namespace,
		Name:      podAutoscaler.Name,
	}
	pai.UpdateFromPodAutoscaler(podAutoscaler)
	pai.UpdateFromStatus(podAutoscaler)

	return pai
}

// NewPodAutoscalerFromSettings creates a new PodAutoscalerInternal from settings received through remote configuration
func NewPodAutoscalerFromSettings(ns, name string, podAutoscalerSpec *datadoghq.DatadogPodAutoscalerSpec, settingsVersion uint64, settingsTimestamp time.Time) PodAutoscalerInternal {
	pda := PodAutoscalerInternal{
		Namespace: ns,
		Name:      name,
	}
	pda.UpdateFromSettings(podAutoscalerSpec, settingsVersion, settingsTimestamp)

	return pda
}

// ID returns the functional identifier of the PodAutoscaler
func (p *PodAutoscalerInternal) ID() string {
	return p.Namespace + "/" + p.Name
}

// GetTargetGVK resolves the GroupVersionKind if empty and returns it
func (p *PodAutoscalerInternal) GetTargetGVK() (schema.GroupVersionKind, error) {
	if !p.targetGVK.Empty() {
		return p.targetGVK, nil
	}

	gv, err := schema.ParseGroupVersion(p.Spec.TargetRef.APIVersion)
	if err != nil {
		return schema.GroupVersionKind{}, fmt.Errorf("failed to parse API version %s: %w", p.Spec.TargetRef.APIVersion, err)
	}

	p.targetGVK = schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    p.Spec.TargetRef.Kind,
	}
	return p.targetGVK, nil
}

// UpdateFromPodAutoscaler updates the PodAutoscalerInternal from a PodAutoscaler object inside K8S
func (p *PodAutoscalerInternal) UpdateFromPodAutoscaler(podAutoscaler *datadoghq.DatadogPodAutoscaler) {
	p.Generation = podAutoscaler.Generation
	p.Spec = podAutoscaler.Spec.DeepCopy()
	// Reset the target GVK as it might have changed
	// Resolving the target GVK is done in the controller sync to ensure proper sync and error handling
	p.targetGVK = schema.GroupVersionKind{}
}

// UpdateFromValues updates the PodAutoscalerInternal from a new scaling values
func (p *PodAutoscalerInternal) UpdateFromValues(scalingValues ScalingValues, scalingValuesVersion uint64, scalingValuesTimestamp time.Time) error {
	p.ScalingValues = scalingValues
	p.ScalingValuesTimestamp = scalingValuesTimestamp
	p.ScalingValuesVersion = pointer.Ptr(scalingValuesVersion)

	valuesHash, err := autoscaling.ObjectHash(scalingValues)
	if err != nil {
		return err
	}

	p.ScalingValuesHash = valuesHash
	return nil
}

// RemoveValues clears autoscaling values data from the PodAutoscalerInternal as we stopped autoscaling
func (p *PodAutoscalerInternal) RemoveValues() {
	p.ScalingValues = ScalingValues{}
	p.ScalingValuesTimestamp = time.Time{}
	p.ScalingValuesVersion = nil
	p.ScalingValuesHash = ""
}

// UpdateFromSettings updates the PodAutoscalerInternal from a new settings
func (p *PodAutoscalerInternal) UpdateFromSettings(podAutoscalerSpec *datadoghq.DatadogPodAutoscalerSpec, settingsVersion uint64, settingsTimestamp time.Time) {
	p.SettingsTimestamp = settingsTimestamp
	p.Spec = podAutoscalerSpec // From settings, we don't need to deep copy as the object is not stored anywhere else
	p.Spec.RemoteVersion = pointer.Ptr(settingsVersion)
	// Reset the target GVK as it might have changed
	// Resolving the target GVK is done in the controller sync to ensure proper sync and error handling
	p.targetGVK = schema.GroupVersionKind{}
}

// UpdateFromStatus updates the PodAutoscalerInternal from an existing status.
func (p *PodAutoscalerInternal) UpdateFromStatus(podAutoscaler *datadoghq.DatadogPodAutoscaler) {
	if podAutoscaler.Status.RecommendationsVersion != nil {
		p.ScalingValuesVersion = pointer.Ptr(*podAutoscaler.Status.RecommendationsVersion)

		if podAutoscaler.Status.UpdateTime != nil {
			p.ScalingValuesTimestamp = podAutoscaler.Status.UpdateTime.Time
		}
	} else {
		p.ScalingValuesVersion = nil
		p.ScalingValuesTimestamp = time.Time{}
	}

	if podAutoscaler.Status.CurrentReplicas != nil {
		p.CurrentReplicas = podAutoscaler.Status.CurrentReplicas
	} else {
		p.CurrentReplicas = nil
	}

	if podAutoscaler.Status.Horizontal != nil {
		p.ScalingValues.Horizontal = &HorizontalScalingValues{
			Source:   podAutoscaler.Status.Horizontal.Source,
			Replicas: &podAutoscaler.Status.Horizontal.DesiredReplicas,
		}
		p.HorizontalLastAction = podAutoscaler.Status.Horizontal.LastAction
	} else {
		p.ScalingValues.Horizontal = nil
	}

	if podAutoscaler.Status.Vertical != nil {
		p.ScalingValues.Vertical = &VerticalScalingValues{
			Source:             podAutoscaler.Status.Vertical.Source,
			ContainerResources: podAutoscaler.Status.Vertical.DesiredResources,
		}
		p.VerticalLastAction = podAutoscaler.Status.Vertical.LastAction
	} else {
		p.ScalingValues.Vertical = nil
	}

	// Reading potential errors from conditions. Resetting internal errors first.
	// We're only keeping error string, loosing type, but it's not important for what we do.
	p.GlobalError = nil
	p.HorizontalLastActionError = nil
	p.VerticalRolloutError = nil

	for _, cond := range podAutoscaler.Status.Conditions {
		switch {
		case cond.Type == datadoghq.DatadogPodAutoscalerErrorCondition && cond.Status == corev1.ConditionTrue:
			p.GlobalError = errors.New(cond.Reason)
		case cond.Type == datadoghq.DatadogPodAutoscalerHorizontalAbleToScaleCondition && cond.Status == corev1.ConditionFalse:
			p.HorizontalLastActionError = errors.New(cond.Reason)
		case cond.Type == datadoghq.DatadogPodAutoscalerVerticalAbleToRollout && cond.Status == corev1.ConditionFalse:
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
	if p.ScalingValues.Horizontal != nil {
		status.Horizontal = &datadoghq.DatadogPodAutoscalerHorizontalStatus{
			DesiredReplicas: *p.ScalingValues.Horizontal.Replicas,
			LastAction:      p.HorizontalLastAction,
		}
	}

	// Produce Vertical status only if we have a desired container resources
	if p.ScalingValues.Vertical != nil {
		status.Vertical = &datadoghq.DatadogPodAutoscalerVerticalStatus{
			LastAction:       p.VerticalLastAction,
			DesiredResources: p.ScalingValues.Vertical.ContainerResources,
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
