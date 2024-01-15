// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package config

import (
	"bytes"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DataDog/datadog-agent/pkg/config/model"
	"github.com/DataDog/datadog-agent/pkg/config/setup"
)

/*

The config file is a list of config sections.

Each section contains:
- a title, surrounded by # characters
- at least one config entry
- an empty line

Each entry contains:
- a description of the value itself, where each line starts with ##
- an example of setting the value, all lines start with a single # or no # at all
- an empty line

The entry description optionally starts with a description of the parameter with the following format:
@param <name> - <type> - <required/optional> - default: <default value>
The part with the default value is optional.

It then contains any number (possibly 0) of description of env variables:
@env <name> - <type> - <required/optional> - default: <default value>
The part with the default value is optional.

If provided, the entry setting example starts with an empty line, followed by the example itself.
The only case where it can be omitted is when the entry is actually an object (ie. it has sub-entries).

Entries can be nested, in that case they have an increased indentation of 2 spaces per level.
Each entry is at most one more level deep than the previous one in the file.

*/

type configSection struct {
	title   []string
	entries []configEntry

	shortTitle string
	startLine  int
}

type entryParam struct {
	name         string
	ty           string
	required     bool
	defaultValue string
}

type entryEnv struct {
	name         string
	ty           string
	required     bool
	defaultValue string
}

type configEntry struct {
	params      entryParam
	envVars     []entryEnv
	description []string
	example     []string

	// basePath is the path to the entry in the config file, excluding t
	basePath  string
	startLine int
}

// parseConfigFile parses the config file and returns a list of config sections.
//
// It asserts that the config file is correctly formatted.
// Formatting errors do not stop the test immediately, so that we can report all errors at once.
func parseConfigFile(t *testing.T, configFile string) []configSection {
	lines := strings.Split(configFile, "\n")
	require.NotEmpty(t, lines)

	// ease detecting empty lines, trailing spaces are not relevant
	for idx, line := range lines {
		lines[idx] = strings.TrimRightFunc(line, unicode.IsSpace)
	}

	var startLine int
	if lines[0] == "" {
		// skip the first empty line
		startLine++
	}

	var sections []configSection
	var section configSection
	ok := true
	for ok && startLine < len(lines) {
		section, startLine, ok = parseSection(t, lines, len(sections))
		if ok {
			sections = append(sections, section)
		}
	}

	assert.Equal(t, startLine, len(lines), "unexpected lines at the end of the file")

	return sections
}

func parseSectionTitle(t *testing.T, lines []string, startLine int) ([]string, int, bool) {
	for startLine < len(lines) && assert.Empty(t, lines[startLine], "superfluous empty line at index", startLine) {
		startLine++
	}

	if startLine >= len(lines) {
		return nil, startLine, false
	}

	if !assert.Truef(t, strings.HasPrefix(lines[startLine], "###"), "was expecting a section title at index %d but got '%s'", startLine, lines[startLine]) {
		return nil, startLine, false
	}

	for idx := startLine; idx < len(lines); idx++ {

	}

	title := lines[startLine:endLine]

	return title, endLine + 1, true
}

func parseSection(t *testing.T, lines []string, startLine int) (configSection, int, bool) {
	section := configSection{
		startLine: startLine,
	}

	title, startLine, ok := parseSectionTitle(t, lines, startLine)
	if !ok {
		return configSection{}, startLine, false
	}

	section.title = title

	//TODO parse entries

	return section, startLine, true
}

// func testConfigSections(t *testing.T, sections []configSection, config model.Config) {
// }

func TestConfigFiles(t *testing.T) {
	datadogConfig := model.NewConfig("datadog", "DD", strings.NewReplacer(".", "_"))
	setup.InitConfig(datadogConfig)

	testCases := []struct {
		buildType string
		config    model.Config
	}{
		{"agent-py3", datadogConfig},
	}

	for _, testCase := range testCases {
		t.Run(testCase.buildType, func(t *testing.T) {
			buff := new(bytes.Buffer)
			err := WriteConfigFromTemplate(buff, testCase.buildType)
			require.NoError(t, err)

			configFile := buff.String()
			_ = parseConfigFile(t, configFile)
			// testConfigSections(t, sections, testCase.config)
		})
	}
}
