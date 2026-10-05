// Package openai talks to the OpenAI Images API.
//
//	POST {base}/images/generations  text -> image (JSON)
//	POST {base}/images/edits        image(s) -> image (multipart)
//
// Both are synchronous, so Create returns an already-terminal job. GPT Image
// models always answer with base64 payloads rather than URLs.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/catmargiela/fuckhiggsfield/internal/catalog"
	"github.com/catmargiela/fuckhiggsfield/internal/provider"
)

// Client is an OpenAI Images API client.
type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

// New builds a client. baseURL defaults to the public API root.
func New(baseURL, apiKey string, hc *http.Client) *Client {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if hc == nil {
		hc = &http.Client{Timeout: 300 * time.Second}
	}
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), APIKey: apiKey, HTTP: hc}
}

// Name implements provider.Provider.
func (c *Client) Name() string { return "openai" }

type imagesResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		B64JSON       string `json:"b64_json"`
		URL           string `json:"url"`
		RevisedPrompt string `json:"revised_prompt"`
	} `json:"data"`
	OutputFormat string `json:"output_format"`
	Error        *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error"`
}

// Create runs a generation or an edit, depending on whether input images were
// supplied, and returns a terminal job.
func (c *Client) Create(ctx context.Context, m catalog.Model, params map[string]any) (*provider.Job, error) {
	images := stringSlice(params["image"])
	isEdit := len(images) > 0

	var (
		raw []byte
		err error
	)
	if isEdit {
		raw, err = c.createEdit(ctx, m, params, images)
	} else {
		raw, err = c.createGeneration(ctx, m, params)
	}
	if err != nil {
		return nil, err
	}

	var ir imagesResponse
	if err := json.Unmarshal(raw, &ir); err != nil {
		return nil, fmt.Errorf("parse OpenAI response: %w", err)
	}
	job := &provider.Job{
		ID:       fmt.Sprintf("openai-%d", time.Now().UnixNano()),
		Model:    m.ID,
		Provider: c.Name(),
		Endpoint: m.Endpoint,
		Status:   provider.StatusCompleted,
		Created:  time.Now().UTC(),
		Raw:      redactImages(raw),
	}
	if ir.Error != nil && ir.Error.Message != "" {
		job.Status = provider.StatusFailed
		job.Error = ir.Error.Message
		return job, nil
	}
	ext := ir.OutputFormat
	if ext == "" {
		ext = stringOr(params["output_format"], "png")
	}
	for _, d := range ir.Data {
		out := provider.Output{Ext: ext}
		switch {
		case d.B64JSON != "":
			out.B64 = d.B64JSON
		case d.URL != "":
			out.URL = d.URL
			out.Ext = provider.ExtFor(catalog.KindImage, d.URL)
		default:
			continue
		}
		job.Outputs = append(job.Outputs, out)
	}
	if len(job.Outputs) == 0 {
		job.Status = provider.StatusFailed
		if job.Error == "" {
			job.Error = "OpenAI returned no image data"
		}
	}
	return job, nil
}

// Get has nothing to poll: the Images API is synchronous, so a job id can only
// be resolved from the local job store.
func (c *Client) Get(ctx context.Context, endpoint, id string) (*provider.Job, error) {
	return nil, fmt.Errorf("OpenAI image jobs complete synchronously and are not re-fetchable; `bara generate get %s` reads the local job store", id)
}

func (c *Client) createGeneration(ctx context.Context, m catalog.Model, params map[string]any) ([]byte, error) {
	body := map[string]any{"model": m.Endpoint}
	for k, v := range params {
		switch k {
		case "image", "mask":
			continue
		}
		if isEmpty(v) {
			continue
		}
		body[k] = v
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/images/generations", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.send(req)
}

func (c *Client) createEdit(ctx context.Context, m catalog.Model, params map[string]any, images []string) ([]byte, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	if err := mw.WriteField("model", m.Endpoint); err != nil {
		return nil, err
	}
	for _, ref := range images {
		if err := attachFile(ctx, mw, "image[]", ref, c.HTTP); err != nil {
			return nil, err
		}
	}
	if mask := stringOr(params["mask"], ""); mask != "" {
		if err := attachFile(ctx, mw, "mask", mask, c.HTTP); err != nil {
			return nil, err
		}
	}
	for k, v := range params {
		switch k {
		case "image", "mask":
			continue
		}
		if isEmpty(v) {
			continue
		}
		if err := mw.WriteField(k, scalarString(v)); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/images/edits", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return c.send(req)
}

func (c *Client) send(req *http.Request) ([]byte, error) {
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call OpenAI: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 128<<20))
	if err != nil {
		return nil, fmt.Errorf("read OpenAI response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &provider.APIError{
			Provider:   "OpenAI",
			StatusCode: resp.StatusCode,
			Message:    errMessage(raw),
			Body:       string(raw),
		}
	}
	return raw, nil
}

// attachFile copies a local path or an http(s) URL into a multipart field.
func attachFile(ctx context.Context, mw *multipart.Writer, field, ref string, hc *http.Client) error {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ref, nil)
		if err != nil {
			return err
		}
		resp, err := hc.Do(req)
		if err != nil {
			return fmt.Errorf("fetch %s: %w", ref, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("fetch %s: HTTP %d", ref, resp.StatusCode)
		}
		name := filepath.Base(strings.SplitN(ref, "?", 2)[0])
		if name == "" || name == "/" || name == "." {
			name = "image.png"
		}
		w, err := mw.CreateFormFile(field, name)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, io.LimitReader(resp.Body, 128<<20))
		return err
	}
	f, err := os.Open(ref)
	if err != nil {
		return fmt.Errorf("open %s: %w", ref, err)
	}
	defer f.Close()
	w, err := mw.CreateFormFile(field, filepath.Base(ref))
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

// redactImages returns the response with each b64_json payload replaced by its
// size. The bytes live in the job outputs; keeping them in Raw as well bloats
// the job store by megabytes per image. A response it cannot rewrite is
// dropped rather than kept whole, for the same reason.
func redactImages(raw []byte) json.RawMessage {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil
	}
	data, ok := doc["data"].([]any)
	if !ok {
		return raw
	}
	for _, item := range data {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if b64, ok := entry["b64_json"].(string); ok && b64 != "" {
			entry["b64_json"] = fmt.Sprintf("<%d base64 chars omitted>", len(b64))
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return nil
	}
	return out
}

func errMessage(raw []byte) string {
	var e struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &e); err == nil {
		return e.Error.Message
	}
	return ""
}

func stringSlice(v any) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		if t == "" {
			return nil
		}
		return []string{t}
	case []string:
		return t
	case []any:
		var out []string
		for _, x := range t {
			if s, ok := x.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func stringOr(v any, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}

func scalarString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case int:
		return strconv.Itoa(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func isEmpty(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return t == ""
	case []string:
		return len(t) == 0
	case []any:
		return len(t) == 0
	}
	return false
}
