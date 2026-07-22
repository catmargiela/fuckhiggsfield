// Package mcpserver exposes the generation catalog over the Model Context
// Protocol, so an agent can list models and run generations as tools.
//
// Transport is newline-delimited JSON-RPC 2.0 over stdio, which is what MCP
// stdio clients speak.
package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/catmargiela/fuckhiggsfield/internal/catalog"
	"github.com/catmargiela/fuckhiggsfield/internal/jobs"
	"github.com/catmargiela/fuckhiggsfield/internal/media"
	"github.com/catmargiela/fuckhiggsfield/internal/provider"
)

// DefaultProtocolVersion is advertised when the client does not pin one.
const DefaultProtocolVersion = "2025-06-18"

// Deps are the capabilities the server needs from the host CLI.
type Deps struct {
	Catalog *catalog.Catalog
	// ProviderFor resolves a backend client for a model.
	ProviderFor func(catalog.Model) (provider.Provider, error)
	// OutputDir is where generated artifacts are written.
	OutputDir string
	// Download saves an output artifact, returning its local path.
	Download func(ctx context.Context, dir, name, ext, url string) (string, error)
	// ServerName and ServerVersion identify this server to the client.
	ServerName    string
	ServerVersion string
}

// Server speaks MCP over a reader/writer pair.
type Server struct {
	deps Deps
	out  *json.Encoder
	mu   sync.Mutex
}

// New builds a server writing responses to w.
func New(deps Deps, w io.Writer) *Server {
	if deps.ServerName == "" {
		deps.ServerName = "bara"
	}
	if deps.ServerVersion == "" {
		deps.ServerVersion = "dev"
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &Server{deps: deps, out: enc}
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// JSON-RPC 2.0 reserved codes.
const (
	codeParse          = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternal       = -32603
)

// Serve reads requests until the input closes.
func (s *Server) Serve(ctx context.Context, r io.Reader) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 32<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var req request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.send(response{JSONRPC: "2.0", Error: &rpcError{Code: codeParse, Message: "invalid JSON: " + err.Error()}})
			continue
		}
		s.handle(ctx, req)
	}
	return sc.Err()
}

func (s *Server) send(resp response) {
	resp.JSONRPC = "2.0"
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.out.Encode(resp)
}

func (s *Server) reply(id json.RawMessage, result any) {
	s.send(response{ID: id, Result: result})
}

func (s *Server) fail(id json.RawMessage, code int, msg string) {
	s.send(response{ID: id, Error: &rpcError{Code: code, Message: msg}})
}

func (s *Server) handle(ctx context.Context, req request) {
	// Notifications carry no id and must not be answered.
	isNotification := len(req.ID) == 0

	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &p)
		version := p.ProtocolVersion
		if version == "" {
			version = DefaultProtocolVersion
		}
		s.reply(req.ID, map[string]any{
			"protocolVersion": version,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": s.deps.ServerName, "version": s.deps.ServerVersion},
		})
	case "notifications/initialized", "initialized":
		// Nothing to do.
	case "ping":
		if !isNotification {
			s.reply(req.ID, map[string]any{})
		}
	case "tools/list":
		s.reply(req.ID, map[string]any{"tools": s.tools()})
	case "tools/call":
		if isNotification {
			return
		}
		s.callTool(ctx, req)
	default:
		if isNotification {
			return
		}
		s.fail(req.ID, codeMethodNotFound, "unknown method "+req.Method)
	}
}

