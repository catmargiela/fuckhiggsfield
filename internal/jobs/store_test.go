package jobs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catmargiela/fuckhiggsfield/internal/provider"
)

// A store written by an older version holds whole base64 images in Raw; they
// must be shed on the next load so the file shrinks back on the next save.
func TestOpenDropsOversizedRaw(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.json")
	t.Setenv("BARA_JOBS_PATH", path)
	big := `{"b64_json":"` + strings.Repeat("A", MaxRawBytes) + `"}`
	small := `{"created":1}`
	old := Store{Jobs: []*provider.Job{
		{ID: "big", Created: time.Now(), Raw: json.RawMessage(big)},
		{ID: "small", Created: time.Now(), Raw: json.RawMessage(small)},
	}}
	b, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}

	s, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	for _, id := range []string{"big", "small"} {
		j, err := s.Get(id)
		if err != nil {
			t.Fatalf("Get %s: %v", id, err)
		}
		if id == "big" && j.Raw != nil {
			t.Error("oversized raw was kept")
		}
		if id == "small" && string(j.Raw) != small {
			t.Errorf("small raw = %s, want it untouched", j.Raw)
		}
	}
}
