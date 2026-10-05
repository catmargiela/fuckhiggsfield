package catalog

import (
	"strings"
	"testing"
)

func load(t *testing.T) *Catalog {
	t.Helper()
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return c
}

// The registry is the CLI's contract; a malformed entry would only surface at
// generation time, after the user has already typed a prompt.
func TestRegistryIsWellFormed(t *testing.T) {
	c := load(t)
	if len(c.All()) == 0 {
		t.Fatal("catalog is empty")
	}
	for _, m := range c.All() {
		if m.ID == "" || m.Name == "" || m.Endpoint == "" {
			t.Errorf("%q: id, name, and endpoint are all required", m.ID)
		}
		switch m.Provider {
		case "openai":
		default:
			t.Errorf("%s: unknown provider %q", m.ID, m.Provider)
		}
		switch m.Kind {
		case KindImage, KindVideo, KindAudio, KindText:
		default:
			t.Errorf("%s: unknown kind %q", m.ID, m.Kind)
		}
		if len(m.Params) == 0 {
			t.Errorf("%s: no parameters declared", m.ID)
		}
		seen := map[string]bool{}
		for _, p := range m.Params {
			if seen[p.Name] {
				t.Errorf("%s: duplicate parameter %q", m.ID, p.Name)
			}
			seen[p.Name] = true
			switch p.Type {
			case TypeString, TypeInt, TypeFloat, TypeBool, TypeStringArray, TypeJSON:
			default:
				t.Errorf("%s.%s: unknown type %q", m.ID, p.Name, p.Type)
			}
			if p.Media != "" && p.Media != MediaImage && p.Media != MediaVideo && p.Media != MediaAudio {
				t.Errorf("%s.%s: unknown media kind %q", m.ID, p.Name, p.Media)
			}
		}
		// Every generative model needs some way to say what to make.
		if m.Kind != KindText {
			if _, ok := m.Param("prompt"); !ok {
				t.Errorf("%s: no prompt parameter", m.ID)
			}
		}
	}
}

func TestDefaultsSatisfyTheirOwnConstraints(t *testing.T) {
	for _, m := range load(t).All() {
		for _, p := range m.Params {
			if p.Default == nil || len(p.Enum) == 0 {
				continue
			}
			var s string
			switch v := p.Default.(type) {
			case string:
				s = v
			case float64:
				if v == float64(int64(v)) {
					s = trimInt(int64(v))
				}
			}
			if s == "" {
				continue
			}
			found := false
			for _, e := range p.Enum {
				if e == s {
					found = true
				}
			}
			if !found {
				t.Errorf("%s.%s: default %q is not in enum %v", m.ID, p.Name, s, p.Enum)
			}
		}
	}
}

func trimInt(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

func TestGetUnknownModelSuggests(t *testing.T) {
	c := load(t)
	_, err := c.Get("gpt-image-2")
	if err == nil {
		t.Fatal("expected an error for a mistyped id")
	}
	if !strings.Contains(err.Error(), "gpt_image_2") {
		t.Errorf("error should suggest the real id, got: %v", err)
	}
}

func TestFilterByKindAndProvider(t *testing.T) {
	c := load(t)
	if len(c.Filter(KindImage, "openai", "")) == 0 {
		t.Error("expected OpenAI image models")
	}
	if got := c.Filter(KindVideo, "", ""); len(got) != 0 {
		t.Errorf("kind filter leaked %d model(s)", len(got))
	}
	if got := c.Filter("", "runpod", ""); len(got) != 0 {
		t.Errorf("provider filter leaked %d model(s)", len(got))
	}
}

func TestCoerceTypes(t *testing.T) {
	if v, err := Coerce(Param{Name: "n", Type: TypeInt}, "5"); err != nil || v != int64(5) {
		t.Errorf("int coerce = %v, %v", v, err)
	}
	if v, err := Coerce(Param{Name: "g", Type: TypeFloat}, "2.5"); err != nil || v != 2.5 {
		t.Errorf("float coerce = %v, %v", v, err)
	}
	if v, err := Coerce(Param{Name: "b", Type: TypeBool}, "true"); err != nil || v != true {
		t.Errorf("bool coerce = %v, %v", v, err)
	}
	if _, err := Coerce(Param{Name: "n", Type: TypeInt}, "abc"); err == nil {
		t.Error("expected an error for a non-integer")
	}
}

func TestCoerceJSON(t *testing.T) {
	v, err := Coerce(Param{Name: "loras", Type: TypeJSON}, `[{"path":"u","scale":0.5}]`)
	if err != nil {
		t.Fatalf("json coerce: %v", err)
	}
	arr, ok := v.([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("parsed = %#v", v)
	}
	if _, err := Coerce(Param{Name: "loras", Type: TypeJSON}, "{not json"); err == nil {
		t.Error("expected an error for malformed JSON")
	}
}

func TestValidateRequiredAndUnknown(t *testing.T) {
	m := Model{ID: "m", Params: []Param{{Name: "prompt", Type: TypeString, Required: true}}}
	if err := Validate(m, map[string]any{}); err == nil {
		t.Error("expected an error for a missing required parameter")
	}
	if err := Validate(m, map[string]any{"prompt": "x", "bogus": 1}); err == nil {
		t.Error("expected an error for an unknown parameter")
	}
	if err := Validate(m, map[string]any{"prompt": "x"}); err != nil {
		t.Errorf("valid input rejected: %v", err)
	}
}

func TestApplyDefaultsDoesNotOverrideUserValues(t *testing.T) {
	m := Model{Params: []Param{
		{Name: "size", Type: TypeString, Default: "1024*1024"},
		{Name: "seed", Type: TypeInt, Default: float64(-1)},
	}}
	got := ApplyDefaults(m, map[string]any{"size": "512*512"})
	if got["size"] != "512*512" {
		t.Errorf("user value overwritten: %v", got["size"])
	}
	if got["seed"] != float64(-1) {
		t.Errorf("default not applied: %v", got["seed"])
	}
}

// The endpoint is the model id sent to OpenAI, so a typo silently 404s.
func TestKnownEndpointSlugs(t *testing.T) {
	m, err := load(t).Get("gpt_image_2")
	if err != nil {
		t.Fatalf("missing model gpt_image_2: %v", err)
	}
	if m.Endpoint != "gpt-image-2" {
		t.Errorf("gpt_image_2 endpoint = %q, want gpt-image-2", m.Endpoint)
	}
}
