// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf

package http2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/http2/hpack"
)

func TestHTTP2Path(t *testing.T) {
	tests := []struct {
		name        string
		rawPath     string
		expectedErr bool
	}{
		{
			name:    "Short path",
			rawPath: "/hello.HelloService/SayHello",
		},
		{
			name:    "Long path",
			rawPath: "/resourcespb.ResourceTagging/GetResourceTags",
		},
		{
			name:        "Path does not start with /",
			rawPath:     "hello.HelloService/SayHello",
			expectedErr: true,
		},
		{
			name:        "Empty path",
			rawPath:     "",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf []byte
			var arr [maxHTTP2Path]uint8
			buf = hpack.AppendHuffmanString(buf, tt.rawPath)
			copy(arr[:], buf)

			path, err := decodeHTTP2Path(arr, uint8(len(buf)))
			if tt.expectedErr {
				assert.Error(t, err)
				return
			}
			assert.Equal(t, tt.rawPath, string(path))
		})
	}
}
