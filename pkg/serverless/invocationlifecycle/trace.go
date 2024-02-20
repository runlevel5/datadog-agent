// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package invocationlifecycle

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	json "github.com/json-iterator/go"

	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	functionNameEnvVar = "AWS_LAMBDA_FUNCTION_NAME"
)

var /* const */ runtimeRegex = regexp.MustCompile(`^(dotnet|go|java|ruby)(\d+(\.\d+)*|\d+(\.x))$`)

// ExecutionStartInfo is saved information from when an execution span was started
type ExecutionStartInfo struct {
	startTime      time.Time
	TraceID        uint64
	SpanID         uint64
	parentID       uint64
	requestPayload []byte
	//SamplingPriority sampler.SamplingPriority
}

// startExecutionSpan records information from the start of the invocation.
// It should be called at the start of the invocation.
func (lp *LifecycleProcessor) startExecutionSpan(event interface{}, rawPayload []byte, startDetails *InvocationStartDetails) {

	executionContext := lp.GetExecutionInfo()
	executionContext.requestPayload = rawPayload
	executionContext.startTime = startDetails.StartTime

}

// ParseLambdaPayload removes extra data sent by the proxy that surrounds
// a JSON payload. For example, for `a5a{"event":"aws_lambda"...}0` it would remove
// a5a at the front and 0 at the end, and just leave a correct JSON payload.
func ParseLambdaPayload(rawPayload []byte) []byte {
	leftIndex := bytes.Index(rawPayload, []byte("{"))
	rightIndex := bytes.LastIndex(rawPayload, []byte("}"))
	if leftIndex == -1 || rightIndex == -1 {
		return rawPayload
	}
	return rawPayload[leftIndex : rightIndex+1]
}

func convertStrToUnit64(s string) (uint64, error) {
	num, err := strconv.ParseUint(s, 0, 64)
	if err != nil {
		log.Debugf("Error while converting %s, failing with : %s", s, err)
	}
	return num, err
}

// InjectContext injects the context
func InjectContext(executionContext *ExecutionStartInfo, headers http.Header) {
	if value, err := convertStrToUnit64(headers.Get(TraceIDHeader)); err == nil {
		log.Debugf("injecting traceID = %v", value)
		executionContext.TraceID = value
	}
	if value, err := convertStrToUnit64(headers.Get(ParentIDHeader)); err == nil {
		log.Debugf("injecting parentId = %v", value)
		executionContext.parentID = value
	}
	if value, err := strconv.ParseInt(headers.Get(SamplingPriorityHeader), 10, 8); err == nil {
		log.Debugf("injecting samplingPriority = %v", value)
		//executionContext.SamplingPriority = sampler.SamplingPriority(value)
	}
}

// InjectSpanID injects the spanId
func InjectSpanID(executionContext *ExecutionStartInfo, headers http.Header) {
	if value, err := strconv.ParseUint(headers.Get(SpanIDHeader), 10, 64); err == nil {
		log.Debugf("injecting spanID = %v", value)
		executionContext.SpanID = value
	}
}

func convertJSONToString(payloadJSON interface{}) string {
	jsonData, err := json.Marshal(payloadJSON)
	if err != nil {
		return fmt.Sprintf("%v", payloadJSON)
	}
	return string(jsonData)
}
