// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021-present Datadog, Inc.

//go:build !kubeapiserver

//nolint:revive // TODO(CINT) Fix revive linter
package kubernetesapiserver

import "github.com/DataDog/datadog-agent/pkg/collector/check"

const (
	Enabled   = false
	CheckName = "kube_apiserver_controlplane"
)

func Factory() check.Check {
	return nil
}
