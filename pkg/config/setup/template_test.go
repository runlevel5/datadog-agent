// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build test

package setup

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type templateEntry struct {
	msg, key, val string
}

func parseTemplate(r io.Reader, file string) []templateEntry {
	re := regexp.MustCompile(`^(\s*)## @param (\w+)(.*)`)
	sc := bufio.NewScanner(r)

	ln := 0
	ks := []string{}
	ss := []int{}
	out := []templateEntry{}

	for sc.Scan() {
		ln += 1
		if k := re.FindStringSubmatch(sc.Text()); k != nil {
			shift := len(k[1])
			top := sort.SearchInts(ss, shift)
			ss = append(ss[:top], shift)
			ks = append(ks[:top], k[2])

			for _, val := range strings.Split(k[3], " - ") {
				if strings.HasPrefix(val, "default: ") {
					key := strings.Join(ks, ".")
					out = append(out, templateEntry{
						fmt.Sprintf("%s:%d: %s", file, ln, key),
						key,
						val,
					})
					break
				}
			}
		}
	}

	return out
}

// assert.EqualValues can't compare []string with []any.
func sliceOfAny(v []string) any {
	w := make([]any, len(v))
	for i := range v {
		w[i] = v[i]
	}
	return w
}

func TestTemplate(t *testing.T) {
	path := "../config_template.yaml"
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	config := Conf()

	for _, p := range parseTemplate(f, path) {
		v := config.Get(p.key)

		// Zero values are often overridden at later stages with real defaults.
		if !config.IsKnown(p.key) || v == nil || v == "" || v == 0 {
			continue
		}

		var d struct {
			V any `yaml:"default"`
		}
		if !assert.NoError(t, yaml.Unmarshal([]byte(p.val), &d), p.msg) {
			continue
		}

		switch v.(type) {
		case string:
			assert.Equal(t, v, fmt.Sprintf("%v", d.V), p.msg)
		case []string:
			assert.EqualValues(t, sliceOfAny(v.([]string)), d.V, p.msg)
		case time.Duration:
			dv, err := time.ParseDuration(fmt.Sprintf("%s", d.V))
			if assert.NoError(t, err, p.msg) {
				assert.Equal(t, v, dv, p.msg)
			}
		default:
			assert.EqualValues(t, v, d.V, p.msg)
		}
	}
}
