// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package defaultforwarder

import (
	"embed"
	"encoding/json"
	"expvar"
	htmlTemplate "html/template"
	"io"
	"path"
	"strconv"
	textTemplate "text/template"

	"github.com/DataDog/datadog-agent/comp/core/config"
	"github.com/DataDog/datadog-agent/comp/core/status"
)

//go:embed status_templates
var templatesFS embed.FS

type statusProvider struct {
	config config.Component
}

func (s statusProvider) getStatusInfo() map[string]interface{} {
	stats := make(map[string]interface{})

	s.populateStatus(stats)

	return stats
}

func (s statusProvider) populateStatus(stats map[string]interface{}) {
	forwarderStatsJSON := []byte(expvar.Get("forwarder").String())
	forwarderStats := make(map[string]interface{})
	json.Unmarshal(forwarderStatsJSON, &forwarderStats) //nolint:errcheck
	forwarderStorageMaxSizeInBytes := s.config.GetInt("forwarder_storage_max_size_in_bytes")
	if forwarderStorageMaxSizeInBytes > 0 {
		forwarderStats["forwarder_storage_max_size_in_bytes"] = strconv.Itoa(forwarderStorageMaxSizeInBytes)
	}
	stats["forwarderStats"] = forwarderStats
}

func (s statusProvider) Name() string {
	return "Forwarder"
}

func (s statusProvider) Section() string {
	return "forwarder"
}

func (s statusProvider) JSON(stats map[string]interface{}) error {
	s.populateStatus(stats)

	return nil
}

func (s statusProvider) Text(buffer io.Writer) error {
	return renderText(buffer, s.getStatusInfo())
}

func (s statusProvider) HTML(buffer io.Writer) error {
	return renderHTML(buffer, s.getStatusInfo())
}

func renderHTML(buffer io.Writer, data any) error {
	tmpl, tmplErr := templatesFS.ReadFile(path.Join("status_templates", "forwarderHTML.tmpl"))
	if tmplErr != nil {
		return tmplErr
	}
	t := htmlTemplate.Must(htmlTemplate.New("forwarderHTML").Funcs(status.HTMLFmap()).Parse(string(tmpl)))
	return t.Execute(buffer, data)
}

func renderText(buffer io.Writer, data any) error {
	tmpl, tmplErr := templatesFS.ReadFile(path.Join("status_templates", "forwarder.tmpl"))
	if tmplErr != nil {
		return tmplErr
	}
	t := textTemplate.Must(textTemplate.New("forwarder").Funcs(status.TextFmap()).Parse(string(tmpl)))
	return t.Execute(buffer, data)
}
