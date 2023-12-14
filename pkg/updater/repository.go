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

	stable     *Link
	experiment *Link
}

func CreateRepository(rootPath string, stableSourcePath string) (*Repository, error) {
	err := os.MkdirAll(rootPath, 0755)
	if err != nil {
		return nil, fmt.Errorf("could not create packages root directory: %w", err)
	}
	repository, err := openRepository(rootPath)
	if err != nil {
		return nil, err
	}
	err = repository.setStable(stableSourcePath)
	if err != nil {
		return nil, fmt.Errorf("could not set first stable: %w", err)
	}
	return repository, nil
}

func OpenRepository(rootPath string) (*Repository, error) {
	repository, err := openRepository(rootPath)
	if err != nil {
		return nil, err
	}
	if !repository.stable.Exists() {
		return nil, fmt.Errorf("stable package does not exist, invalid state")
	}
	return repository, nil
}

func openRepository(rootPath string) (*Repository, error) {
	stableLink, err := newLink(filepath.Join(rootPath, stableVersionLink))
	if err != nil {
		return nil, fmt.Errorf("could not load stable link: %w", err)
	}
	experimentLink, err := newLink(filepath.Join(rootPath, experimentVersionLink))
	if err != nil {
		return nil, fmt.Errorf("could not load experiment link: %w", err)
	}

	return &Repository{
		rootPath:   rootPath,
		stable:     stableLink,
		experiment: experimentLink,
	}, nil
}

func (r *Repository) SetExperiment(sourcePath string) error {
	err := movePackageFromSource(r.rootPath, sourcePath)
	if err != nil {
		return fmt.Errorf("could not move source: %w", err)
	}
	return r.experiment.Set(sourcePath)
}

func (r *Repository) PromoteExperiment() error {
	if !r.experiment.Exists() {
		return fmt.Errorf("invalid state: no experiment package to promote")
	}
	err := r.stable.Set()
	if err != nil {
		return fmt.Errorf("could not set stable: %w", err)
	}
	err = r.experiment.Delete()
	if err != nil {
		return fmt.Errorf("could not delete experiment link: %w", err)
	}
	err = os.RemoveAll(path string)
	return nil
}

func (r *Repository) RemoveExperiment() error {
	if !r.experiment.Exists() {
		return fmt.Errorf("invalid state: no experiment package to remove")
	}
	err := r.experiment.Delete()
	if err != nil {
		return fmt.Errorf("could not delete experiment link: %w", err)
	}

}

func movePackageFromSource(rootPath string, sourcePath string) error {
	targetPath := filepath.Join(rootPath, filepath.Base(sourcePath))
	err := os.Rename(sourcePath, targetPath)
	if err != nil {
		return fmt.Errorf("could not move source: %w", err)
	}
	return nil
}

func (r *Repository) setStable(sourcePath string) error {
	err := movePackageFromSource(r.rootPath, sourcePath)
	if err != nil {
		return fmt.Errorf("could not move source: %w", err)
	}
	return r.stable.Set(sourcePath)
}

type Link struct {
	linkPath    string
	packagePath *string
}

func newLink(linkPath string) (*Link, error) {
	linkExists, err := linkExists(linkPath)
	if err != nil {
		return nil, fmt.Errorf("could check if link exists: %w", err)
	}
	if !linkExists {
		return &Link{
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

	return &Link{
		linkPath:    linkPath,
		packagePath: &packagePath,
	}, nil
}

func (l *Link) Exists() bool {
	return l.packagePath != nil
}

func (l *Link) Set(path string) error {
	err := linkSet(l.linkPath, path)
	if err != nil {
		return fmt.Errorf("could not set link: %w", err)
	}
	l.packagePath = &path
	return nil
}

func (l *Link) Delete() error {
	err := linkDelete(l.linkPath)
	if err != nil {
		return fmt.Errorf("could not delete link: %w", err)
	}
	l.packagePath = nil
	return nil
}
