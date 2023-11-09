// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

// Package proto holds protobuf encoding and decoding functions
package proto

import "time"

// EncodeTimestamp encodes a timestamp to its nanosecond representation
func EncodeTimestamp(t *time.Time) uint64 {
	if t.IsZero() {
		return 0
	}
	return uint64(t.UnixNano())
}

// DecodeTimestamp decodes a nanosecond representation of a timestamp
func DecodeTimestamp(nanos uint64) time.Time {
	return time.Unix(0, int64(nanos))
}
