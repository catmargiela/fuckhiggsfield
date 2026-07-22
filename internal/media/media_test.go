package media

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePassesURLsThrough(t *testing.T) {
	for _, in := range []string{"https://x/y.png", "http://x/y.png", "data:image/png;base64,AAA"} {
		got, err := Resolve(in)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", in, err)
		}
		if got != in {
			t.Errorf("Resolve(%q) = %q, want unchanged", in, got)
		}
	}
}

func TestResolveInlinesLocalFileAsDataURI(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "pic.png")
	content := []byte("binary-png-content")
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(p)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Fatalf("got %q, want an image/png data URI", got[:min(60, len(got))])
	}
	payload := strings.SplitN(got, ",", 2)[1]
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("payload is not valid base64: %v", err)
	}
	if string(decoded) != string(content) {
		t.Errorf("round trip corrupted the file")
	}
}

func TestResolveRejectsMissingFile(t *testing.T) {
	_, err := Resolve(filepath.Join(t.TempDir(), "nope.png"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "neither a URL nor an existing file") {
		t.Errorf("error should explain the two accepted forms, got %q", err.Error())
	}
}

func TestResolveRejectsDirectory(t *testing.T) {
	if _, err := Resolve(t.TempDir()); err == nil {
		t.Fatal("expected an error for a directory")
	}
}

func TestResolveEmptyIsEmpty(t *testing.T) {
	got, err := Resolve("")
	if err != nil || got != "" {
		t.Fatalf("Resolve(\"\") = %q, %v", got, err)
	}
}

func TestResolveAll(t *testing.T) {
	got, err := ResolveAll([]string{"https://a/1.png", "https://b/2.png"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d, want 2", len(got))
	}
}

func TestSaveBase64WritesFile(t *testing.T) {
	dir := t.TempDir()
	b64 := base64.StdEncoding.EncodeToString([]byte("hello"))
	p, err := SaveBase64(dir, "job-1", "png", b64)
	if err != nil {
		t.Fatalf("SaveBase64: %v", err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "hello" {
		t.Errorf("content = %q", b)
	}
	if filepath.Ext(p) != ".png" {
		t.Errorf("path = %q, want a .png extension", p)
	}
}

// Two outputs from the same job must not overwrite each other.
func TestSaveDoesNotClobber(t *testing.T) {
	dir := t.TempDir()
	b64 := base64.StdEncoding.EncodeToString([]byte("one"))
	p1, err := SaveBase64(dir, "job", "png", b64)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := SaveBase64(dir, "job", "png", base64.StdEncoding.EncodeToString([]byte("two")))
	if err != nil {
		t.Fatal(err)
	}
	if p1 == p2 {
		t.Fatalf("second save reused the same path %q", p1)
	}
	b, _ := os.ReadFile(p1)
	if string(b) != "one" {
		t.Errorf("first file was overwritten: %q", b)
	}
}

func TestSaveBase64RejectsBadPayload(t *testing.T) {
	if _, err := SaveBase64(t.TempDir(), "x", "png", "!!!not-base64!!!"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestDownloadWritesFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "video-bytes")
	}))
	defer srv.Close()

	dir := t.TempDir()
	p, err := Download(context.Background(), srv.Client(), dir, "clip", "mp4", srv.URL+"/out.mp4")
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "video-bytes" {
		t.Errorf("content = %q", b)
	}
}

func TestDownloadReportsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := Download(context.Background(), srv.Client(), t.TempDir(), "x", "png", srv.URL+"/gone.png")
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("err = %v, want it to mention 404", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
