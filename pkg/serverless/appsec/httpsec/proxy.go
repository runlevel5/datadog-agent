// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package httpsec

import (
	"bytes"

	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/serverless/invocationlifecycle"
	serverlessMetrics "github.com/DataDog/datadog-agent/pkg/serverless/metrics"
	"github.com/DataDog/datadog-agent/pkg/serverless/trigger"

	//"github.com/DataDog/datadog-agent/pkg/trace/sampler"
	"github.com/DataDog/datadog-agent/pkg/util/log"

	"github.com/aws/aws-lambda-go/events"
	json "github.com/json-iterator/go"
)

// ProxyLifecycleProcessor is an implementation of the invocationlifecycle.InvocationProcessor
// interface called by the Runtime API proxy on every function invocation calls and responses.
// This allows AppSec to run by monitoring the function invocations, and run the security
// rules upon reception of the HTTP request span in the SpanModifier function created by
// the WrapSpanModifier() method.
// A value of this type can be used by a single function invocation at a time.
type ProxyLifecycleProcessor struct {
	// AppSec instance
	appsec Monitorer

	// Parsed invocation event value
	invocationEvent interface{}

	demux aggregator.Demultiplexer
}

// NewProxyLifecycleProcessor returns a new httpsec proxy processor monitored with the
// given Monitorer.
func NewProxyLifecycleProcessor(appsec Monitorer, demux aggregator.Demultiplexer) *ProxyLifecycleProcessor {
	return &ProxyLifecycleProcessor{
		appsec: appsec,
		demux:  demux,
	}
}

//nolint:revive // TODO(ASM) Fix revive linter
func (lp *ProxyLifecycleProcessor) GetExecutionInfo() *invocationlifecycle.ExecutionStartInfo {
	return nil // not used in the runtime api proxy case
}

// OnInvokeStart is the hook triggered when an invocation has started
func (lp *ProxyLifecycleProcessor) OnInvokeStart(startDetails *invocationlifecycle.InvocationStartDetails) {
	log.Debugf("appsec: proxy-lifecycle: invocation started with raw payload `%s`", startDetails.InvokeEventRawPayload)

	payloadBytes := invocationlifecycle.ParseLambdaPayload(startDetails.InvokeEventRawPayload)
	log.Debugf("Parsed payload string: %s", bytesStringer(payloadBytes))

	lowercaseEventPayload, err := trigger.Unmarshal(bytes.ToLower(payloadBytes))
	if err != nil {
		log.Debugf("appsec: proxy-lifecycle: Failed to parse event payload: %v", err)
	}

	eventType := trigger.GetEventType(lowercaseEventPayload)
	if eventType == trigger.Unknown {
		log.Debugf("appsec: proxy-lifecycle: Failed to extract event type")
	} else {
		log.Debugf("appsec: proxy-lifecycle: Extracted event type: %v", eventType)
	}

	var event interface{}
	switch eventType {
	default:
		log.Debugf("appsec: proxy-lifecycle: ignoring unsupported lambda event type %v", eventType)
		return
	case trigger.APIGatewayEvent:
		event = &events.APIGatewayProxyRequest{}
	case trigger.APIGatewayV2Event:
		event = &events.APIGatewayV2HTTPRequest{}
	case trigger.APIGatewayWebsocketEvent:
		event = &events.APIGatewayWebsocketProxyRequest{}
	case trigger.APIGatewayLambdaAuthorizerTokenEvent:
		event = &events.APIGatewayCustomAuthorizerRequest{}
	case trigger.APIGatewayLambdaAuthorizerRequestParametersEvent:
		event = &events.APIGatewayCustomAuthorizerRequestTypeRequest{}
	case trigger.ALBEvent:
		event = &events.ALBTargetGroupRequest{}
	case trigger.LambdaFunctionURLEvent:
		event = &events.LambdaFunctionURLRequest{}
	}
	if lp.demux != nil {
		serverlessMetrics.SendASMInvocationEnhancedMetric(nil, lp.demux)
	}

	if err := json.Unmarshal(payloadBytes, event); err != nil {
		log.Errorf("appsec: proxy-lifecycle: unexpected lambda event parsing error: %v", err)
		return
	}

	// In monitoring-only mode - without blocking - we can wait until the request's end to monitor it
	lp.invocationEvent = event
}

// OnInvokeEnd is the hook triggered when an invocation has ended
func (lp *ProxyLifecycleProcessor) OnInvokeEnd(_ *invocationlifecycle.InvocationEndDetails) {
	// noop: not suitable for both nodejs and python because the python tracer is sending the span before this gets
	// called, so we miss our chance to run the appsec monitoring and attach our tags.
	// So the final appsec monitoring logic moved to the SpanModifier instead and we use it as "invocation end" event.
}

// multiOrSingle picks the first non-nil map, and returns the content formatted
// as the multi-map.
func multiOrSingle(multi map[string][]string, single map[string]string) map[string][]string {
	if multi == nil && single != nil {
		// There is no multi-map, but there is a single-map, so we'll make a multi-map out of that.
		multi = make(map[string][]string, len(single))
		for key, value := range single {
			multi[key] = []string{value}
		}
	}
	return multi
}

//nolint:revive // TODO(ASM) Fix revive linter
type ExecutionContext interface {
	LastRequestID() string
}

type bytesStringer []byte

func (b bytesStringer) String() string {
	return string(b)
}
