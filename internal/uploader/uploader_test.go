package uploader

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewDisabledByDefault(t *testing.T) {
	for _, backend := range []string{"", "none", "NONE", " none "} {
		up, err := New(Config{Backend: backend})
		if err != nil {
			t.Fatalf("New(%q): %v", backend, err)
		}
		if up != nil {
			t.Errorf("New(%q) returned an uploader; uploading must be opt-in", backend)
		}
	}
}

func TestNewUnknownBackend(t *testing.T) {
	_, err := New(Config{Backend: "dropbox"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "catbox") {
		t.Errorf("error should list the valid backends, got %q", err.Error())
	}
}

func TestNewS3ReportsEveryMissingSetting(t *testing.T) {
	_, err := New(Config{Backend: BackendS3})
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"s3_endpoint", "s3_bucket", "s3_access_key_id", "s3_secret_access_key", "upload_base_url"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should name %s, got %q", want, err.Error())
		}
	}
}

func TestNewLitterboxRejectsBadRetention(t *testing.T) {
	if _, err := New(Config{Backend: BackendLitterbox, LitterboxTime: "5h"}); err == nil {
		t.Fatal("expected an error for an unsupported retention window")
	}
	up, err := New(Config{Backend: BackendLitterbox})
	if err != nil {
		t.Fatalf("default retention should be valid: %v", err)
	}
	if !strings.Contains(up.Describe(), "72h") {
		t.Errorf("default retention should be 72h, got %q", up.Describe())
	}
}

func TestCatboxUploadPostsMultipart(t *testing.T) {
	var (
		fields = map[string]string{}
		file   string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Errorf("content type: %v", err)
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
				file = part.FileName() + ":" + string(b)
			} else {
				fields[part.FormName()] = string(b)
			}
		}
		io.WriteString(w, "https://files.catbox.moe/abc123.png\n")
	}))
	defer srv.Close()

	c := &catbox{hc: srv.Client(), api: srv.URL}
	url, err := c.Upload(context.Background(), "pic.png", []byte("data"), "image/png")
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if url != "https://files.catbox.moe/abc123.png" {
		t.Errorf("url = %q (trailing whitespace should be trimmed)", url)
	}
	if fields["reqtype"] != "fileupload" {
		t.Errorf("reqtype = %q", fields["reqtype"])
	}
	if file != "pic.png:data" {
		t.Errorf("file part = %q", file)
	}
}

func TestCatboxTreatsPlainTextErrorAsFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The API answers 200 with a bare message when it rejects an upload.
		io.WriteString(w, "File too large")
	}))
	defer srv.Close()

	c := &catbox{hc: srv.Client(), api: srv.URL}
	_, err := c.Upload(context.Background(), "pic.png", []byte("x"), "image/png")
	if err == nil {
		t.Fatal("a non-URL 200 response must be treated as an error")
	}
	if !strings.Contains(err.Error(), "File too large") {
		t.Errorf("error should carry the server message, got %q", err.Error())
	}
}

func TestCatboxRejectsOversizeBeforeUploading(t *testing.T) {
	c := &catbox{hc: http.DefaultClient}
	_, err := c.Upload(context.Background(), "big.bin", make([]byte, catboxMaxBytes+1), "application/octet-stream")
	if err == nil {
		t.Fatal("expected a size error")
	}
	if !strings.Contains(err.Error(), "200 MiB") {
		t.Errorf("error should state the limit, got %q", err.Error())
	}
}

func TestLitterboxSendsRetention(t *testing.T) {
	var fields = map[string]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			b, _ := io.ReadAll(part)
			if part.FileName() == "" {
				fields[part.FormName()] = string(b)
			}
		}
		io.WriteString(w, "https://litter.catbox.moe/xyz.png")
	}))
	defer srv.Close()

	c := &catbox{litterbox: true, retention: "24h", hc: srv.Client(), api: srv.URL}
	if _, err := c.Upload(context.Background(), "p.png", []byte("x"), "image/png"); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if fields["time"] != "24h" {
		t.Errorf("time field = %q, want 24h", fields["time"])
	}
}

