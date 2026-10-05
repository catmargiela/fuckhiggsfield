package openai

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The image bytes belong in Outputs; a second copy in Raw bloats the job store.
func TestCreateKeepsBase64OutOfRaw(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("PNG", 100)))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"created":1,"data":[{"b64_json":"`+payload+`","revised_prompt":"a cat"}]}`)
	}))
	defer srv.Close()

	job, err := New(srv.URL, "sk-test", srv.Client()).Create(context.Background(), imageModel(), map[string]any{"prompt": "a cat"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(job.Outputs) != 1 || job.Outputs[0].B64 != payload {
		t.Fatal("output lost its image bytes")
	}
	if strings.Contains(string(job.Raw), payload) {
		t.Error("raw response still holds the base64 payload")
	}
	if !strings.Contains(string(job.Raw), "revised_prompt") {
		t.Errorf("raw response lost its other fields: %s", job.Raw)
	}
}
