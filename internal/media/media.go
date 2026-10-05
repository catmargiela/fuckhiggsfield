// Package media writes finished artifacts to disk.
package media

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// SaveBase64 decodes base64 data into dir/name.ext and returns the path.
func SaveBase64(dir, name, ext, b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("decode base64 output: %w", err)
	}
	return writeFile(dir, name, ext, raw)
}

// Download fetches a URL into dir/name.ext and returns the path.
func Download(ctx context.Context, hc *http.Client, dir, name, ext, url string) (string, error) {
	if hc == nil {
		hc = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 512<<20))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", url, err)
	}
	return writeFile(dir, name, ext, raw)
}

func writeFile(dir, name, ext string, raw []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}
	if ext == "" {
		ext = "bin"
	}
	path := filepath.Join(dir, name+"."+strings.TrimPrefix(ext, "."))
	// Never clobber an existing artifact — suffix until the name is free.
	for i := 2; ; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			break
		}
		path = filepath.Join(dir, fmt.Sprintf("%s-%d.%s", name, i, strings.TrimPrefix(ext, ".")))
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}
