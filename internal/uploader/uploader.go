// Package uploader publishes generated artifacts to a public URL.
//
// Two problems need this. OpenAI returns images as base64 with no URL at all,
// and RunPod's result URLs expire after 7 days — so neither gives you a link
// you can keep. It also fixes the input side: RunPod endpoints fetch media by
// URL, and inlining a local file as a data URI stops working on large files.
//
// Uploading is opt-in. Nothing leaves the machine unless a backend is
// configured, because the zero-config hosts are public and permanent.
package uploader

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Backend names accepted by the `upload_backend` setting.
const (
	BackendNone      = "none"
	BackendCatbox    = "catbox"
	BackendLitterbox = "litterbox"
	BackendS3        = "s3"
)

// Uploader publishes bytes and returns a public URL.
type Uploader interface {
	// Name is the backend key.
	Name() string
	// Upload publishes data and returns the URL it can be fetched from.
	Upload(ctx context.Context, filename string, data []byte, contentType string) (string, error)
	// Describe is a one-line summary shown to the user, including any
	// retention caveat.
	Describe() string
}

// Config selects and configures a backend.
type Config struct {
	Backend string

	// Catbox: optional account hash; uploads are anonymous without it.
	CatboxUserHash string
	// Litterbox retention: 1h, 12h, 24h, or 72h.
	LitterboxTime string

	// S3-compatible settings (Cloudflare R2, AWS S3, Backblaze B2, MinIO).
	S3Endpoint  string
	S3Bucket    string
	S3Region    string
	S3AccessKey string
	S3SecretKey string
	S3Prefix    string
	S3PathStyle bool
	// PublicBaseURL is prepended to the object key to form the returned URL.
	// Required for S3, since the signed endpoint is usually not public.
	PublicBaseURL string

	HTTP *http.Client
}

// Backends lists every selectable backend with a description.
func Backends() [][2]string {
	return [][2]string{
		{BackendNone, "no uploading; outputs stay local (default)"},
		{BackendCatbox, "catbox.moe — no account, 200 MB, permanent, public"},
		{BackendLitterbox, "litterbox.catbox.moe — no account, 1 GB, expires (1h/12h/24h/72h)"},
		{BackendS3, "S3-compatible bucket you control (Cloudflare R2, AWS S3, B2, MinIO)"},
	}
}

// New builds an uploader. It returns (nil, nil) when uploading is disabled, so
// callers can treat "no uploader" as a normal state rather than an error.
func New(cfg Config) (Uploader, error) {
	backend := strings.TrimSpace(strings.ToLower(cfg.Backend))
	if backend == "" {
		backend = BackendNone
	}
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{Timeout: 300 * time.Second}
	}
	switch backend {
	case BackendNone:
		return nil, nil
	case BackendCatbox:
		return &catbox{userHash: cfg.CatboxUserHash, hc: cfg.HTTP}, nil
	case BackendLitterbox:
		d := cfg.LitterboxTime
		if d == "" {
			d = "72h"
		}
		if !validLitterboxTime(d) {
			return nil, fmt.Errorf("litterbox_time must be one of 1h, 12h, 24h, 72h (got %q)", d)
		}
		return &catbox{litterbox: true, retention: d, hc: cfg.HTTP}, nil
	case BackendS3:
		return newS3(cfg)
	}
	var names []string
	for _, b := range Backends() {
		names = append(names, b[0])
	}
	sort.Strings(names)
	return nil, fmt.Errorf("unknown upload_backend %q (choose one of: %s)", cfg.Backend, strings.Join(names, ", "))
}

func validLitterboxTime(d string) bool {
	switch d {
	case "1h", "12h", "24h", "72h":
		return true
	}
	return false
}
