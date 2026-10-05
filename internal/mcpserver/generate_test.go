package mcpserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catmargiela/fuckhiggsfield/internal/catalog"
	"github.com/catmargiela/fuckhiggsfield/internal/provider"
)

// fakeProvider answers synchronously with one base64 image, like OpenAI.
type fakeProvider struct{}

func (fakeProvider) Name() string { return "openai" }

func (fakeProvider) Create(_ context.Context, m catalog.Model, _ map[string]any) (*provider.Job, error) {
	return &provider.Job{
		ID: "openai-1", Model: m.ID, Provider: "openai", Endpoint: m.Endpoint,
		Status:  provider.StatusCompleted,
		Created: time.Now().UTC(),
		Outputs: []provider.Output{{B64: base64.StdEncoding.EncodeToString([]byte("PNG")), Ext: "png"}},
		Raw:     json.RawMessage(`{"data":[{"b64_json":"UE5H"}]}`),
	}, nil
}

func (fakeProvider) Get(context.Context, string, string) (*provider.Job, error) {
	return nil, errors.New("not pollable")
}

func newGenerateServer(t *testing.T, upload func(context.Context, string) (string, error)) (*Server, *bytes.Buffer) {
	t.Helper()
	t.Setenv("BARA_JOBS_PATH", filepath.Join(t.TempDir(), "jobs.json"))
	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	var out bytes.Buffer
	s := New(Deps{
		Catalog:     cat,
		ProviderFor: func(catalog.Model) (provider.Provider, error) { return fakeProvider{}, nil },
		OutputDir:   t.TempDir(),
		Download: func(context.Context, string, string, string, string) (string, error) {
			return "", errors.New("no download expected for base64 output")
		},
		Upload:        upload,
		UploadBackend: "catbox",
	}, &out)
	return s, &out
}

func callGenerate(t *testing.T, s *Server, out *bytes.Buffer, args string) map[string]any {
	t.Helper()
	got := roundTrip(t, s, out,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"generate","arguments":`+args+`}}`)
	res := got[0]["result"].(map[string]any)
	if res["isError"] == true {
		t.Fatalf("generate failed: %#v", res)
	}
	return res["structuredContent"].(map[string]any)
}

func firstOutput(t *testing.T, job map[string]any) map[string]any {
	t.Helper()
	outs, ok := job["outputs"].([]any)
	if !ok || len(outs) != 1 {
		t.Fatalf("outputs = %#v", job["outputs"])
	}
	return outs[0].(map[string]any)
}

// With a backend configured, generate publishes without being asked, and the
// result never carries the raw provider payload into the agent's context.
func TestGeneratePublishesByDefault(t *testing.T) {
	var uploaded string
	s, out := newGenerateServer(t, func(_ context.Context, path string) (string, error) {
		uploaded = path
		return "https://files.catbox.moe/abc123.png", nil
	})
	job := callGenerate(t, s, out, `{"model":"gpt_image_2","params":{"prompt":"a cat"}}`)
	o := firstOutput(t, job)
	if o["public_url"] != "https://files.catbox.moe/abc123.png" {
		t.Errorf("public_url = %v", o["public_url"])
	}
	if uploaded == "" || uploaded != o["local_path"] {
		t.Errorf("uploaded %q, want the saved file %v", uploaded, o["local_path"])
	}
	if b, err := os.ReadFile(uploaded); err != nil || string(b) != "PNG" {
		t.Errorf("saved file = %q, %v", b, err)
	}
	if _, has := job["raw"]; has {
		t.Error("generate result should not include the raw provider response")
	}
}

// A paid-for image must not be lost because the upload host is down.
func TestGenerateSurvivesUploadFailure(t *testing.T) {
	s, out := newGenerateServer(t, func(context.Context, string) (string, error) {
		return "", errors.New("catbox returned HTTP 412: uploads paused")
	})
	o := firstOutput(t, callGenerate(t, s, out, `{"model":"gpt_image_2","params":{"prompt":"a cat"}}`))
	if p, _ := o["local_path"].(string); p == "" {
		t.Error("local_path should still be returned")
	}
	if _, has := o["public_url"]; has {
		t.Errorf("public_url = %v, want none", o["public_url"])
	}
	if msg, _ := o["upload_error"].(string); !strings.Contains(msg, "412") {
		t.Errorf("upload_error = %v", o["upload_error"])
	}
}

func TestGenerateUploadFalseKeepsLocal(t *testing.T) {
	s, out := newGenerateServer(t, func(context.Context, string) (string, error) {
		t.Error("upload should not run when upload is false")
		return "", nil
	})
	o := firstOutput(t, callGenerate(t, s, out, `{"model":"gpt_image_2","params":{"prompt":"a cat"},"upload":false}`))
	if _, has := o["public_url"]; has {
		t.Errorf("public_url = %v, want none", o["public_url"])
	}
}

// Asking for a URL with uploading switched off is a mistake worth reporting.
func TestGenerateUploadTrueWithoutBackendFails(t *testing.T) {
	s, out := newGenerateServer(t, nil)
	got := roundTrip(t, s, out,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"generate","arguments":{"model":"gpt_image_2","params":{"prompt":"a cat"},"upload":true}}}`)
	res := got[0]["result"].(map[string]any)
	if res["isError"] != true {
		t.Fatalf("expected an in-band error, got %#v", res)
	}
}
