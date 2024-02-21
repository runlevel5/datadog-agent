package types

import (
	pb "github.com/DataDog/datadog-agent/pkg/proto/pbgo/trace"
	serverlessLogs "github.com/DataDog/datadog-agent/pkg/serverless/logs"
	"github.com/DataDog/datadog-agent/pkg/serverless/plugin"
	"github.com/DataDog/datadog-agent/pkg/trace/config"
)

// Load abstracts the file configuration loading
type Load interface {
	Load() (*config.AgentConfig, error)
}

type ColdStartSpanArgs struct {
	LambdaSpanChan       chan<- *pb.Span
	LambdaInitMetricChan chan *serverlessLogs.LambdaInitMetric
	TraceAgent           plugin.Plugin
	StopChan             chan struct{}
	ColdStartSpanId      uint64
}

type Args struct {
	LoadConfig      Load
	LambdaSpanChan  chan<- *pb.Span
	ColdStartSpanId uint64
}
