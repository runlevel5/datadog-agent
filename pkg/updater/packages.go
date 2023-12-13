package updater

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	stableVersionLink     = "stable"
	experimentVersionLink = "experiment"
)

type Packages struct {
	packages []Package

	stableVersion     string
	experimentVersion string
}

func NewPackageManager(rootPath string) (*Packages, error) {
	err := os.Mkdir(rootPath, 0755)
	if err != nil {
		return nil, fmt.Errorf("could not create packages root directory: %w", err)
	}

	packages := Packages{}

	stableLinkPath = filepath.Join(rootPath, stableVersionLink)
	stableLinkExists, err := linkExists(rootPath, stableVersionLink)
	if err != nil {
		return nil, fmt.Errorf("could not check if stable link exists: %w", err)
	}
	if !stableLinkExists {
		return &packages, nil
	}

}

func linkExists(rootPath, linkName string) (bool, error) {
	linkPath := filepath.Join(rootPath, linkName)
	_, err := os.Stat(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("could not stat link: %w", err)
	}
	return true, nil
}

func (p *Packages) Download(version string, sha256hash [32]byte, source io.ReadCloser) {

}

type Package struct {
	Version string
	Path    string
}
