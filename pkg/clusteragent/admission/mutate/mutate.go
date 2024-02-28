// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver

// Package mutate defines the interface that all the mutating webhooks used in
// the admission controller must implement
package mutate

import (
	admiv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/DataDog/datadog-agent/cmd/cluster-agent/admission"
)

// MutatingWebhook represents a mutating webhook
type MutatingWebhook interface {
	// Name returns the name of the webhook
	Name() string
	// IsEnabled returns whether the webhook is enabled
	IsEnabled() bool
	// Endpoint returns the endpoint of the webhook
	Endpoint() string
	// Resources returns the kubernetes resources for which the webhook should
	// be invoked
	Resources() []string
	// Operations returns the operations on the resources specified for which
	// the webhook should be invoked
	Operations() []admiv1.OperationType
	// LabelSelectors returns the label selectors that specify when the webhook
	// should be invoked
	LabelSelectors(useNamespaceSelector bool) (namespaceSelector *metav1.LabelSelector, objectSelector *metav1.LabelSelector)
	// MutateFunc returns the function that mutates the resources
	MutateFunc() admission.WebhookFunc
}
