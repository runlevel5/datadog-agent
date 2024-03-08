// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package service

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/DataDog/datadog-agent/pkg/config/setup"
)

var (
	//go:embed root_exec.sh
	rootExecScript []byte

	rootExecScriptPath = filepath.Join(setup.InstallPath, "bin", "root_exec.sh")
)

// SetupRootExec sets up the root exec script
func SetupRootExec() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("Error getting current user: %v", err)
	}
	if currentUser.Uid != "0" {
		return fmt.Errorf("Error setup is not root")
	}
	err = os.WriteFile(rootExecScriptPath, rootExecScript, 0750)
	if err != nil {
		return fmt.Errorf("Error writing root exec script: %v", err)
	}
	err = exec.Command("chown root:dd-updater %s").Run()
	if err != nil {
		return fmt.Errorf("Error changing file ownership: %v", err)
	}
	return exec.Command("chmod g+s %s", rootExecScriptPath).Run()
}

// RootExec executes a command as root
func RootExec(command string) error {
	return exec.Command(fmt.Sprintf(`script.sh "%s"`, command)).Run()
}
