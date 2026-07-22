// Package provider defines the backend abstraction: submit a generation,
// poll it, and collect its outputs.
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/catmargiela/fuckhiggsfield/internal/catalog"
)

// Status is a normalized job state shared by every backend.
type Status string

const (
	StatusQueued    Status = "QUEUED"
	StatusRunning   Status = "IN_PROGRESS"
	StatusCompleted Status = "COMPLETED"
	StatusFailed    Status = "FAILED"
	StatusCancelled Status = "CANCELLED"
	StatusTimedOut  Status = "TIMED_OUT"
)

// Terminal reports whether a status will never change again.
func (s Status) Terminal() bool {
	switch s {
	case StatusCompleted, StatusFailed, StatusCancelled, StatusTimedOut:
		return true
	}
	return false
}

// OK reports whether the job finished successfully.
func (s Status) OK() bool { return s == StatusCompleted }

// Output is one artifact produced by a job.
type Output struct {
	// URL is a remote link, when the backend returns one.
	URL string `json:"url,omitempty"`
	// B64 is inline base64 data, when the backend returns bytes.
	B64 string `json:"-"`
	// Text is the payload for text models.
	Text string `json:"text,omitempty"`
	// Ext is the file extension to use when saving, without the dot.
	Ext string `json:"ext,omitempty"`
	// LocalPath is filled in once the CLI writes the artifact to disk.
	LocalPath string `json:"local_path,omitempty"`
	// PublicURL is set once the artifact has been republished to a durable
	// URL. Unlike URL, it does not expire out from under you.
	PublicURL string `json:"public_url,omitempty"`
}

// Job is a normalized generation job.
type Job struct {
	ID       string    `json:"id"`
	Model    string    `json:"model"`
	Provider string    `json:"provider"`
	Endpoint string    `json:"endpoint"`
	Status   Status    `json:"status"`
	Outputs  []Output  `json:"outputs,omitempty"`
	Error    string    `json:"error,omitempty"`
	Cost     float64   `json:"cost,omitempty"`
	Created  time.Time `json:"created_at"`
	// Raw is the untouched provider response, surfaced by --json.
	Raw json.RawMessage `json:"raw,omitempty"`
}

// Provider submits and polls generations for one backend.
type Provider interface {
	// Name is the registry provider key, e.g. "runpod".
	Name() string
	// Create submits a generation. Implementations that are synchronous
	// return an already-terminal job.
	Create(ctx context.Context, m catalog.Model, params map[string]any) (*Job, error)
	// Get fetches current job state. endpoint is the provider-side model
	// identifier the job was created against.
	Get(ctx context.Context, endpoint, id string) (*Job, error)
}

// Canceler is implemented by providers that can abort a running job.
type Canceler interface {
	Cancel(ctx context.Context, endpoint, id string) error
}

// APIError is a structured backend failure.
type APIError struct {
	Provider   string
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = strings.TrimSpace(e.Body)
	}
	if len(msg) > 600 {
		msg = msg[:600] + "…"
	}
	switch e.StatusCode {
	case 401, 403:
		return fmt.Sprintf("%s rejected the API key (HTTP %d): %s", e.Provider, e.StatusCode, msg)
	case 404:
		return fmt.Sprintf("%s endpoint not found (HTTP 404): %s", e.Provider, msg)
	case 429:
		return fmt.Sprintf("%s rate limit reached (HTTP 429): %s", e.Provider, msg)
	}
	return fmt.Sprintf("%s request failed (HTTP %d): %s", e.Provider, e.StatusCode, msg)
}

// ExtFor guesses a file extension from a URL or explicit format hint.
func ExtFor(kind catalog.Kind, urlOrFormat string) string {
	s := strings.ToLower(urlOrFormat)
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	for _, ext := range []string{"png", "jpeg", "jpg", "webp", "mp4", "webm", "mov", "wav", "mp3", "flac", "ogg", "m4a", "glb"} {
		if strings.HasSuffix(s, "."+ext) || s == ext {
			return ext
		}
	}
	switch kind {
	case catalog.KindImage:
		return "png"
	case catalog.KindVideo:
		return "mp4"
	case catalog.KindAudio:
		return "wav"
	}
	return "bin"
}
