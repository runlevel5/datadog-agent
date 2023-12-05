// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

// Package main is a tool to compute the test to run for a list of modified files using code coverage
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/exp/slices"
)

func main() {
	// testCmd := exec.Command("go", "test", "-v", "-run", "TestE2ESuite", "./internal/tools/smarttest")
	// computeCoveragePerPackage()

	var mode string
	var commitSha string
	var modifiedFiles string

	flag.StringVar(&mode, "mode", "", "Chose mode to use: compute or test")
	flag.StringVar(&commitSha, "commit-sha", "", "Commit sha to use")
	flag.StringVar(&modifiedFiles, "modified-files", "", "List of modified files")
	flag.Parse()

	if slices.Contains([]string{"compute", "test"}, mode) {
		log.Fatal("Mode must be either compute or test")
	}

	if commitSha == "" {
		log.Fatal("Commit sha must be set")
	}

	if mode == "compute" {
		coveragePerPackage := computeCoveragePerPackage()
		packageToTest := computePackageToTest(coveragePerPackage)
		writeMapToFile(packageToTest, "packageToTest.json")
	} else {
		packageToTest := loadFromJSON("packageToTest.json")
		prettyPrint(packageToTest)
		getTestToRun(packageToTest, modifiedFiles)
	}

}

func computeCoveragePerPackage() map[string]map[string]bool {

	coveragePerPackage := make(map[string]map[string]bool)
	pkgCmd := exec.Command("go", "list", "./...")
	pkgOutput, err := pkgCmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	pkgList := strings.Split(string(pkgOutput), "\n")
	fmt.Println(pkgList)

	for _, pkg := range pkgList {
		if pkg == "" {
			continue
		}
		fmt.Println("Computing coverage for package: ", pkg)
		testFiles, err := filepath.Glob(pkg[33:] + "/*_test.go")
		fmt.Println("Test files: ", testFiles)
		if err != nil || testFiles == nil {
			log.Println("No test files found for package, skipping...  ", pkg[33:], err)
			continue
		}
		coverCmd := exec.Command("go", "test", "-coverpkg=./...", "-coverprofile=profile.cov", pkg, "--tags=test")
		_, err = coverCmd.Output()
		if err != nil {
			log.Println("Failed to compute coverage for package, skipping...  ", pkg)
			continue
		}
		coverOutput, err := exec.Command("cat", "profile.cov").Output()
		if err != nil {
			log.Println("Failed to read coverage file for package, skipping...  ", pkg)
		}
		coveragePerPackage[pkg] = map[string]bool{}
		for _, line := range strings.Split(string(coverOutput), "\n") {
			if line == "" {
				continue
			}
			coveredPkg := filepath.Dir(line)
			coveragePerPackage[pkg][coveredPkg] = true

		}
	}

	return coveragePerPackage

}

func computePackageToTest(coveragePerPackage map[string]map[string]bool) map[string]map[string]bool {
	packageToTest := make(map[string]map[string]bool)
	for pkg, coverage := range coveragePerPackage {
		for coveredPkg := range coverage {
			if coveredPkg == "." {
				continue
			}
			if packageToTest[coveredPkg] == nil {
				packageToTest[coveredPkg] = map[string]bool{}
			}
			packageToTest[coveredPkg][pkg] = true
		}
	}
	writeMapToFile(packageToTest, "packatetotest.json")
	return packageToTest

}

func prettyPrint(coveragePerPackage map[string]map[string]bool) {
	for pkg, coverage := range coveragePerPackage {
		fmt.Println("Package: ", pkg)

		for coveredPkg := range coverage {
			fmt.Println("  ", coveredPkg)
		}
	}
}

func loadFromJSON(filename string) map[string]map[string]bool {
	byteValue, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	var coveragePerPackage map[string]map[string]bool
	err = json.Unmarshal(byteValue, &coveragePerPackage)
	if err != nil {
		log.Fatal(err)
	}
	return coveragePerPackage
}

func writeMapToFile(coveragePerPackage map[string]map[string]bool, filename string) {
	jsonStr, err := json.Marshal(coveragePerPackage)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(filename, jsonStr, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func getTestToRun(packageToTest map[string]map[string]bool, modifiedFiles string) {
	fmt.Println("TOIMPLEMENT Modified files: ", modifiedFiles)
}
