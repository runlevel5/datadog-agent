// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2017-present Datadog, Inc.

//go:build kubeapiserver

package kubernetesapiserver

import (
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/patrickmn/go-cache"

	"github.com/DataDog/datadog-agent/comp/core/tagger"
	"github.com/DataDog/datadog-agent/comp/core/tagger/types"
	"github.com/DataDog/datadog-agent/pkg/metrics/event"
	"github.com/DataDog/datadog-agent/pkg/util/kubernetes"
	"github.com/DataDog/datadog-agent/pkg/util/kubernetes/apiserver"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

var (
	hostProviderIDCache *cache.Cache
)

type eventHostInfo struct {
	hostname   string
	nodename   string
	providerID string
}

// getDDAlertType converts kubernetes event types into datadog alert types
func getDDAlertType(k8sType string) event.EventAlertType {
	switch k8sType {
	case v1.EventTypeNormal:
		return event.EventAlertTypeInfo
	case v1.EventTypeWarning:
		return event.EventAlertTypeWarning
	default:
		log.Debugf("Unknown event type '%s'", k8sType)
		return event.EventAlertTypeInfo
	}
}

func getInvolvedObjectTags(involvedObject v1.ObjectReference, taggerInstance tagger.Component) []string {
	// NOTE: we now standardized on using kube_* tags, instead of
	// non-namespaced ones, or kubernetes_*. The latter two are now
	// considered deprecated.
	tags := []string{
		fmt.Sprintf("kube_kind:%s", involvedObject.Kind),
		fmt.Sprintf("kube_name:%s", involvedObject.Name),

		// DEPRECATED:
		fmt.Sprintf("kubernetes_kind:%s", involvedObject.Kind),
		fmt.Sprintf("name:%s", involvedObject.Name),
	}

	if involvedObject.Namespace != "" {
		tags = append(tags,
			fmt.Sprintf("kube_namespace:%s", involvedObject.Namespace),

			// DEPRECATED:
			fmt.Sprintf("namespace:%s", involvedObject.Namespace),
		)

		namespaceEntityId := fmt.Sprintf("namespace://%s", involvedObject.Namespace)
		namespaceEntity, err := taggerInstance.GetEntity(namespaceEntityId)
		if err == nil {
			tags = append(tags, namespaceEntity.GetTags(types.HighCardinality)...)
		}
	}

	kindTag := getKindTag(involvedObject.Kind, involvedObject.Name)
	if kindTag != "" {
		tags = append(tags, kindTag)
	}

	return tags
}

const (
	podKind  = "Pod"
	nodeKind = "Node"
)

func getEventHostInfo(clusterName string, ev *v1.Event) eventHostInfo {
	return getEventHostInfoImpl(getHostProviderID, clusterName, ev)
}

// getEventHostInfoImpl get the host information (hostname,nodename) from where the event has been generated.
// This function takes `hostProviderIDFunc` function to ease unit-testing by mocking the
// providers logic
//
//nolint:revive // TODO(CINT) Fix revive linter
func getEventHostInfoImpl(hostProviderIDFunc func(string) string, clusterName string, ev *v1.Event) eventHostInfo {
	info := eventHostInfo{}

	switch ev.InvolvedObject.Kind {
	case podKind:
		info.nodename = ev.Source.Host
		// works fine with Pod's events generated by the kubelet, but not with other
		// source like the draino controller.
		// We should be able to resolve this issue with the workloadmetadatastore
		// in the cluster-agent
	case nodeKind:
		// on Node the host is not always provided in the ev.Source.Host
		// But it is always available in `ev.InvolvedObject.Name`
		info.nodename = ev.InvolvedObject.Name
	default:
		return info
	}

	info.hostname = info.nodename
	if info.hostname != "" {
		info.providerID = getHostProviderID(info.hostname)

		if clusterName != "" {
			info.hostname += "-" + clusterName
		}
	}

	return info
}

func getHostProviderID(nodename string) string {
	if hostProviderID, hit := hostProviderIDCache.Get(nodename); hit {
		return hostProviderID.(string)
	}

	cl, err := apiserver.GetAPIClient()
	if err != nil {
		log.Warnf("Can't create client to query the API Server: %v", err)
		return ""
	}

	node, err := apiserver.GetNode(cl, nodename)
	if err != nil {
		log.Warnf("Can't get node from API Server: %v", err)
		return ""
	}

	providerID := node.Spec.ProviderID
	if providerID == "" {
		log.Warnf("ProviderID for nodename %q not found", nodename)
		return ""
	}

	// e.g. gce://datadog-test-cluster/us-east1-a/some-instance-id or
	// aws:///us-east-1e/i-instanceid
	s := strings.Split(providerID, "/")
	hostProviderID := s[len(s)-1]

	hostProviderIDCache.Set(nodename, hostProviderID, cache.DefaultExpiration)

	return hostProviderID
}

// getKindTag returns the kube_<kind>:<name> tag. The exact same tag names and
// object kinds are supported by the tagger. It returns an empty string if the
// kind doesn't correspond to a known/supported kind tag.
func getKindTag(kind, name string) string {
	if tagName, found := kubernetes.KindToTagName[kind]; found {
		return fmt.Sprintf("%s:%s", tagName, name)
	}
	return ""
}

func buildReadableKey(obj v1.ObjectReference) string {
	if obj.Namespace != "" {
		return fmt.Sprintf("%s %s/%s", obj.Kind, obj.Namespace, obj.Name)
	}

	return fmt.Sprintf("%s %s", obj.Kind, obj.Name)
}

func init() {
	hostProviderIDCache = cache.New(time.Hour, time.Hour)
}