type toolDef struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func (s *Server) tools() []toolDef {
	return []toolDef{
		{
			Name:        "list_models",
			Title:       "List models",
			Description: "List available generation models. Filter by kind (image, video, audio, text), provider (openai, runpod), or a substring query.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"kind":     map[string]any{"type": "string", "enum": []string{"image", "video", "audio", "text"}},
					"provider": map[string]any{"type": "string", "enum": []string{"openai", "runpod"}},
					"query":    map[string]any{"type": "string", "description": "substring match on model id or name"},
				},
			},
		},
		{
			Name:        "get_model",
			Title:       "Get model schema",
			Description: "Return one model's full parameter schema: types, defaults, allowed values, and price.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"model": map[string]any{"type": "string", "description": "model id, e.g. gpt_image_2"}},
				"required":   []string{"model"},
			},
		},
		{
			Name:  "generate",
			Title: "Generate media",
			Description: "Run a generation and wait for the result. Returns the saved file path(s) and any URLs. " +
				"Call get_model first to learn a model's parameters. Media parameters accept a local path or a URL.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"model":  map[string]any{"type": "string", "description": "model id"},
					"params": map[string]any{"type": "object", "description": "model parameters, e.g. {\"prompt\": \"a cat\"}"},
					"wait": map[string]any{
						"type": "boolean", "default": true,
						"description": "block until the job finishes",
					},
					"timeout_seconds": map[string]any{"type": "integer", "default": 600},
				},
				"required": []string{"model", "params"},
			},
		},
		{
			Name:        "get_job",
			Title:       "Get job",
			Description: "Look up a previously submitted job by id, refreshing it from the provider when it is still running.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"job_id": map[string]any{"type": "string"}},
				"required":   []string{"job_id"},
			},
		},
		{
			Name:        "list_jobs",
			Title:       "List jobs",
			Description: "List recently submitted generation jobs with their status and outputs.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status": map[string]any{"type": "string"},
					"limit":  map[string]any{"type": "integer", "default": 20},
				},
			},
		},
	}
}

type callParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (s *Server) callTool(ctx context.Context, req request) {
	var cp callParams
	if err := json.Unmarshal(req.Params, &cp); err != nil {
		s.fail(req.ID, codeInvalidParams, "invalid params: "+err.Error())
		return
	}
	result, err := s.dispatch(ctx, cp)
	if err != nil {
		// Tool failures are reported in-band so the model can react to them,
		// rather than as protocol errors.
		s.reply(req.ID, map[string]any{
			"content": []map[string]any{{"type": "text", "text": err.Error()}},
			"isError": true,
		})
		return
	}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		s.fail(req.ID, codeInternal, err.Error())
		return
	}
	s.reply(req.ID, map[string]any{
		"content":           []map[string]any{{"type": "text", "text": string(b)}},
		"structuredContent": result,
		"isError":           false,
	})
}

func (s *Server) dispatch(ctx context.Context, cp callParams) (any, error) {
	switch cp.Name {
	case "list_models":
		var a struct {
			Kind     string `json:"kind"`
			Provider string `json:"provider"`
			Query    string `json:"query"`
		}
		_ = json.Unmarshal(cp.Arguments, &a)
		models := s.deps.Catalog.Filter(catalog.Kind(a.Kind), a.Provider, a.Query)
		out := make([]map[string]any, 0, len(models))
		for _, m := range models {
			out = append(out, map[string]any{
				"id": m.ID, "name": m.Name, "kind": m.Kind, "provider": m.Provider, "price": m.Price,
			})
		}
		return map[string]any{"models": out, "count": len(out)}, nil

	case "get_model":
		var a struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal(cp.Arguments, &a); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		m, err := s.deps.Catalog.Get(a.Model)
		if err != nil {
			return nil, err
		}
		return m, nil

	case "generate":
		return s.toolGenerate(ctx, cp.Arguments)

	case "get_job":
		var a struct {
			JobID string `json:"job_id"`
		}
		if err := json.Unmarshal(cp.Arguments, &a); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		store, err := jobs.Open()
		if err != nil {
			return nil, err
		}
		job, err := store.Get(a.JobID)
		if err != nil {
			return nil, err
		}
		return job, nil

	case "list_jobs":
		var a struct {
			Status string `json:"status"`
			Limit  int    `json:"limit"`
		}
		_ = json.Unmarshal(cp.Arguments, &a)
		if a.Limit == 0 {
			a.Limit = 20
		}
		store, err := jobs.Open()
		if err != nil {
			return nil, err
		}
		return map[string]any{"jobs": store.List(a.Status, a.Limit)}, nil
	}
	return nil, fmt.Errorf("unknown tool %q", cp.Name)
}