// The signature is cross-checked against a value computed by an independent
// implementation; a silent change in canonicalization would break this.
func TestS3SignatureMatchesKnownVector(t *testing.T) {
	const (
		wantSig     = "6f79b4150c98ce92466f8327962db1287ae45a09876de333a4b8300059e01798"
		wantPayload = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	)
	var gotAuth, gotDate, gotSHA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotDate = r.Header.Get("X-Amz-Date")
		gotSHA = r.Header.Get("X-Amz-Content-Sha256")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	up, err := New(Config{
		Backend:       BackendS3,
		S3Endpoint:    "https://acct.r2.cloudflarestorage.com",
		S3Bucket:      "test-bucket",
		S3Region:      "auto",
		S3AccessKey:   "AKIAIOSFODNN7EXAMPLE",
		S3SecretKey:   "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		S3PathStyle:   true,
		PublicBaseURL: "https://cdn.example.com",
		HTTP:          srv.Client(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s := up.(*s3)
	s.now = func() time.Time { return time.Date(2026, 7, 22, 18, 0, 0, 0, time.UTC) }
	// Send the request to the test server while signing the real host.
	s.endpoint.Host = "acct.r2.cloudflarestorage.com"
	s.hc = &http.Client{Transport: redirectTo(srv.URL, srv.Client().Transport)}

	url, err := s.Upload(context.Background(), "pic.png", []byte("hello"), "image/png")
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if gotSHA != wantPayload {
		t.Errorf("x-amz-content-sha256 = %s, want %s", gotSHA, wantPayload)
	}
	if gotDate != "20260722T180000Z" {
		t.Errorf("x-amz-date = %s", gotDate)
	}
	if !strings.Contains(gotAuth, "Signature="+wantSig) {
		t.Errorf("signature mismatch\n got %s\nwant Signature=%s", gotAuth, wantSig)
	}
	if !strings.Contains(gotAuth, "Credential=AKIAIOSFODNN7EXAMPLE/20260722/auto/s3/aws4_request") {
		t.Errorf("credential scope wrong: %s", gotAuth)
	}
	if !strings.Contains(gotAuth, "SignedHeaders=content-type;host;x-amz-content-sha256;x-amz-date") {
		t.Errorf("signed headers wrong: %s", gotAuth)
	}
	if want := "https://cdn.example.com/2026/07/22/pic.png"; url != want {
		t.Errorf("public url = %q, want %q", url, want)
	}
}

func TestS3ObjectKeyUsesPrefixAndDate(t *testing.T) {
	s := &s3{prefix: "renders", now: func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }}
	if got := s.objectKey("a.png"); got != "renders/2026/01/02/a.png" {
		t.Errorf("objectKey = %q", got)
	}
	s.prefix = ""
	if got := s.objectKey("a.png"); got != "2026/01/02/a.png" {
		t.Errorf("objectKey without prefix = %q", got)
	}
}

func TestS3RequestURLStyles(t *testing.T) {
	base := func(pathStyle bool) *s3 {
		u, _ := url.Parse("https://acct.r2.cloudflarestorage.com")
		return &s3{endpoint: u, bucket: "b", pathStyle: pathStyle}
	}
	got, host := base(true).requestURL("x/y.png")
	if got != "https://acct.r2.cloudflarestorage.com/b/x/y.png" || host != "acct.r2.cloudflarestorage.com" {
		t.Errorf("path style: %q %q", got, host)
	}
	got, host = base(false).requestURL("x/y.png")
	if got != "https://b.acct.r2.cloudflarestorage.com/x/y.png" || host != "b.acct.r2.cloudflarestorage.com" {
		t.Errorf("virtual host style: %q %q", got, host)
	}
}

func TestS3ReportsUploadFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, "<Error><Code>SignatureDoesNotMatch</Code></Error>")
	}))
	defer srv.Close()

	up, err := New(Config{
		Backend: BackendS3, S3Endpoint: srv.URL, S3Bucket: "b",
		S3AccessKey: "a", S3SecretKey: "s", PublicBaseURL: "https://cdn",
		// Path style keeps the bucket out of the hostname, which a bare IP
		// test server cannot serve.
		S3PathStyle: true,
		HTTP:        srv.Client(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = up.Upload(context.Background(), "x.png", []byte("d"), "image/png")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "SignatureDoesNotMatch") {
		t.Errorf("error should carry the body, got %q", err.Error())
	}
}

func TestEncodePathEscapesPerSegment(t *testing.T) {
	if got := encodePath("a b/c+d/e.png"); got != "a%20b/c%2Bd/e.png" {
		t.Errorf("encodePath = %q", got)
	}
}

// redirectTo sends requests to addr while leaving the signed Host header alone,
// so signing can be verified against the production hostname.
func redirectTo(addr string, rt http.RoundTripper) http.RoundTripper {
	target, _ := url.Parse(addr)
	if rt == nil {
		rt = http.DefaultTransport
	}
	return roundTripFunc(func(r *http.Request) (*http.Response, error) {
		clone := r.Clone(r.Context())
		clone.URL.Scheme = target.Scheme
		clone.URL.Host = target.Host
		return rt.RoundTrip(clone)
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
