// Package jobs keeps a local record of submitted generations.
//
// OpenAI image jobs are synchronous and cannot be fetched again, so they need
// somewhere local to live for `generate list` and `generate get` to work
// against a plain job id.
package jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/catmargiela/fuckhiggsfield/internal/config"
	"github.com/catmargiela/fuckhiggsfield/internal/provider"
)

// MaxEntries bounds the store so it cannot grow without limit.
const MaxEntries = 500

// MaxRawBytes caps the provider response kept per job. Older versions stored
// whole OpenAI responses, base64 image included, which grew the store to tens
// of megabytes; anything above the cap is dropped on load and on insert.
const MaxRawBytes = 64 << 10

// Store is a append-mostly list of jobs on disk.
type Store struct {
	path string
	Jobs []*provider.Job `json:"jobs"`
}

func storePath() (string, error) {
	if p := os.Getenv("BARA_JOBS_PATH"); p != "" {
		return p, nil
	}
	d, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "jobs.json"), nil
}

// Open loads the store, creating an empty one if absent.
func Open() (*Store, error) {
	p, err := storePath()
	if err != nil {
		return nil, err
	}
	s := &Store{path: p}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read job store: %w", err)
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(b, s); err != nil {
		return nil, fmt.Errorf("parse job store %s: %w", p, err)
	}
	for _, j := range s.Jobs {
		trimRaw(j)
	}
	return s, nil
}

// Save persists the store, trimming the oldest entries past MaxEntries.
func (s *Store) Save() error {
	sort.SliceStable(s.Jobs, func(i, j int) bool { return s.Jobs[i].Created.After(s.Jobs[j].Created) })
	if len(s.Jobs) > MaxEntries {
		s.Jobs = s.Jobs[:MaxEntries]
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return fmt.Errorf("write job store: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("install job store: %w", err)
	}
	return nil
}

// Put inserts or replaces a job by id.
func (s *Store) Put(j *provider.Job) {
	trimRaw(j)
	for i, existing := range s.Jobs {
		if existing.ID == j.ID {
			// Keep the original submit time; it is the useful ordering key.
			if !existing.Created.IsZero() && j.Created.IsZero() {
				j.Created = existing.Created
			}
			if j.Model == "" {
				j.Model = existing.Model
			}
			if j.Endpoint == "" {
				j.Endpoint = existing.Endpoint
			}
			if j.Provider == "" {
				j.Provider = existing.Provider
			}
			s.Jobs[i] = j
			return
		}
	}
	s.Jobs = append(s.Jobs, j)
}

func trimRaw(j *provider.Job) {
	if len(j.Raw) > MaxRawBytes {
		j.Raw = nil
	}
}

// Get finds a job by exact id, or by unique id prefix.
func (s *Store) Get(id string) (*provider.Job, error) {
	for _, j := range s.Jobs {
		if j.ID == id {
			return j, nil
		}
	}
	var hits []*provider.Job
	for _, j := range s.Jobs {
		if strings.HasPrefix(j.ID, id) {
			hits = append(hits, j)
		}
	}
	switch len(hits) {
	case 0:
		return nil, fmt.Errorf("no local record of job %q — run `bara generate list`", id)
	case 1:
		return hits[0], nil
	}
	return nil, fmt.Errorf("job id %q is ambiguous (%d matches)", id, len(hits))
}

// List returns jobs newest-first, optionally filtered and capped.
func (s *Store) List(status string, limit int) []*provider.Job {
	out := make([]*provider.Job, 0, len(s.Jobs))
	for _, j := range s.Jobs {
		if status != "" && !strings.EqualFold(string(j.Status), status) {
			continue
		}
		out = append(out, j)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Created.After(out[j].Created) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
