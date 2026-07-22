package runpod

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/catmargiela/fuckhiggsfield/internal/catalog"
	"github.com/catmargiela/fuckhiggsfield/internal/provider"
)

func model(id, endpoint string, params ...catalog.Param) catalog.Model {
	return catalog.Model{ID: id, Endpoint: endpoint, Provider: "runpod", Kind: catalog.KindImage, Params: params}
}

func TestCreatePostsInputEnvelope(t *testing.T) {
	var (
		gotPath   string
		gotAuth   string
		gotBody   map[string]any
		gotMethod string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth, gotMethod = r.URL.Path, r.Header.Get("Authorization"), r.Method
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"job-1","status":"IN_QUEUE"}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-key", srv.Client())
	m := model("qwen_image", "qwen-image-t2i")
	job, err := c.Create(context.Background(), m, map[string]any{"prompt": "a cat", "seed": int64(7), "negative_prompt": ""})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/qwen-image-t2i/run" {
		t.Errorf("path = %s, want /qwen-image-t2i/run", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("auth = %q", gotAuth)
	}
	input, ok := gotBody["input"].(map[string]any)
	if !ok {
		t.Fatalf("body has no input object: %#v", gotBody)
	}
	if input["prompt"] != "a cat" {
		t.Errorf("prompt = %v", input["prompt"])
	}
	if _, present := input["negative_prompt"]; present {
		t.Error("empty values should be dropped from the request")
	}
	if job.ID != "job-1" || job.Status != provider.StatusQueued {
		t.Errorf("job = %+v", job)
	}
	if job.Endpoint != "qwen-image-t2i" {
		t.Errorf("endpoint = %q; needed to poll status later", job.Endpoint)
	}
}

func TestGetExtractsImageURLAndCost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/qwen-image-t2i/status/job-1" {
			t.Errorf("unexpected status path %s", r.URL.Path)
		}
		io.WriteString(w, `{"id":"job-1","status":"COMPLETED","output":{"cost":0.021,"image_url":"https://image.runpod.ai/out.png"}}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "k", srv.Client())
	job, err := c.Get(context.Background(), "qwen-image-t2i", "job-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if job.Status != provider.StatusCompleted {
		t.Fatalf("status = %s", job.Status)
	}
	if len(job.Outputs) != 1 {
		t.Fatalf("outputs = %#v, want 1", job.Outputs)
	}
	if job.Outputs[0].URL != "https://image.runpod.ai/out.png" {
		t.Errorf("url = %q", job.Outputs[0].URL)
	}
	if job.Outputs[0].Ext != "png" {
		t.Errorf("ext = %q, want png", job.Outputs[0].Ext)
	}
	if job.Cost != 0.021 {
		t.Errorf("cost = %v", job.Cost)
	}
}

func TestGetExtractsMultipleOutputsFromArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"status":"COMPLETED","output":{"images":["https://x/1.jpg","https://x/2.jpg"]}}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "k", srv.Client())
	job, err := c.Get(context.Background(), "e", "id")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(job.Outputs) != 2 {
		t.Fatalf("outputs = %#v, want 2", job.Outputs)
	}
}

func TestGetSurfacesFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"status":"FAILED","error":"prompt rejected by safety checker"}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "k", srv.Client())
	job, err := c.Get(context.Background(), "e", "id")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if job.Status != provider.StatusFailed {
		t.Errorf("status = %s, want FAILED", job.Status)
	}
	if !strings.Contains(job.Error, "safety checker") {
		t.Errorf("error = %q", job.Error)
	}
}

func TestHTTPErrorIsTyped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":"invalid api key"}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "bad", srv.Client())
	_, err := c.Create(context.Background(), model("m", "e"), map[string]any{"prompt": "x"})
	if err == nil {
		t.Fatal("expected an error")
	}
	var apiErr *provider.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *provider.APIError: %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d", apiErr.StatusCode)
	}
	if !strings.Contains(err.Error(), "rejected the API key") {
		t.Errorf("message should name the cause, got %q", err.Error())
	}
}

func asAPIError(err error, target **provider.APIError) bool {
	e, ok := err.(*provider.APIError)
	if ok {
		*target = e
	}
	return ok
}

// Chat-shaped models take `messages`; `prompt` is a shorthand the client folds in.
func TestBuildInputFoldsPromptIntoMessages(t *testing.T) {
	m := model("granite_4", "granite-4-0-h-small",
		catalog.Param{Name: "prompt", Type: catalog.TypeString},
		catalog.Param{Name: "messages", Type: catalog.TypeJSON},
	)
	in := buildInput(m, map[string]any{"prompt": "hello"})
	if _, has := in["prompt"]; has {
		t.Error("prompt should be replaced by messages")
	}
	msgs, ok := in["messages"].([]map[string]string)
	if !ok || len(msgs) != 1 || msgs[0]["content"] != "hello" || msgs[0]["role"] != "user" {
		t.Fatalf("messages = %#v", in["messages"])
	}
}

func TestBuildInputKeepsExplicitMessages(t *testing.T) {
	m := model("granite_4", "granite-4-0-h-small",
		catalog.Param{Name: "prompt", Type: catalog.TypeString},
		catalog.Param{Name: "messages", Type: catalog.TypeJSON},
	)
	explicit := []any{map[string]any{"role": "system", "content": "be terse"}}
	in := buildInput(m, map[string]any{"prompt": "hello", "messages": explicit})
	if got := in["messages"]; !sameLen(got, 1) {
		t.Fatalf("explicit messages were overwritten: %#v", got)
	}
}

func sameLen(v any, n int) bool {
	s, ok := v.([]any)
	return ok && len(s) == n
}

func TestNormalizeStatus(t *testing.T) {
	cases := map[string]provider.Status{
		"IN_QUEUE":    provider.StatusQueued,
		"IN_PROGRESS": provider.StatusRunning,
		"COMPLETED":   provider.StatusCompleted,
		"FAILED":      provider.StatusFailed,
		"CANCELLED":   provider.StatusCancelled,
		"TIMED_OUT":   provider.StatusTimedOut,
		"":            provider.StatusQueued,
	}
	for in, want := range cases {
		if got := normalizeStatus(in); got != want {
			t.Errorf("normalizeStatus(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestTerminalStatuses(t *testing.T) {
	if provider.StatusQueued.Terminal() || provider.StatusRunning.Terminal() {
		t.Error("queued/running must not be terminal")
	}
	for _, s := range []provider.Status{provider.StatusCompleted, provider.StatusFailed, provider.StatusCancelled, provider.StatusTimedOut} {
		if !s.Terminal() {
			t.Errorf("%s should be terminal", s)
		}
	}
}
