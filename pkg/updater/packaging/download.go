package packaging

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/mholt/archiver/v3"
)

// Downloader is the downloader used by the updater to download packages.
type Downloader struct {
	client *http.Client
}

func NewDownloader(client *http.Client) *Downloader {
	return &Downloader{
		client: client,
	}
}

func (d *Downloader) Download(ctx context.Context, url string, expectedSHA256 []byte, destinationPath string) error {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return fmt.Errorf("could not create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("could not create download request: %w", err)
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("could not download package: %w", err)
	}
	defer resp.Body.Close()
	hashWriter := sha256.New()
	reader := io.TeeReader(resp.Body, hashWriter)
	sha256 := hashWriter.Sum(nil)
	if !bytes.Equal(expectedSHA256, sha256) {
		return fmt.Errorf("invalid hash for %s: expected %x, got %x", url, expectedSHA256, sha256)
	}
	archive := archiver.NewTarGz()
	archive.Open(reader, 0)
	archive.Walk(archive string, walkFn archiver.WalkFunc)
	return "", nil
}
