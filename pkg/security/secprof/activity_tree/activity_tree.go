// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

// Package activitytree holds activity tree related files
package activitytree

import (
	"github.com/DataDog/datadog-agent/pkg/security/secl/model"
	"github.com/DataDog/datadog-agent/pkg/security/utils"
)

// ActivityTree contains a process tree and its activities. This structure has no locks.
type ActivityTree struct {
	differentiateArgs bool
	DNSMatchMaxDepth  int

	ProcessNodes []*ProcessNode `json:"-"`

	// top level lists used to summarize the content of the tree
	DNSNames     *utils.StringKeys
	SyscallsMask map[int]int
}

// Owner is used to communicate with the owner of the activity tree
type Owner interface {
	MatchesSelector(entry *model.ProcessCacheEntry) bool
	IsEventTypeValid(evtType model.EventType) bool
	NewProcessNodeCallback(p *ProcessNode)
}
