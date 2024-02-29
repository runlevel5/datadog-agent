module github.com/DataDog/datadog-agent/pkg/util/compression

go 1.21.7

replace (
	github.com/DataDog/datadog-agent/pkg/config/model => ../../config/model
)

require (
	github.com/DataDog/datadog-agent/pkg/config/model v0.52.0-rc.3
	github.com/DataDog/zstd v1.5.5
)
