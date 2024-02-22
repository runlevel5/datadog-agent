// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build serverless

package aggregator

import (
	collectorstypes "github.com/DataDog/datadog-agent/comp/core/tagger/collectors/types"
	"github.com/DataDog/datadog-agent/pkg/tagset"
)

func enrichTags(tb tagset.TagsAccumulator, udsOrigin string, clientOrigin string, cardinalityName string) {
	// nothing to do here
}

func agentTags(cardinality collectorstypes.TagCardinality) ([]string, error) {
	return []string{}, nil
}

func globalTags(cardinality collectorstypes.TagCardinality) ([]string, error) {
	return []string{}, nil
}

func checkCardinality() collectorstypes.TagCardinality {
	return 0
}
