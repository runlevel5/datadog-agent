// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

// Package proto holds protobuf encoding and decoding functions
package proto

import (
	adproto "github.com/DataDog/agent-payload/v5/cws/dumpsv1"
	"github.com/DataDog/datadog-agent/pkg/security/secprof/metadata"
)

// EncodeMetadata encodes a Metadata structure
func EncodeMetadata(meta *metadata.Metadata) *adproto.Metadata {
	if meta == nil {
		return nil
	}

	pmeta := &adproto.Metadata{
		AgentVersion:      meta.AgentVersion,
		AgentCommit:       meta.AgentCommit,
		KernelVersion:     meta.KernelVersion,
		LinuxDistribution: meta.LinuxDistribution,

		Name:              meta.Name,
		ProtobufVersion:   meta.ProtobufVersion,
		DifferentiateArgs: meta.DifferentiateArgs,
		Comm:              meta.Comm,
		ContainerId:       meta.ContainerID,
		Start:             EncodeTimestamp(&meta.Start),
		End:               EncodeTimestamp(&meta.End),
		Size:              meta.Size,
		Arch:              meta.Arch,
		Serialization:     meta.Serialization,
	}

	return pmeta
}

// DecodeMetadata decodes a Metadata structure
func DecodeMetadata(meta *adproto.Metadata) metadata.Metadata {
	if meta == nil {
		return metadata.Metadata{}
	}

	return metadata.Metadata{
		AgentVersion:      meta.AgentVersion,
		AgentCommit:       meta.AgentCommit,
		KernelVersion:     meta.KernelVersion,
		LinuxDistribution: meta.LinuxDistribution,
		Arch:              meta.Arch,

		Name:              meta.Name,
		ProtobufVersion:   meta.ProtobufVersion,
		DifferentiateArgs: meta.DifferentiateArgs,
		Comm:              meta.Comm,
		ContainerID:       meta.ContainerId,
		Start:             DecodeTimestamp(meta.Start),
		End:               DecodeTimestamp(meta.End),
		Size:              meta.Size,
		Serialization:     meta.GetSerialization(),
	}
}
