// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

//nolint:revive // TODO(SERV) Fix revive linter
package trigger

import (
	"strings"

	"github.com/DataDog/datadog-agent/pkg/util/json"
	jsonEncoder "github.com/json-iterator/go"
)

// AWSEventType corresponds to the various event triggers
// we get from AWS
type AWSEventType int

const (
	// Unknown describes an unknown event type
	Unknown AWSEventType = iota

	// APIGatewayEvent describes an event from classic AWS API Gateways
	APIGatewayEvent

	// APIGatewayV2Event describes an event from AWS API Gateways
	APIGatewayV2Event

	// APIGatewayWebsocketEvent describes an event from websocket AWS API Gateways
	APIGatewayWebsocketEvent

	// APIGatewayLambdaAuthorizerTokenEvent describes an event from a token-based API Gateway lambda authorizer
	APIGatewayLambdaAuthorizerTokenEvent

	// APIGatewayLambdaAuthorizerRequestParametersEvent describes an event from a request-parameters-based API Gateway lambda authorizer
	APIGatewayLambdaAuthorizerRequestParametersEvent

	// ALBEvent describes an event from application load balancers
	ALBEvent

	// CloudWatchEvent describes an event from Cloudwatch
	CloudWatchEvent

	// CloudWatchLogsEvent describes an event from Cloudwatch logs
	CloudWatchLogsEvent

	// CloudFrontRequestEvent describes an event from Cloudfront
	CloudFrontRequestEvent

	// DynamoDBStreamEvent describes an event from DynamoDB
	DynamoDBStreamEvent

	// KinesisStreamEvent describes an event from Kinesis
	KinesisStreamEvent

	// S3Event describes an event from S3
	S3Event

	// SNSEvent describes an event from SNS
	SNSEvent

	// SQSEvent describes an event from SQS
	SQSEvent

	// SNSSQSEvent describes an event from an SQS-wrapped SNS topic
	SNSSQSEvent

	// AppSyncResolverEvent describes an event from Appsync
	AppSyncResolverEvent

	// EventBridgeEvent describes an event from EventBridge
	EventBridgeEvent

	// LambdaFunctionURLEvent describes an event from an HTTP lambda function URL invocation
	LambdaFunctionURLEvent
)

// eventParseFunc defines the signature of AWS event parsing functions
type eventParseFunc func([]byte) bool

type checker struct {
	check eventParseFunc
	typ   AWSEventType
}

var (
	unknownChecker = new(checker)
	// TODO: refactor to store the last event type on the execution context
	// instead of as a global
	lastEventChecker = unknownChecker
	eventCheckers    = []*checker{
		{isAPIGatewayEvent, APIGatewayEvent},
		{isAPIGatewayV2Event, APIGatewayV2Event},
		{isAPIGatewayWebsocketEvent, APIGatewayWebsocketEvent},
		{isAPIGatewayLambdaAuthorizerTokenEvent, APIGatewayLambdaAuthorizerTokenEvent},
		{isAPIGatewayLambdaAuthorizerRequestParametersEvent, APIGatewayLambdaAuthorizerRequestParametersEvent},
		{isALBEvent, ALBEvent},
		{isCloudFrontRequestEvent, CloudFrontRequestEvent},
		{isCloudwatchEvent, CloudWatchEvent},
		{isCloudwatchLogsEvent, CloudWatchLogsEvent},
		{isDynamoDBStreamEvent, DynamoDBStreamEvent},
		{isKinesisStreamEvent, KinesisStreamEvent},
		{isS3Event, S3Event},
		{isSNSEvent, SNSEvent},
		{isSNSSQSEvent, SNSSQSEvent},
		{isSQSEvent, SQSEvent},
		{isAppSyncResolverEvent, AppSyncResolverEvent},
		{isEventBridgeEvent, EventBridgeEvent},
		{isLambdaFunctionURLEvent, LambdaFunctionURLEvent},
		// Ultimately check this is a Kong API Gateway event as a last resort.
		// This is because Kong API Gateway events are a subset of API Gateway events
		// as of https://github.com/Kong/kong/blob/348c980/kong/plugins/aws-lambda/request-util.lua#L248-L260
		{isKongAPIGatewayEvent, APIGatewayEvent},
	}
)

// GetEventType takes in a payload string and returns an AWSEventType
// that matches the input payload. Returns `Unknown` if a payload could not be
// matched to an event.
func GetEventType(payload []byte) AWSEventType {
	// allow last event type to jump ahead in line since it is likely the event
	// type is the same between invocations
	if lastEventChecker != unknownChecker {
		if lastEventChecker.check(payload) {
			return lastEventChecker.typ
		}
	}
	for _, checker := range eventCheckers {
		if checker == lastEventChecker {
			continue // typ already checked
		}
		if checker.check(payload) {
			lastEventChecker = checker
			return checker.typ
		}
	}
	lastEventChecker = unknownChecker
	return Unknown
}

// Unmarshal unmarshals a payload string into a generic interface
func Unmarshal(payload []byte) (map[string]any, error) {
	jsonPayload := make(map[string]any)
	if err := jsonEncoder.Unmarshal(payload, &jsonPayload); err != nil {
		return nil, err
	}
	return jsonPayload, nil
}

