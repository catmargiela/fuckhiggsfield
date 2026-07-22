package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catmargiela/fuckhiggsfield/internal/catalog"
	"github.com/catmargiela/fuckhiggsfield/internal/provider"
)

func imageModel() catalog.Model {
	return catalog.Model{ID: "gpt_image_2", Endpoint: "gpt-image-2", Provider: "openai", Kind: catalog.KindImage}
}

func TestCreateGenerationSendsJSON(t *testing.T) {
	var (
		gotPath string
		gotBody map[string]any
		gotCT   string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotCT = r.URL.Path, r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		io.WriteString(w, `{"created":1,"output_format":"png","data":[{"b64_json":"`+base64.StdEncoding.EncodeToString([]byte("PNG"))+`"}]}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "sk-test", srv.Client())
	job, err := c.Create(context.Background(), imageModel(), map[string]any{
		"prompt": "a cat in a spaceship",
		"size":   "1024x1024",
		"n":      int64(1),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gotPath != "/images/generations" {
		t.Errorf("path = %s", gotPath)
	}
	if !strings.HasPrefix(gotCT, "application/json") {
		t.Errorf("content-type = %s", gotCT)
	}
	if gotBody["model"] != "gpt-image-2" {
		t.Errorf("model = %v, want gpt-image-2", gotBody["model"])
	}
	if gotBody["prompt"] != "a cat in a spaceship" {
		t.Errorf("prompt = %v", gotBody["prompt"])
	}
	if job.Status != provider.StatusCompleted {
		t.Errorf("status = %s, want COMPLETED (the images API is synchronous)", job.Status)
	}
	if len(job.Outputs) != 1 || job.Outputs[0].B64 == "" {
		t.Fatalf("outputs = %#v", job.Outputs)
	}
	if job.Outputs[0].Ext != "png" {
		t.Errorf("ext = %q", job.Outputs[0].Ext)
	}
}

// Supplying --image must route to the edits endpoint as multipart.
func TestCreateWithImageUsesEditsMultipart(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "in.png")
	if err := os.WriteFile(src, []byte("fake-png-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	var (
		gotPath   string
		gotFields = map[string]string{}
		gotFiles  []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Errorf("parse content type: %v", err)
			return
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			b, _ := io.ReadAll(part)
			if part.FileName() != "" {
				gotFiles = append(gotFiles, part.FormName()+":"+part.FileName()+":"+string(b))
			} else {
				gotFields[part.FormName()] = string(b)
			}
		}
		io.WriteString(w, `{"created":1,"data":[{"b64_json":"AAAA"}]}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "sk-test", srv.Client())
	_, err := c.Create(context.Background(), imageModel(), map[string]any{
		"prompt":  "make it snow",
		"image":   []string{src},
		"quality": "high",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gotPath != "/images/edits" {
		t.Fatalf("path = %s, want /images/edits", gotPath)
	}
	if gotFields["model"] != "gpt-image-2" {
		t.Errorf("model field = %q", gotFields["model"])
	}
	if gotFields["prompt"] != "make it snow" {
		t.Errorf("prompt field = %q", gotFields["prompt"])
	}
	if gotFields["quality"] != "high" {
		t.Errorf("quality field = %q", gotFields["quality"])
	}
	if len(gotFiles) != 1 || !strings.HasPrefix(gotFiles[0], "image[]:in.png:") {
		t.Fatalf("files = %#v", gotFiles)
	}
	if !strings.HasSuffix(gotFiles[0], "fake-png-bytes") {
		t.Errorf("file content not forwarded: %q", gotFiles[0])
	}
}

func TestCreateSurfacesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error":{"message":"Invalid size","type":"invalid_request_error"}}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "sk", srv.Client())
	_, err := c.Create(context.Background(), imageModel(), map[string]any{"prompt": "x"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "Invalid size") {
		t.Errorf("error should carry the API message, got %q", err.Error())
	}
}

func TestCreateFlagsEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"created":1,"data":[]}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "sk", srv.Client())
	job, err := c.Create(context.Background(), imageModel(), map[string]any{"prompt": "x"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if job.Status != provider.StatusFailed {
		t.Errorf("status = %s, want FAILED when no image came back", job.Status)
	}
}

func TestGetIsNotSupported(t *testing.T) {
	c := New("", "sk", nil)
	if _, err := c.Get(context.Background(), "gpt-image-2", "openai-1"); err == nil {
		t.Fatal("expected an explanatory error, since OpenAI image jobs are synchronous")
	}
}
