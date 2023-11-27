// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

// Package serverlessmetric ... /* TODO: detailed doc comment for the component */
package serverlessmetric

// team: serverless

// Component is the component type.
type Component interface {
	GetExtraTags() []string
	Flush()
	IsReady() bool
	SetExtraTags([]string)
	Start()
	Stop()
}
