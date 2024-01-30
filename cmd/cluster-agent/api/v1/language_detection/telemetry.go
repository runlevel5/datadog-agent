package language_detection

import "github.com/DataDog/datadog-agent/pkg/telemetry"

const subsystem = "language_detection_dca_handler"

var (
	commonOpts = telemetry.Options{NoDoubleUnderscoreSep: true}
)

var (
	// OkResponses tracks the number the request was processed successfully
	OkResponses = telemetry.NewCounterWithOpts(
		subsystem,
		"ok_response",
		[]string{},
		"Tracks the number the request was processed successfully",
		commonOpts,
	)

	// ErrorResponses tracks the number of times request processsing fails
	ErrorResponses = telemetry.NewCounterWithOpts(
		subsystem,
		"fail_response",
		[]string{},
		"Tracks the number of times request processing fails",
		commonOpts,
	)
)
