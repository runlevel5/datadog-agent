// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build unix

//go:generate go run github.com/DataDog/datadog-agent/pkg/security/secl/compiler/generators/accessors -tags unix -types-file model.go -output accessors_unix.go -field-handlers field_handlers_unix.go -doc ../../../../docs/cloud-workload-security/secl.json -field-accessors-output field_accessors_unix.go

// Package model holds model related files
package model

import (
	"time"

	"github.com/DataDog/datadog-agent/pkg/security/secl/compiler/eval"
)
