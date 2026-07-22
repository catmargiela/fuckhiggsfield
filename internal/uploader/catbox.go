package uploader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// catbox uploads to catbox.moe (permanent) or litterbox (time-limited).
// Both take a multipart POST and answer with the bare URL as plain text.
type catbox struct {
	litterbox bool
	retention string
	userHash  string
	hc        *http.Client
	// api overrides the endpoint; empty means the real service.
	api string
}

const (
	catboxAPI    = "https://catbox.moe/user/api.php"
	litterboxAPI = "https://litterbox.catbox.moe/resources/internals/api.php"

	catboxMaxBytes    = 200 << 20
	litterboxMaxBytes = 1 << 30
)

func (c *catbox) Name() string {
	if c.litterbox {
		return BackendLitterbox
	}
	return BackendCatbox
}

func (c *catbox) Describe() string {
	if c.litterbox {
		return fmt.Sprintf("litterbox.catbox.moe — public URL, deleted after %s", c.retention)
	}
	if c.userHash != "" {
		return "catbox.moe — public URL, permanent, linked to your account"
	}
	return "catbox.moe — public URL, permanent, anonymous"
}

func (c *catbox) Upload(ctx context.Context, filename string, data []byte, contentType string) (string, error) {
	limit, host := catboxMaxBytes, "catbox"
	api := catboxAPI
	if c.litterbox {
		limit, host, api = litterboxMaxBytes, "litterbox", litterboxAPI
	}
	if c.api != "" {
		api = c.api
	}
	if len(data) > limit {
		return "", fmt.Errorf("%s accepts at most %d MiB, this file is %.1f MiB",
			host, limit>>20, float64(len(data))/(1<<20))
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("reqtype", "fileupload"); err != nil {
		return "", err
	}
	if c.litterbox {
		if err := mw.WriteField("time", c.retention); err != nil {
			return "", err
		}
	} else if c.userHash != "" {
		if err := mw.WriteField("userhash", c.userHash); err != nil {
			return "", err
		}
	}
	part, err := mw.CreateFormFile("fileToUpload", filename)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(data); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("User-Agent", "bara")

	resp, err := c.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload to %s: %w", host, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read %s response: %w", host, err)
	}
	out := strings.TrimSpace(string(raw))
	if resp.StatusCode != http.StatusOK {
		// catbox.moe periodically suspends uploads while litterbox keeps
		// working, so point at the alternatives rather than just the code.
		hint := ""
		if !c.litterbox {
			hint = "\n\ncatbox.moe pauses uploads from time to time. Try `--backend litterbox`" +
				" for a temporary URL, or configure `upload_backend s3` for a bucket you control."
		}
		return "", fmt.Errorf("%s returned HTTP %d: %s%s", host, resp.StatusCode, truncate(out, 300), hint)
	}
	// The API answers 200 with a plain error string on failure.
	if !strings.HasPrefix(out, "http://") && !strings.HasPrefix(out, "https://") {
		return "", fmt.Errorf("%s rejected the upload: %s", host, truncate(out, 300))
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
