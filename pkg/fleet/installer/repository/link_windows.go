// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build windows

package repository

import (
	"errors"
)

func linkRead2(_ string) (string, error) {
	return "", errors.New("not supported on windows")
}

func linkExists2(_ string) (bool, error) {
	return false, errors.New("not supported on windows")
}

func linkSet2(_ string, _ string) error {
	return errors.New("not supported on windows")
}

func linkDelete2(_ string) error {
	return errors.New("not supported on windows")
}
