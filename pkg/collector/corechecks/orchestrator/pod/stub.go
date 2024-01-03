// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build !kubelet && !orchestrator

// Package pod is used for the orchestrator pod check
package pod

import "github.com/DataDog/datadog-agent/pkg/collector/check"

const Enabled = false
const CheckName = "orchestrator_pod"

func Factory() check.Check {
	return nil
}
