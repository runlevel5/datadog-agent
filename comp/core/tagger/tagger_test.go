// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package tagger

import (
	"testing"

	"github.com/stretchr/testify/assert"

	collectortypes "github.com/DataDog/datadog-agent/comp/core/tagger/collectors/types"
	"github.com/DataDog/datadog-agent/pkg/tagset"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
)

func Test_taggerCardinality(t *testing.T) {
	tests := []struct {
		name        string
		cardinality string
		want        collectortypes.TagCardinality
	}{
		{
			name:        "high",
			cardinality: "high",
			want:        collectortypes.HighCardinality,
		},
		{
			name:        "orchestrator",
			cardinality: "orchestrator",
			want:        collectortypes.OrchestratorCardinality,
		},
		{
			name:        "orch",
			cardinality: "orch",
			want:        collectortypes.OrchestratorCardinality,
		},
		{
			name:        "low",
			cardinality: "low",
			want:        collectortypes.LowCardinality,
		},
		{
			name:        "empty",
			cardinality: "",
			want:        DogstatsdCardinality,
		},
		{
			name:        "unknown",
			cardinality: "foo",
			want:        DogstatsdCardinality,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, taggerCardinality(tt.cardinality))
		})
	}
}

func TestEnrichTagsOrchestrator(t *testing.T) {
	fakeTagger := fxutil.Test[Mock](t, MockModule())
	defer fakeTagger.ResetTagger()
	fakeTagger.SetTags("foo", "fooSource", []string{"lowTag"}, []string{"orchTag"}, nil, nil)
	tb := tagset.NewHashingTagsAccumulator()
	EnrichTags(tb, "foo", "", "orchestrator")
	assert.Equal(t, []string{"lowTag", "orchTag"}, tb.Get())
}
