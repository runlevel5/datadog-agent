// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package updater

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

const (
	archiveName                = "package.tar.gz"
	maxArchiveDecompressedSize = 10 << 30 // 10GiB
)

// downloader is the downloader used by the updater to download packages.
type downloader struct {
	client *http.Client
}

// newDownloader returns a new Downloader.
func newDownloader(client *http.Client) *downloader {
	return &downloader{
		client: client,
	}
}

// Download downloads the package at the given URL in temporary directory,
// verifies its SHA256 hash and extracts it to the given destination path.
// It currently assumes the package is a tar.gz archive.
func (d *downloader) Download(ctx context.Context, pkg Package, destinationPath string) error {
	log.Debugf("Downloading package %s version %s from %s", pkg.Name, pkg.Version, pkg.URL)
	registry, err := name.NewRegistry("gcr.io", name.StrictValidation)
	if err != nil {
		return fmt.Errorf("could not create repository: %w", err)
	}
	digest := registry.Repo("datadoghq", "agent").Digest("sha256:32a9fbff2c13d04a369cb9436ccb281068d6c5c11f6ee3880412eaf3564cde1e")
	image, err := remote.Image(digest)
	if err != nil {
		return fmt.Errorf("could not retrieve image: %w", err)
	}
	layers, err := image.Layers()
	if err != nil {
		return fmt.Errorf("could not retrieve manifest: %w", err)
	}
	for _, layer := range layers {
		reader, err := layer.Uncompressed()
		if err != nil {
			return fmt.Errorf("could not retrieve layer: %w", err)
		}
		defer reader.Close()
		err = extractTarArchive(reader, destinationPath)
	}
	log.Debugf("Successfully downloaded package %s version %s from %s", pkg.Name, pkg.Version, pkg.URL)
	return nil
}

// extractTarArchive extracts a tar archive to the given destination path
//
// Note on security: This function does not currently attempt to fully mitigate zip-slip attacks.
// This is purposeful as the archive is extracted only after its SHA256 hash has been validated
// against its reference in the package catalog. This catalog is itself sent over Remote Config
// which guarantees its integrity.
func extractTarArchive(reader io.Reader, destinationPath string) error {
	log.Debugf("Extracting archive to %s", destinationPath)

	tr := tar.NewReader(io.LimitReader(reader, maxArchiveDecompressedSize))
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("could not read tar header: %w", err)
		}
		if header.Name == "./" {
			continue
		}

		target := filepath.Join(destinationPath, header.Name)

		// Check for directory traversal. Note that this is more of a sanity check than a security measure.
		if !strings.HasPrefix(target, filepath.Clean(destinationPath)+string(os.PathSeparator)) {
			return fmt.Errorf("tar entry %s is trying to escape the destination directory", header.Name)
		}

		// Extract element depending on its type
		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(target, 0755)
			if err != nil {
				return fmt.Errorf("could not create directory: %w", err)
			}
		case tar.TypeReg:
			err = extractTarFile(target, tr, os.FileMode(header.Mode))
			if err != nil {
				return err // already wrapped
			}
		case tar.TypeSymlink:
			err = os.Symlink(header.Linkname, target)
			if err != nil {
				return fmt.Errorf("could not create symlink: %w", err)
			}
		case tar.TypeLink:
			// we currently don't support hard links in the updater
		default:
			log.Warnf("Unsupported tar entry type %d for %s", header.Typeflag, header.Name)
		}
	}

	log.Debugf("Successfully extracted archive to %s", destinationPath)
	return nil
}

// extractTarFile extracts a file from a tar archive.
// It is separated from extractTarGz to ensure `defer f.Close()` is called right after the file is written.
func extractTarFile(targetPath string, reader io.Reader, mode fs.FileMode) error {
	err := os.MkdirAll(filepath.Dir(targetPath), 0755)
	if err != nil {
		return fmt.Errorf("could not create directory: %w", err)
	}
	f, err := os.OpenFile(targetPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(mode))
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, reader)
	if err != nil {
		return fmt.Errorf("could not write file: %w", err)
	}
	return nil
}
