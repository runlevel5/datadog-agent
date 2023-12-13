// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package updater

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	previousVersionLink   = "previous"
	stableVersionLink     = "stable"
	experimentVersionLink = "experiment"
)

type Repository struct {
	rootPath string

	stable     *Package
	experiment *Package
	previous   *Package
}

func NewRepository(rootPath string) (*Repository, error) {
	err := os.MkdirAll(rootPath, 0755)
	if err != nil {
		return nil, fmt.Errorf("could not create packages root directory: %w", err)
	}

	stablePackage, err := newPackage(filepath.Join(rootPath, stableVersionLink))
	if err != nil {
		return nil, fmt.Errorf("could not load stable package: %w", err)
	}
	experimentPackage, err := newPackage(filepath.Join(rootPath, experimentVersionLink))
	if err != nil {
		return nil, fmt.Errorf("could not load experiment package: %w", err)
	}
	previousPackage, err := newPackage(filepath.Join(rootPath, previousVersionLink))
	if err != nil {
		return nil, fmt.Errorf("could not load previous package: %w", err)
	}

	return &Repository{
		rootPath:   rootPath,
		stable:     stablePackage,
		experiment: experimentPackage,
		previous:   previousPackage,
	}, nil
}

func (r *Repository) SetStable() *Package {
	return r.stable
}

func (r *Repository) SetExperiment() *Package {
	return r.stable
}

type Package struct {
	linkPath string
	path     *string
}

func newPackage(linkPath string) (*Package, error) {
	linkExists, err := linkExists(linkPath)
	if err != nil {
		return nil, fmt.Errorf("could check if link exists: %w", err)
	}
	if !linkExists {
		return &Package{
			linkPath: linkPath,
		}, nil
	}
	packagePath, err := linkRead(linkPath)
	if err != nil {
		return nil, fmt.Errorf("could not read link: %w", err)
	}
	_, err = os.Stat(packagePath)
	if err != nil {
		return nil, fmt.Errorf("could not read package: %w", err)
	}

	return &Package{
		linkPath: linkPath,
		path:     &packagePath,
	}, nil
}

func (p *Package) Exists() bool {
	return p.path != nil
}

func (p *Package) Set(path string) error {
	err := linkSet(p.linkPath, path)
	if err != nil {
		return fmt.Errorf("could not set link: %w", err)
	}
	p.path = &path
	return nil
}

func (p *Package) Delete() error {
	err := linkDelete(p.linkPath)
	if err != nil {
		return fmt.Errorf("could not delete link: %w", err)
	}
	p.path = nil
	return nil
}
