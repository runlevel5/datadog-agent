// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

// Package storage holds storage related files
package storage

import (
	"fmt"
	"io"
	"os"

	"github.com/DataDog/datadog-agent/pkg/security/utils"
)

// LoadProfileFromFile loads profile from file
func LoadProfileFromFile(filepath string) (*proto.SecurityProfile, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("couldn't open profile: %w", err)
	}
	defer f.Close()

	raw, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("couldn't open profile: %w", err)
	}

	profile := &proto.SecurityProfile{}
	if err = profile.UnmarshalVT(raw); err != nil {
		return nil, fmt.Errorf("couldn't decode protobuf profile: %w", err)
	}

	if len(utils.GetTagValue("image_tag", profile.Tags)) == 0 {
		profile.Tags = append(profile.Tags, "image_tag:latest")
	}
	return profile, nil
}
