// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver && test

// Package model implements data model structures and helpers for workload autoscaling.
package model

import (
	"reflect"

	"github.com/DataDog/datadog-agent/pkg/clusteragent/autoscaling"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// ComparePodAutoscalers compares two PodAutoscalerInternal objects with cmp.Diff.
func ComparePodAutoscalers(expected, actual any) string {
	return cmp.Diff(
		expected, actual,
		cmpopts.SortSlices(func(a, b PodAutoscalerInternal) bool {
			return autoscaling.BuildObjectID(a.Namespace, a.Name) < autoscaling.BuildObjectID(b.Namespace, b.Name)
		}),
		cmp.Exporter(func(t reflect.Type) bool {
			return t == reflect.TypeOf(PodAutoscalerInternal{})
		}),
	)
}
