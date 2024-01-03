// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.
//go:build windows

//nolint:revive // TODO(PLINT) Fix revive linter
package disk

import "github.com/DataDog/datadog-agent/pkg/collector/check"

const (
	Enabled   = false
	CheckName = "disk"
)

func Factory() check.Check {
	return nil
}