func isAPIGatewayEvent(event []byte) bool {
	return json.GetNestedValue(event, "requestContext", "stage") != nil &&
		json.GetNestedValue(event, "httpMethod") != nil &&
		json.GetNestedValue(event, "resource") != nil &&
		!isAPIGatewayLambdaAuthorizerRequestParametersEvent(event)
}

func isAPIGatewayV2Event(event []byte) bool {
	if version, ok := json.GetNestedValue(event, "version").(string); !ok || version != "2.0" {
		return false
	}
	if domainName, ok := json.GetNestedValue(event, "requestContext", "domainName").(string); !ok || strings.Contains(domainName, "lambda-url") {
		return false
	}
	return json.GetNestedValue(event, "rawQueryString") != nil
}

// Kong API Gateway events are regular API Gateway events with a few missing
// fields (cf. https://github.com/Kong/kong/blob/348c980/kong/plugins/aws-lambda/request-util.lua#L248-L260)
// As a result, this function must be called after isAPIGatewayEvent() and
// related API Gateway event payload checks. It returns true when httpmethod and
// resource are present.
func isKongAPIGatewayEvent(event []byte) bool {
	return json.GetNestedValue(event, "httpMethod") != nil &&
		json.GetNestedValue(event, "resource") != nil
}

func isAPIGatewayWebsocketEvent(event []byte) bool {
	// TODO: this isn't tested along with the others
	return json.GetNestedValue(event, "requestContext") != nil &&
		json.GetNestedValue(event, "requestContext", "messageDirection") != nil
}

func isAPIGatewayLambdaAuthorizerTokenEvent(event []byte) bool {
	return json.GetNestedValue(event, "type") == "TOKEN" &&
		json.GetNestedValue(event, "authorizationToken") != nil &&
		json.GetNestedValue(event, "methodArn") != nil
}

func isAPIGatewayLambdaAuthorizerRequestParametersEvent(event []byte) bool {
	return json.GetNestedValue(event, "type") == "REQUEST" &&
		json.GetNestedValue(event, "methodArn") != nil &&
		json.GetNestedValue(event, "headers") != nil &&
		json.GetNestedValue(event, "queryStringParameters") != nil &&
		json.GetNestedValue(event, "requestContext", "apiId") != nil
}

func isALBEvent(event []byte) bool {
	return json.GetNestedValue(event, "requestContext", "elb") != nil
}

func isCloudwatchEvent(event []byte) bool {
	source, ok := json.GetNestedValue(event, "source").(string)
	return ok && source == "aws.events"
}

func isCloudwatchLogsEvent(event []byte) bool {
	return json.GetNestedValue(event, "awslogs") != nil
}

func isCloudFrontRequestEvent(event []byte) bool {
	return eventRecordsKeyExists(event, "cf")
}

func isDynamoDBStreamEvent(event []byte) bool {
	return eventRecordsKeyExists(event, "dynamodb")
}

func isKinesisStreamEvent(event []byte) bool {
	return eventRecordsKeyExists(event, "kinesis")
}

func isS3Event(event []byte) bool {
	return eventRecordsKeyExists(event, "s3")
}

func isSNSEvent(event []byte) bool {
	return eventRecordsKeyExists(event, "Sns")
}

func isSQSEvent(event []byte) bool {
	return eventRecordsKeyEquals(event, "eventSource", "aws:sqs")
}

func isSNSSQSEvent(event []byte) bool {
	if !eventRecordsKeyEquals(event, "eventSource", "aws:sqs") {
		return false
	}
	messageType, ok := json.GetNestedValue(event, "body", "type").(string)
	if !ok {
		return false
	}

	topicArn, ok := json.GetNestedValue(event, "body", "topicarn").(string)
	if !ok {
		return false
	}

	return messageType == "notification" && topicArn != ""
}

func isAppSyncResolverEvent(event []byte) bool {
	return json.GetNestedValue(event, "info", "selectionsetgraphql") != nil
}

func isEventBridgeEvent(event []byte) bool {
	return json.GetNestedValue(event, "detail-type") != nil && json.GetNestedValue(event, "source") != "aws.events"
}

func isLambdaFunctionURLEvent(event []byte) bool {
	lambdaURL, ok := json.GetNestedValue(event, "requestContext", "domainName").(string)
	if !ok {
		return false
	}
	return strings.Contains(lambdaURL, "lambda-url")
}

func eventRecordsKeyExists(event []byte, key string) bool {
	records, ok := json.GetNestedValue(event, "Records").([]interface{})
	if !ok {
		return false
	}
	if len(records) > 0 {
		if records[0].(map[string]any)[key] != nil {
			return true
		}
	}
	return false
}

func eventRecordsKeyEquals(event []byte, key string, val string) bool {
	records, ok := json.GetNestedValue(event, "Records").([]interface{})
	if !ok {
		return false
	}
	if len(records) > 0 {
		if mapVal := records[0].(map[string]any)[key]; mapVal != nil {
			return mapVal == val
		}
	}
	return false
}
