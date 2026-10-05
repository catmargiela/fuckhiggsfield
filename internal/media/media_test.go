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
