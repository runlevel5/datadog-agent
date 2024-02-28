// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver

// Package merged implements a webhook that merges all the other webhooks into a
// single one and exposes them in a single endpoint
package merged

import (
	"context"

	admiv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilserror "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/dynamic"

	"github.com/DataDog/datadog-agent/cmd/cluster-agent/admission"
	"github.com/DataDog/datadog-agent/pkg/clusteragent/admission/mutate"
	agentsidecar "github.com/DataDog/datadog-agent/pkg/clusteragent/admission/mutate/agent_sidecar"
	"github.com/DataDog/datadog-agent/pkg/clusteragent/admission/mutate/autoinstrumentation"
	"github.com/DataDog/datadog-agent/pkg/clusteragent/admission/mutate/common"
	configWebhook "github.com/DataDog/datadog-agent/pkg/clusteragent/admission/mutate/config"
	"github.com/DataDog/datadog-agent/pkg/clusteragent/admission/mutate/cwsinstrumentation"
	"github.com/DataDog/datadog-agent/pkg/clusteragent/admission/mutate/tagsfromlabels"
	"github.com/DataDog/datadog-agent/pkg/config"
)

const webhookName = "merged"

// WebhookCollection contains the webhooks that are merged into the merged
// webhook
type WebhookCollection struct {
	Sidecar *agentsidecar.Webhook
	APM     *autoinstrumentation.Webhook
	Config  *configWebhook.Webhook
	CWS     *cwsinstrumentation.CWSInstrumentation
	Tags    *tagsfromlabels.Webhook
}

// Webhook is the merged webhook
type Webhook struct {
	name                 string
	isEnabled            bool
	endpoint             string
	resources            []string
	operations           []admiv1.OperationType
	useNamespaceSelector bool
	webhooks             map[mutate.MutatingWebhook]common.MutationFunc
}

// NewWebhook returns a new Webhook
func NewWebhook(webhooks WebhookCollection, useNamespaceSelector bool) *Webhook {
	w := &Webhook{
		name:                 webhookName,
		isEnabled:            config.Datadog.GetBool("admission_controller.merged.enabled"),
		endpoint:             config.Datadog.GetString("admission_controller.merged.endpoint"),
		resources:            []string{"pods"},
		operations:           []admiv1.OperationType{admiv1.Create},
		useNamespaceSelector: useNamespaceSelector,
	}

	w.webhooks = map[mutate.MutatingWebhook]common.MutationFunc{
		webhooks.Sidecar:              agentsidecar.InjectAgentSidecar,
		webhooks.APM:                  webhooks.APM.Inject,
		webhooks.Config:               webhooks.Config.Inject,
		webhooks.CWS.WebhookForPods(): webhooks.CWS.InjectCWSPodInstrumentation,
		webhooks.Tags:                 tagsfromlabels.InjectTags,
	}

	return w
}

// Name returns the name of the webhook
func (w *Webhook) Name() string {
	return w.name
}

// IsEnabled returns whether the webhook is enabled
func (w *Webhook) IsEnabled() bool {
	return w.isEnabled
}

// Endpoint returns the endpoint of the webhook
func (w *Webhook) Endpoint() string {
	return w.endpoint
}

// Resources returns the kubernetes resources for which the webhook should
// be invoked
func (w *Webhook) Resources() []string {
	return w.resources
}

// Operations returns the operations on the resources specified for which
// the webhook should be invoked
func (w *Webhook) Operations() []admiv1.OperationType {
	return w.operations
}

// LabelSelectors returns the label selectors that specify when the webhook
// should be invoked
func (w *Webhook) LabelSelectors(_ bool) (namespaceSelector *metav1.LabelSelector, objectSelector *metav1.LabelSelector) {
	// Need to accept everything and use the label selector for each webhook in
	// the inject function.
	return nil, nil
}

// MutateFunc returns the function that mutates the resources
func (w *Webhook) MutateFunc() admission.WebhookFunc {
	return w.mutate
}

func (w *Webhook) mutate(request *admission.MutateRequest) ([]byte, error) {
	return common.Mutate(request.Raw, request.Namespace, w.inject, request.DynamicClient)
}

func (w *Webhook) inject(pod *corev1.Pod, namespace string, dc dynamic.Interface) error {
	gvrNamespace := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}
	ns, errGetNamespace := dc.Resource(gvrNamespace).Get(context.TODO(), namespace, metav1.GetOptions{})
	nsLabels := ns.GetLabels()

	var errors []error

	for wh, mutationFunc := range w.webhooks {
		err := w.injectUsingWebhook(wh, mutationFunc, pod, namespace, nsLabels, errGetNamespace, dc)
		if err != nil {
			errors = append(errors, err)
		}
	}

	return utilserror.NewAggregate(errors)
}

func selectorsMatch(pod *corev1.Pod, namespaceSelector *metav1.LabelSelector, objSelector *metav1.LabelSelector, nsLabels map[string]string, nsError error) (bool, error) {
	if objSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(objSelector)
		if err != nil {
			return false, err
		}

		if !selector.Matches(labels.Set(pod.Labels)) {
			return false, nil
		}
	}

	if namespaceSelector != nil {
		if nsError != nil {
			return false, nsError
		}

		selector, err := metav1.LabelSelectorAsSelector(namespaceSelector)
		if err != nil {
			return false, err
		}

		if !selector.Matches(labels.Set(nsLabels)) {
			return false, nil
		}
	}

	return true, nil
}

func (w *Webhook) injectUsingWebhook(webhook mutate.MutatingWebhook, mutationFunc common.MutationFunc, pod *corev1.Pod, namespace string, nsLabels map[string]string, nsError error, dc dynamic.Interface) error {
	if webhook == nil || !webhook.IsEnabled() {
		return nil
	}

	nsSelector, objSelector := webhook.LabelSelectors(w.useNamespaceSelector)

	matches, err := selectorsMatch(pod, nsSelector, objSelector, nsLabels, nsError)
	if err != nil {
		return err
	}

	if !matches {
		return nil
	}

	return mutationFunc(pod, namespace, dc)
}
