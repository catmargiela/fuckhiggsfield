// Package runpod talks to RunPod Public Endpoints.
//
//	POST {base}/{endpoint}/run          submit async, returns a job id
//	GET  {base}/{endpoint}/status/{id}  poll
//	POST {base}/{endpoint}/cancel/{id}  abort
//
// Every model takes its parameters nested under an "input" object, and returns
// results under "output" — but the shape of that output varies per model, so
// results are extracted structurally rather than against a fixed schema.
package runpod

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/catmargiela/fuckhiggsfield/internal/catalog"
	"github.com/catmargiela/fuckhiggsfield/internal/provider"
)

// Client is a RunPod public-endpoints client.
type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

// New builds a client. baseURL defaults to the public API root.
func New(baseURL, apiKey string, hc *http.Client) *Client {
	if baseURL == "" {
		baseURL = "https://api.runpod.ai/v2"
	}
	if hc == nil {
		hc = &http.Client{Timeout: 120 * time.Second}
	}
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), APIKey: apiKey, HTTP: hc}
}

// Name implements provider.Provider.
func (c *Client) Name() string { return "runpod" }

type submitResponse struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Output json.RawMessage `json:"output"`
	Error  any             `json:"error"`
}

// Create submits a generation asynchronously.
func (c *Client) Create(ctx context.Context, m catalog.Model, params map[string]any) (*provider.Job, error) {
	body := map[string]any{"input": buildInput(m, params)}
	raw, err := c.do(ctx, http.MethodPost, fmt.Sprintf("%s/%s/run", c.BaseURL, m.Endpoint), body)
	if err != nil {
		return nil, err
	}
	var sr submitResponse
	if err := json.Unmarshal(raw, &sr); err != nil {
		return nil, fmt.Errorf("parse RunPod submit response: %w", err)
	}
	if sr.ID == "" {
		return nil, fmt.Errorf("RunPod returned no job id: %s", truncate(string(raw), 400))
	}
	job := &provider.Job{
		ID:       sr.ID,
		Model:    m.ID,
		Provider: c.Name(),
		Endpoint: m.Endpoint,
		Status:   normalizeStatus(sr.Status),
		Created:  time.Now().UTC(),
		Raw:      raw,
	}
	applyOutput(job, m.Kind, sr.Output, sr.Error)
	return job, nil
}

// Get polls a job.
func (c *Client) Get(ctx context.Context, endpoint, id string) (*provider.Job, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("RunPod job %s has no endpoint recorded; re-run with the model name", id)
	}
	raw, err := c.do(ctx, http.MethodGet, fmt.Sprintf("%s/%s/status/%s", c.BaseURL, endpoint, id), nil)
	if err != nil {
		return nil, err
	}
	var sr submitResponse
	if err := json.Unmarshal(raw, &sr); err != nil {
		return nil, fmt.Errorf("parse RunPod status response: %w", err)
	}
	job := &provider.Job{
		ID:       id,
		Provider: c.Name(),
		Endpoint: endpoint,
		Status:   normalizeStatus(sr.Status),
		Raw:      raw,
	}
	applyOutput(job, "", sr.Output, sr.Error)
	return job, nil
}

// Cancel aborts a queued or running job.
func (c *Client) Cancel(ctx context.Context, endpoint, id string) error {
	_, err := c.do(ctx, http.MethodPost, fmt.Sprintf("%s/%s/cancel/%s", c.BaseURL, endpoint, id), nil)
	return err
}

func (c *Client) do(ctx context.Context, method, url string, body any) ([]byte, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call RunPod: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("read RunPod response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &provider.APIError{
			Provider:   "RunPod",
			StatusCode: resp.StatusCode,
			Message:    extractErrMessage(raw),
			Body:       string(raw),
		}
	}
	return raw, nil
}

