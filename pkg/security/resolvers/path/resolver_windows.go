// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build windows

// Package path holds path related files
package path

import "regexp"

type Resolver struct {
}

var devicePathRe = regexp.MustCompile(`(?i)^\\device\\harddiskvolume\d+\\`)

func NewPathResolver() *Resolver {
	return &Resolver{}
}

func (r *Resolver) ResolveUserPath(devicePath string) string {
	return devicePathRe.ReplaceAllString(devicePath, "C:\\")
}
