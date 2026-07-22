// Package media resolves user-supplied media references into something a
// backend accepts, and writes finished artifacts to disk.
//
// RunPod endpoints take media as URLs. A user working locally has files, so a
// local path is inlined as a base64 data URI — no bucket, no upload step. Very
// large files make for very large request bodies, hence MaxInlineBytes.
package media

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// MaxInlineBytes caps how large a local file may be before inlining is refused.
const MaxInlineBytes = 20 << 20 // 20 MiB

// IsURL reports whether ref is already a remote reference.
func IsURL(ref string) bool {
	return strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") || strings.HasPrefix(ref, "data:")
}

// Resolve turns a reference into something a backend can fetch. URLs pass
// through untouched; local paths become data URIs.
func Resolve(ref string) (string, error) {
	if ref == "" {
		return "", nil
	}
	if IsURL(ref) {
		return ref, nil
	}
	info, err := os.Stat(ref)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%q is neither a URL nor an existing file", ref)
		}
		return "", fmt.Errorf("stat %s: %w", ref, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%q is a directory, expected a file", ref)
	}
	if info.Size() > MaxInlineBytes {
		return "", fmt.Errorf("%s is %.1f MiB, above the %d MiB inline limit — upload it and pass the URL instead",
			ref, float64(info.Size())/(1<<20), MaxInlineBytes>>20)
	}
	b, err := os.ReadFile(ref)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", ref, err)
	}
	ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(ref)))
	if ct == "" {
		ct = http.DetectContentType(b)
	}
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = ct[:i]
	}
	return "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(b), nil
}

// ResolveAll resolves each reference in a list.
func ResolveAll(refs []string) ([]string, error) {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		v, err := Resolve(r)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

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
