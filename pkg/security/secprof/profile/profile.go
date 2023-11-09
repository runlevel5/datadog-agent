// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

// Package profile holds profile related files
package profile

import (
	activitytree "github.com/DataDog/datadog-agent/pkg/security/secprof/activity_tree"
	"github.com/DataDog/datadog-agent/pkg/security/secprof/metadata"
)

type Profile struct {
	metadata          metadata.Metadata
	Host              string                     `json:"host,omitempty"`
	Service           string                     `json:"service,omitempty"`
	Source            string                     `json:"ddsource,omitempty"`
	Tags              []string                   `json:"tags,omitempty"`
	activityTree      *activitytree.ActivityTree `json:"-"`
	activityTreeStats *Stats

	Status  uint32 `json:"status,omitempty"`
	Version string `json:"version,omitempty"`
}
