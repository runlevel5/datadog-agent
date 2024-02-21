package main

import (
	"github.com/DataDog/datadog-agent/pkg/serverless/plugin"
	"github.com/DataDog/datadog-agent/pkg/serverless/trace"
)

func main() {
	// Empty
}

// NewAgent returns a ServerlessTraceAgent
func Build() plugin.Plugin {
	return &trace.ServerlessTraceAgent{}
}

// BuildColdStartSpanCreator returns a ColdStartSpanCreator
func BuildColdStartSpanCreator() plugin.Plugin {
	return &trace.ColdStartSpanCreator{}
}