// buildInput drops empty values so a model never receives a key it rejects,
// and folds the `prompt` shorthand into `messages` for chat-shaped models.
func buildInput(m catalog.Model, params map[string]any) map[string]any {
	in := make(map[string]any, len(params))
	for k, v := range params {
		if isEmpty(v) {
			continue
		}
		in[k] = v
	}
	if _, chat := m.Param("messages"); chat {
		if _, has := in["messages"]; !has {
			if p, ok := in["prompt"].(string); ok && p != "" {
				in["messages"] = []map[string]string{{"role": "user", "content": p}}
				delete(in, "prompt")
			}
		}
	}
	return in
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

func normalizeStatus(s string) provider.Status {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "IN_QUEUE", "QUEUED":
		return provider.StatusQueued
	case "IN_PROGRESS", "RUNNING":
		return provider.StatusRunning
	case "COMPLETED", "SUCCESS":
		return provider.StatusCompleted
	case "FAILED", "ERROR":
		return provider.StatusFailed
	case "CANCELLED", "CANCELED":
		return provider.StatusCancelled
	case "TIMED_OUT", "TIMEOUT":
		return provider.StatusTimedOut
	case "":
		return provider.StatusQueued
	}
	return provider.Status(strings.ToUpper(s))
}

func applyOutput(job *provider.Job, kind catalog.Kind, out json.RawMessage, apiErr any) {
	if apiErr != nil {
		if s, ok := apiErr.(string); ok && s != "" {
			job.Error = s
			job.Status = provider.StatusFailed
		} else if apiErr != nil {
			if b, err := json.Marshal(apiErr); err == nil && string(b) != "null" {
				job.Error = string(b)
				job.Status = provider.StatusFailed
			}
		}
	}
	if len(out) == 0 || string(out) == "null" {
		return
	}
	var v any
	if err := json.Unmarshal(out, &v); err != nil {
		return
	}
	job.Outputs = append(job.Outputs, extractOutputs(v, kind)...)
	if c, ok := findCost(v); ok {
		job.Cost = c
	}
	if job.Error == "" {
		if msg, ok := findErrorMessage(v); ok {
			job.Error = msg
			job.Status = provider.StatusFailed
		}
	}
}

// extractOutputs walks an arbitrary output payload and collects anything that
// looks like a produced artifact: URL-ish strings, base64 blobs, or text.
func extractOutputs(v any, kind catalog.Kind) []provider.Output {
	var outs []provider.Output
	switch t := v.(type) {
	case string:
		if looksLikeURL(t) {
			outs = append(outs, provider.Output{URL: t, Ext: provider.ExtFor(kind, t)})
		} else if t != "" {
			outs = append(outs, provider.Output{Text: t})
		}
	case []any:
		for _, item := range t {
			outs = append(outs, extractOutputs(item, kind)...)
		}
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			lk := strings.ToLower(k)
			switch {
			case lk == "cost" || lk == "seed" || lk == "delaytime" || lk == "executiontime":
				continue
			case strings.Contains(lk, "url") || lk == "image" || lk == "images" || lk == "video" || lk == "audio" || lk == "output":
				outs = append(outs, extractOutputs(t[k], kind)...)
			case lk == "base64" || strings.Contains(lk, "b64"):
				if s, ok := t[k].(string); ok && s != "" {
					outs = append(outs, provider.Output{B64: s, Ext: provider.ExtFor(kind, "")})
				}
			case lk == "text" || lk == "content" || lk == "generated_text":
				if s, ok := t[k].(string); ok && s != "" {
					outs = append(outs, provider.Output{Text: s})
				}
			}
		}
		// Chat-shaped replies: {"choices":[{"message":{"content":"..."}}]}
		if ch, ok := t["choices"].([]any); ok {
			for _, c := range ch {
				cm, ok := c.(map[string]any)
				if !ok {
					continue
				}
				if msg, ok := cm["message"].(map[string]any); ok {
					if s, ok := msg["content"].(string); ok && s != "" {
						outs = append(outs, provider.Output{Text: s})
					}
				}
				if s, ok := cm["text"].(string); ok && s != "" {
					outs = append(outs, provider.Output{Text: s})
				}
			}
		}
	}
	return dedupe(outs)
}

func dedupe(in []provider.Output) []provider.Output {
	seen := map[string]bool{}
	var out []provider.Output
	for _, o := range in {
		key := o.URL + "\x00" + o.Text + "\x00" + o.B64
		if key == "\x00\x00" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, o)
	}
	return out
}

func looksLikeURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func findCost(v any) (float64, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return 0, false
	}
	if c, ok := m["cost"].(float64); ok {
		return c, true
	}
	return 0, false
}

func findErrorMessage(v any) (string, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", false
	}
	for _, k := range []string{"error", "error_message", "message"} {
		if s, ok := m[k].(string); ok && s != "" {
			return s, true
		}
	}
	return "", false
}

func extractErrMessage(raw []byte) string {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	if s, ok := findErrorMessage(m); ok {
		return s
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