func (s *Server) toolGenerate(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Model          string         `json:"model"`
		Params         map[string]any `json:"params"`
		Wait           *bool          `json:"wait"`
		TimeoutSeconds int            `json:"timeout_seconds"`
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	m, err := s.deps.Catalog.Get(a.Model)
	if err != nil {
		return nil, err
	}
	if a.Params == nil {
		a.Params = map[string]any{}
	}
	if err := catalog.Validate(m, a.Params); err != nil {
		return nil, err
	}
	values := catalog.ApplyDefaults(m, a.Params)
	if m.Provider == "runpod" {
		if err := resolveMedia(m, values); err != nil {
			return nil, err
		}
	}
	p, err := s.deps.ProviderFor(m)
	if err != nil {
		return nil, err
	}
	job, err := p.Create(ctx, m, values)
	if err != nil {
		return nil, err
	}
	job.Model = m.ID

	store, err := jobs.Open()
	if err != nil {
		return nil, err
	}
	store.Put(job)
	_ = store.Save()

	wait := a.Wait == nil || *a.Wait
	if wait && !job.Status.Terminal() {
		timeout := time.Duration(a.TimeoutSeconds) * time.Second
		if timeout <= 0 {
			timeout = 10 * time.Minute
		}
		deadline := time.Now().Add(timeout)
		for !job.Status.Terminal() {
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("timed out after %s waiting for job %s", timeout, job.ID)
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(3 * time.Second):
			}
			updated, err := p.Get(ctx, job.Endpoint, job.ID)
			if err != nil {
				return nil, err
			}
			updated.Model, updated.Created = job.Model, job.Created
			job = updated
			store.Put(job)
			_ = store.Save()
		}
	}
	if !job.Status.OK() && job.Status.Terminal() {
		msg := job.Error
		if msg == "" {
			msg = "no error detail returned"
		}
		return nil, fmt.Errorf("job %s ended with status %s: %s", job.ID, job.Status, msg)
	}
	if job.Status.OK() && s.deps.Download != nil {
		for i := range job.Outputs {
			o := &job.Outputs[i]
			if o.LocalPath != "" || o.Text != "" {
				continue
			}
			name := fmt.Sprintf("%s-%d", job.ID, i+1)
			if o.B64 != "" {
				path, err := media.SaveBase64(s.deps.OutputDir, name, o.Ext, o.B64)
				if err != nil {
					return nil, err
				}
				o.LocalPath = path
			} else if o.URL != "" {
				path, err := s.deps.Download(ctx, s.deps.OutputDir, name, o.Ext, o.URL)
				if err != nil {
					return nil, err
				}
				o.LocalPath = path
			}
		}
		store.Put(job)
		_ = store.Save()
	}
	return job, nil
}

func resolveMedia(m catalog.Model, values map[string]any) error {
	for _, p := range m.Params {
		if p.Media == "" {
			continue
		}
		v, ok := values[p.Name]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case string:
			r, err := media.Resolve(t)
			if err != nil {
				return fmt.Errorf("%s: %w", p.Name, err)
			}
			values[p.Name] = r
		case []any:
			out := make([]string, 0, len(t))
			for _, item := range t {
				s, ok := item.(string)
				if !ok {
					continue
				}
				r, err := media.Resolve(s)
				if err != nil {
					return fmt.Errorf("%s: %w", p.Name, err)
				}
				out = append(out, r)
			}
			values[p.Name] = out
		case []string:
			out, err := media.ResolveAll(t)
			if err != nil {
				return fmt.Errorf("%s: %w", p.Name, err)
			}
			values[p.Name] = out
		}
	}
	return nil
}
