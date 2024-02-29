// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build zlib

package compression

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"

	pkgconfigmodel "github.com/DataDog/datadog-agent/pkg/config/model"

	"github.com/DataDog/zstd"
)

const (
	ZlibCompressor      = "zlib"
	ZlibContentEncoding = "deflate"
	ZstdCompressor      = "zstd"
	ZstdContentEncoding = "zstd"
)

// ContentEncoding default as ZlibContentEncoding
var ContentEncoding = ZlibContentEncoding

func GetContentEncoding(cfg pkgconfigmodel.Reader) string {
	kind := cfg.GetString("serializer_compressor_kind")
	if kind == ZlibCompressor {
		return ZlibContentEncoding
	}
	return ZstdContentEncoding
}

func Compress(cfg pkgconfigmodel.Reader, payload []byte) ([]byte, error) {
	kind := cfg.GetString("serializer_compressor_kind")
	var compressedPayload []byte
	var err error
	switch kind {
	case ZlibCompressor:
		compressedPayload, err = zlibCompress(payload)
	case ZstdCompressor:
		compressedPayload, err = zstd.Compress(nil, payload)
	}
	return compressedPayload, err
}

func Decompress(cfg pkgconfigmodel.Reader, payload []byte) ([]byte, error) {
	kind := cfg.GetString("serializer_compressor_kind")
	return DecompressWithKind(payload, kind)
}

func DecompressWithKind(payload []byte, kind string) ([]byte, error) {
	switch kind {
	case ZstdCompressor:
		return zstd.Decompress(nil, payload)
	case ZlibCompressor:
		return zlibDecompress(payload)
	}
	return nil, fmt.Errorf("invalid compressor kind")
}

func NewCompressBound(sourceLen int, kind string) int {
	if kind == ZlibCompressor {
		return sourceLen + (sourceLen >> 12) + (sourceLen >> 14) + (sourceLen >> 25) + 13
	}
	return zstd.CompressBound(sourceLen)
}

func zlibCompress(src []byte) ([]byte, error) {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	_, err := w.Write(src)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	dst := b.Bytes()
	return dst, nil
}

func zlibDecompress(src []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	dst, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return dst, nil
}
