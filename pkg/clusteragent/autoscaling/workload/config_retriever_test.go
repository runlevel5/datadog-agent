// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver

package workload

import (
	"encoding/json"
	"testing"

	"github.com/DataDog/datadog-agent/pkg/clusteragent/autoscaling"
	"github.com/DataDog/datadog-agent/pkg/clusteragent/autoscaling/workload/model"
	"github.com/DataDog/datadog-agent/pkg/config/remote/data"
	"github.com/DataDog/datadog-agent/pkg/remoteconfig/state"
	"github.com/DataDog/datadog-agent/pkg/util/pointer"
	datadoghq "github.com/DataDog/datadog-operator/apis/datadoghq/v1alpha1"

	"github.com/stretchr/testify/assert"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
)

type rcCallbackFunc func(map[string]state.RawConfig, func(string, state.ApplyStatus))

type mockRCClient struct {
	subscribers map[string][]rcCallbackFunc
}

func (m *mockRCClient) Subscribe(product string, callback func(map[string]state.RawConfig, func(string, state.ApplyStatus))) {
	if m.subscribers == nil {
		m.subscribers = make(map[string][]rcCallbackFunc)
	}
	m.subscribers[product] = append(m.subscribers[product], callback)
}

func (m *mockRCClient) triggerUpdate(product string, update map[string]state.RawConfig, applyStateCallback func(string, state.ApplyStatus)) {
	callbacks := m.subscribers[product]

	for _, callback := range callbacks {
		callback(update, applyStateCallback)
	}
}

func TestConfigRetriverAutoscalingSettingsLeader(t *testing.T) {
	cr, mockRCClient := newMockConfigRetriever(t, true)

	object1Spec := datadoghq.DatadogPodAutoscalerSpec{
		Owner: datadoghq.DatadogPodAutoscalerRemoteOwner,
		TargetRef: autoscalingv2.CrossVersionObjectReference{
			APIVersion: "v1",
			Kind:       "Deployment",
			Name:       "name1",
		},
	}
	object2Spec := datadoghq.DatadogPodAutoscalerSpec{
		Owner: datadoghq.DatadogPodAutoscalerRemoteOwner,
		TargetRef: autoscalingv2.CrossVersionObjectReference{
			APIVersion: "v1",
			Kind:       "Deployment",
			Name:       "name2",
		},
	}
	object3Spec := datadoghq.DatadogPodAutoscalerSpec{
		Owner: datadoghq.DatadogPodAutoscalerRemoteOwner,
		TargetRef: autoscalingv2.CrossVersionObjectReference{
			APIVersion: "v1",
			Kind:       "Deployment",
			Name:       "name3",
		},
	}

	t.Run("new autoscalingsettings received", func(t *testing.T) {
		stateCallbackCalled := 0

		mockRCClient.triggerUpdate(
			data.ProductContainerAutoscalingSettings,
			map[string]state.RawConfig{
				"foo1": buildAutoscalingSettingsRawConfig(t, 1, model.AutoscalingSettingsList{
					CluterID: "fake",
					Settings: []model.AutoscalingSettings{
						{
							ID:   "ns/name1",
							Spec: object1Spec,
						},
						{
							ID:   "ns/name2",
							Spec: object2Spec,
						},
					},
				}),
				"foo2": buildAutoscalingSettingsRawConfig(t, 10, model.AutoscalingSettingsList{
					CluterID: "fake",
					Settings: []model.AutoscalingSettings{
						{
							ID:   "ns/name3",
							Spec: object3Spec,
						},
					},
				}),
			},
			func(configKey string, applyState state.ApplyStatus) {
				stateCallbackCalled++
				assert.Equal(t, applyState, state.ApplyStatus{
					State: state.ApplyStateAcknowledged,
					Error: "",
				})
			},
		)

		assert.Equal(t, 2, stateCallbackCalled)
		podAutoscalers := cr.store.GetAll()

		assert.Empty(t, model.ComparePodAutoscalers([]model.PodAutoscalerInternal{
			{
				Namespace:       "ns",
				Name:            "name1",
				Spec:            object1Spec,
				SettingsVersion: pointer.Ptr[uint64](1),
			},
			{
				Namespace:       "ns",
				Name:            "name2",
				Spec:            object2Spec,
				SettingsVersion: pointer.Ptr[uint64](1),
			},
			{
				Namespace:       "ns",
				Name:            "name3",
				Spec:            object3Spec,
				SettingsVersion: pointer.Ptr[uint64](10),
			},
		}, podAutoscalers))
	})

	t.Run("update to existing autoscalingsettings received", func(t *testing.T) {
		// Update the settings for object3
		object3Spec.Recommender = &datadoghq.DatadogPodAutoscalerRecommender{
			Name: "some-option",
		}

		stateCallbackCalled := 0
		mockRCClient.triggerUpdate(
			data.ProductContainerAutoscalingSettings,
			map[string]state.RawConfig{
				"foo2": buildAutoscalingSettingsRawConfig(t, 11, model.AutoscalingSettingsList{
					CluterID: "fake",
					Settings: []model.AutoscalingSettings{
						{
							ID:   "ns/name3",
							Spec: object3Spec,
						},
					},
				}),
			},
			func(configKey string, applyState state.ApplyStatus) {
				stateCallbackCalled++
				assert.Equal(t, applyState, state.ApplyStatus{
					State: state.ApplyStateAcknowledged,
					Error: "",
				})
			},
		)

		assert.Equal(t, 1, stateCallbackCalled)
		podAutoscalers := cr.store.GetAll()

		assert.Empty(t, model.ComparePodAutoscalers(podAutoscalers, []model.PodAutoscalerInternal{
			{
				Namespace:       "ns",
				Name:            "name1",
				Spec:            object1Spec,
				SettingsVersion: pointer.Ptr[uint64](1),
			},
			{
				Namespace:       "ns",
				Name:            "name2",
				Spec:            object2Spec,
				SettingsVersion: pointer.Ptr[uint64](1),
			},
			{
				Namespace:       "ns",
				Name:            "name3",
				Spec:            object3Spec,
				SettingsVersion: pointer.Ptr[uint64](11),
			},
		}))
	})
}

func newMockConfigRetriever(t *testing.T, isLeader bool) (*configRetriever, *mockRCClient) {
	t.Helper()

	store := autoscaling.NewStore[model.PodAutoscalerInternal]()
	mockRCClient := &mockRCClient{}

	cr, err := newConfigRetriever(store, func() bool { return isLeader }, mockRCClient)
	assert.NoError(t, err)

	return cr, mockRCClient
}

func buildAutoscalingSettingsRawConfig(t *testing.T, version uint64, autoscalingSettings model.AutoscalingSettingsList) state.RawConfig {
	t.Helper()

	content, err := json.Marshal(autoscalingSettings)
	assert.NoError(t, err)

	return buildRawConfig(t, data.ProductContainerAutoscalingSettings, version, content)
}

func buildRawConfig(t *testing.T, product string, version uint64, content []byte) state.RawConfig {
	t.Helper()

	return state.RawConfig{
		Metadata: state.Metadata{
			Product: product,
			Version: version,
		},
		Config: content,
	}
}
