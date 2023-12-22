// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

// Package json implements helper functions to interact with json
package json

import "strings"

// GetNestedValue returns the value in the map specified by the array keys,
// where each value is another depth level in the map.
// Returns nil if the map doesn't contain the nested key.
func GetNestedValue(inputMap map[string]interface{}, keys ...string) interface{} {
	var val interface{}
	var exists bool
	for k, v := range inputMap {
		if strings.ToLower(k) == keys[0] {
			val = v
			exists = true
			break
		}
	}
	if !exists {
		return nil
	}
	if len(keys) == 1 {
		return val
	}
	innerMap, ok := val.(map[string]interface{})
	if !ok {
		return nil
	}
	return GetNestedValue(innerMap, keys[1:]...)
}
