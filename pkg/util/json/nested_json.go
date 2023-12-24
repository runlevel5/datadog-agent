// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

// Package json implements helper functions to interact with json
package json

import (
	"encoding/json"
	"strings"

	"github.com/bhmj/jsonslice"
)

// GetNestedValue returns the value in the map specified by the array keys,
// where each value is another depth level in the map.
// Returns nil if the map doesn't contain the nested key.
func GetNestedValue(data []byte, keys ...string) interface{} {
	// XXX: This change makes the code +103.36% slower,
	// but reduces mem by -47.68% and allocs by -42.14%
	jsonpath := strings.Join(append([]string{"$"}, keys...), ".")
	raw, _ := jsonslice.Get(data, jsonpath)

	var val interface{}
	json.Unmarshal(raw, &val)
	return val
}
