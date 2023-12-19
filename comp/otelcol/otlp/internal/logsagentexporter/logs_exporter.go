// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021-present Datadog, Inc.

package logsagentexporter

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/DataDog/datadog-agent/pkg/logs/message"
	"github.com/DataDog/datadog-agent/pkg/logs/sources"
	"github.com/DataDog/datadog-agent/pkg/util/scrubber"
	"go.uber.org/zap"

	logsmapping "github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/logs"
	"go.opentelemetry.io/collector/pdata/plog"
)

// otelTag specifies a tag to be added to all logs sent from the Datadog Agent
const otelTag = "otel_source:datadog_agent"

// createConsumeLogsFunc returns an implementation of consumer.ConsumeLogsFunc
func createConsumeLogsFunc(logger *zap.Logger, logSource *sources.LogSource, logsAgentChannel chan *message.Message) func(context.Context, plog.Logs) error {

	return func(_ context.Context, ld plog.Logs) (err error) {
		defer func() {
			if err != nil {
				newErr, scrubbingErr := scrubber.ScrubString(err.Error())
				if scrubbingErr != nil {
					err = scrubbingErr
				} else {
					err = errors.New(newErr)
				}
			}
		}()

		tr, _ := logsmapping.NewTranslator(...)
		ddLogs := tr.MapLogs(ld)
		for _, ddLog := range ddLogs {
		  	ddLog.Ddtags = nil
			service := ""
			if ddLog.Service != nil {
				service = *ddLog.Service
			}
			status := ddLog.AdditionalProperties["status"]
			if status == "" {
				status = message.StatusInfo
			}
			origin := message.NewOrigin(logSource)
			origin.SetTags(tags)
			origin.SetService(service)
			origin.SetSource(logSourceName)
	
			content, err := ddLog.MarshalJSON()
			if err != nil {
				logger.Error("Error parsing log: " + err.Error())
			}
	
			// ingestionTs is an internal field used for latency tracking on the status page, not the actual log timestamp.
			ingestionTs := time.Now().UnixNano()
			message := message.NewMessage(content, origin, status, ingestionTs)
	
			logsAgentChannel <- message
		}

		return nil
	}
}
