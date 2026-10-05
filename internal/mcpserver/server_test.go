package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/catmargiela/fuckhiggsfield/internal/catalog"
	"github.com/catmargiela/fuckhiggsfield/internal/provider"
)

func newTestServer(t *testing.T) (*Server, *bytes.Buffer) {
	t.Helper()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	var out bytes.Buffer
	s := New(Deps{
		Catalog: cat,
		ProviderFor: func(catalog.Model) (provider.Provider, error) {
			t.Fatal("no provider expected in this test")
			return nil, nil
		},
		ServerVersion: "test",
	}, &out)
	return s, &out
}

// roundTrip feeds newline-delimited requests and returns the decoded responses.
func roundTrip(t *testing.T, s *Server, out *bytes.Buffer, lines ...string) []map[string]any {
	t.Helper()
	out.Reset()
	if err := s.Serve(context.Background(), strings.NewReader(strings.Join(lines, "\n")+"\n")); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	var got []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("decode response %q: %v", l, err)
		}
		got = append(got, m)
	}
	return got
}

func TestInitializeEchoesProtocolVersion(t *testing.T) {
	s, out := newTestServer(t)
	got := roundTrip(t, s, out, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
	if len(got) != 1 {
		t.Fatalf("got %d responses, want 1", len(got))
	}
	res, ok := got[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result: %#v", got[0])
	}
	if res["protocolVersion"] != "2025-06-18" {
		t.Errorf("protocolVersion = %v", res["protocolVersion"])
	}
	info, _ := res["serverInfo"].(map[string]any)
	if info["name"] != "bara" {
		t.Errorf("serverInfo.name = %v", info["name"])
	}
	if _, ok := res["capabilities"].(map[string]any)["tools"]; !ok {
		t.Error("server must advertise the tools capability")
	}
}

// Notifications carry no id and must produce no response at all.
func TestNotificationProducesNoResponse(t *testing.T) {
	s, out := newTestServer(t)
	got := roundTrip(t, s, out, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if len(got) != 0 {
		t.Fatalf("notification produced %d responses: %#v", len(got), got)
	}
}

func TestToolsListAdvertisesSchemas(t *testing.T) {
	s, out := newTestServer(t)
	got := roundTrip(t, s, out, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	res := got[0]["result"].(map[string]any)
	tools, _ := res["tools"].([]any)
	if len(tools) == 0 {
		t.Fatal("no tools advertised")
	}
	names := map[string]bool{}
	for _, tl := range tools {
		m := tl.(map[string]any)
		name, _ := m["name"].(string)
		names[name] = true
		if m["description"] == "" {
			t.Errorf("tool %s has no description", name)
		}
		schema, ok := m["inputSchema"].(map[string]any)
		if !ok || schema["type"] != "object" {
			t.Errorf("tool %s has no object inputSchema", name)
		}
	}
	for _, want := range []string{"list_models", "get_model", "generate", "get_job", "list_jobs"} {
		if !names[want] {
			t.Errorf("missing tool %q", want)
		}
	}
}

func TestCallListModels(t *testing.T) {
	s, out := newTestServer(t)
	got := roundTrip(t, s, out,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_models","arguments":{"query":"gpt"}}}`)
	res := got[0]["result"].(map[string]any)
	if res["isError"] == true {
		t.Fatalf("tool call failed: %#v", res)
	}
	structured := res["structuredContent"].(map[string]any)
	models := structured["models"].([]any)
	if len(models) == 0 {
		t.Fatal("expected OpenAI image models")
	}
	for _, m := range models {
		if m.(map[string]any)["provider"] != "openai" {
			t.Errorf("filter leaked: %#v", m)
		}
	}
}

func TestCallGetModel(t *testing.T) {
	s, out := newTestServer(t)
	got := roundTrip(t, s, out,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"get_model","arguments":{"model":"gpt_image_2"}}}`)
	res := got[0]["result"].(map[string]any)
	if res["isError"] == true {
		t.Fatalf("tool call failed: %#v", res)
	}
	sc := res["structuredContent"].(map[string]any)
	if sc["id"] != "gpt_image_2" {
		t.Errorf("id = %v", sc["id"])
	}
	if len(sc["params"].([]any)) == 0 {
		t.Error("model schema has no params")
	}
}

// Tool-level failures are reported in-band so the model can recover, not as
// JSON-RPC protocol errors.
func TestToolErrorIsInBand(t *testing.T) {
	s, out := newTestServer(t)
	got := roundTrip(t, s, out,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"get_model","arguments":{"model":"does_not_exist"}}}`)
	if _, isProtocolError := got[0]["error"]; isProtocolError {
		t.Fatalf("expected an in-band tool error, got a protocol error: %#v", got[0])
	}
	res := got[0]["result"].(map[string]any)
	if res["isError"] != true {
		t.Fatalf("isError not set: %#v", res)
	}
}

func TestUnknownMethod(t *testing.T) {
	s, out := newTestServer(t)
	got := roundTrip(t, s, out, `{"jsonrpc":"2.0","id":6,"method":"does/not/exist"}`)
	e, ok := got[0]["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected a protocol error: %#v", got[0])
	}
	if e["code"].(float64) != codeMethodNotFound {
		t.Errorf("code = %v, want %d", e["code"], codeMethodNotFound)
	}
}

func TestMalformedJSONIsReported(t *testing.T) {
	s, out := newTestServer(t)
	got := roundTrip(t, s, out, `{not json`)
	e, ok := got[0]["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected a parse error: %#v", got[0])
	}
	if e["code"].(float64) != codeParse {
		t.Errorf("code = %v, want %d", e["code"], codeParse)
	}
}

func TestGenerateRejectsUnknownParam(t *testing.T) {
	s, out := newTestServer(t)
	got := roundTrip(t, s, out,
		`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"generate","arguments":{"model":"flux_dev","params":{"bogus":1}}}}`)
	res := got[0]["result"].(map[string]any)
	if res["isError"] != true {
		t.Fatalf("expected validation to reject an unknown parameter: %#v", res)
	}
}

func TestPing(t *testing.T) {
	s, out := newTestServer(t)
	got := roundTrip(t, s, out, `{"jsonrpc":"2.0","id":8,"method":"ping"}`)
	if len(got) != 1 || got[0]["error"] != nil {
		t.Fatalf("ping failed: %#v", got)
	}
}
