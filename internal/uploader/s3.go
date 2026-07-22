package uploader

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// s3 uploads to any S3-compatible bucket: Cloudflare R2, AWS S3, Backblaze B2,
// MinIO. Requests are signed with AWS Signature Version 4, implemented here so
// the CLI stays dependency-free.
type s3 struct {
	endpoint  *url.URL
	bucket    string
	region    string
	accessKey string
	secretKey string
	prefix    string
	pathStyle bool
	publicURL string
	hc        *http.Client

	// now is overridable so signing can be tested against known vectors.
	now func() time.Time
}

func newS3(cfg Config) (Uploader, error) {
	var missing []string
	if cfg.S3Endpoint == "" {
		missing = append(missing, "s3_endpoint")
	}
	if cfg.S3Bucket == "" {
		missing = append(missing, "s3_bucket")
	}
	if cfg.S3AccessKey == "" {
		missing = append(missing, "s3_access_key_id")
	}
	if cfg.S3SecretKey == "" {
		missing = append(missing, "s3_secret_access_key")
	}
	if cfg.PublicBaseURL == "" {
		missing = append(missing, "upload_base_url")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("upload_backend=s3 needs: %s\n\nSet them with `bara settings set <name> <value>`", strings.Join(missing, ", "))
	}
	ep := cfg.S3Endpoint
	if !strings.Contains(ep, "://") {
		ep = "https://" + ep
	}
	u, err := url.Parse(ep)
	if err != nil {
		return nil, fmt.Errorf("s3_endpoint %q is not a URL: %w", cfg.S3Endpoint, err)
	}
	region := cfg.S3Region
	if region == "" {
		// Cloudflare R2 expects "auto"; it is also accepted by MinIO.
		region = "auto"
	}
	return &s3{
		endpoint:  u,
		bucket:    cfg.S3Bucket,
		region:    region,
		accessKey: cfg.S3AccessKey,
		secretKey: cfg.S3SecretKey,
		prefix:    strings.Trim(cfg.S3Prefix, "/"),
		pathStyle: cfg.S3PathStyle,
		publicURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
		hc:        cfg.HTTP,
		now:       time.Now,
	}, nil
}

func (s *s3) Name() string { return BackendS3 }

func (s *s3) Describe() string {
	return fmt.Sprintf("s3://%s at %s — you control retention", s.bucket, s.endpoint.Host)
}

func (s *s3) Upload(ctx context.Context, filename string, data []byte, contentType string) (string, error) {
	key := s.objectKey(filename)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	target, host := s.requestURL(key)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, target, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = int64(len(data))

	if err := s.sign(req, host, data); err != nil {
		return "", err
	}

	resp, err := s.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload to %s: %w", s.endpoint.Host, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("S3 upload failed (HTTP %d): %s", resp.StatusCode, truncate(strings.TrimSpace(string(body)), 400))
	}
	return s.publicURL + "/" + key, nil
}

// objectKey namespaces uploads by date so a bucket stays browsable.
func (s *s3) objectKey(filename string) string {
	name := strings.TrimLeft(filename, "/")
	day := s.now().UTC().Format("2006/01/02")
	parts := []string{}
	if s.prefix != "" {
		parts = append(parts, s.prefix)
	}
	parts = append(parts, day, name)
	return strings.Join(parts, "/")
}

// requestURL returns the signed endpoint and the Host header value to sign.
func (s *s3) requestURL(key string) (string, string) {
	encoded := encodePath(key)
	if s.pathStyle {
		host := s.endpoint.Host
		return fmt.Sprintf("%s://%s/%s/%s", s.endpoint.Scheme, host, s.bucket, encoded), host
	}
	host := s.bucket + "." + s.endpoint.Host
	return fmt.Sprintf("%s://%s/%s", s.endpoint.Scheme, host, encoded), host
}

func (s *s3) canonicalURI(key string) string {
	if s.pathStyle {
		return "/" + s.bucket + "/" + encodePath(key)
	}
	return "/" + encodePath(key)
}

// sign applies AWS Signature Version 4 to req.
func (s *s3) sign(req *http.Request, host string, payload []byte) error {
	now := s.now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	payloadHash := sha256Hex(payload)

	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	req.Host = host

	// Canonical headers must be sorted, lowercase, with collapsed values.
	headers := map[string]string{
		"host":                 host,
		"x-amz-content-sha256": payloadHash,
		"x-amz-date":           amzDate,
	}
	if ct := req.Header.Get("Content-Type"); ct != "" {
		headers["content-type"] = ct
	}
	names := make([]string, 0, len(headers))
	for k := range headers {
		names = append(names, k)
	}
	sort.Strings(names)

	var canonicalHeaders strings.Builder
	for _, n := range names {
		canonicalHeaders.WriteString(n)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(strings.TrimSpace(headers[n]))
		canonicalHeaders.WriteString("\n")
	}
	signedHeaders := strings.Join(names, ";")

	key := strings.TrimPrefix(req.URL.Path, "/")
	if s.pathStyle {
		key = strings.TrimPrefix(key, s.bucket+"/")
	}
	canonicalRequest := strings.Join([]string{
		http.MethodPut,
		s.canonicalURI(decodePath(key)),
		"", // no query string
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	}, "\n")

	scope := strings.Join([]string{dateStamp, s.region, "s3", "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := hmacSHA256(
		hmacSHA256(
			hmacSHA256(
				hmacSHA256([]byte("AWS4"+s.secretKey), dateStamp),
				s.region),
			"s3"),
		"aws4_request")
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))

	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.accessKey, scope, signedHeaders, signature))
	return nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// encodePath percent-encodes each path segment per RFC 3986, leaving the
// separators intact. AWS rejects Go's default encoding of some characters.
func encodePath(p string) string {
	segments := strings.Split(p, "/")
	for i, seg := range segments {
		segments[i] = encodeSegment(seg)
	}
	return strings.Join(segments, "/")
}

func encodeSegment(s string) string {
	const unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.~"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if strings.IndexByte(unreserved, c) >= 0 {
			b.WriteByte(c)
			continue
		}
		fmt.Fprintf(&b, "%%%02X", c)
	}
	return b.String()
}

func decodePath(p string) string {
	if out, err := url.PathUnescape(p); err == nil {
		return out
	}
	return p
}
