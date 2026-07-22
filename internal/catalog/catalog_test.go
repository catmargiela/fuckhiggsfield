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
	if len(c.All()) < 30 {
		t.Fatalf("catalog has only %d models; expected the full registry", len(c.All()))
	}
	for _, m := range c.All() {
		if m.ID == "" || m.Name == "" || m.Endpoint == "" {
			t.Errorf("%q: id, name, and endpoint are all required", m.ID)
		}
		switch m.Provider {
		case "openai", "runpod":
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
	for _, m := range c.Filter(KindVideo, "runpod", "") {
		if m.Kind != KindVideo || m.Provider != "runpod" {
			t.Errorf("filter leaked %s (%s/%s)", m.ID, m.Kind, m.Provider)
		}
	}
	if len(c.Filter(KindImage, "openai", "")) == 0 {
		t.Error("expected OpenAI image models")
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

// Endpoint slugs are what the URL is built from, so a typo silently 404s.
func TestKnownEndpointSlugs(t *testing.T) {
	c := load(t)
	want := map[string]string{
		"gpt_image_2":        "gpt-image-2",
		"flux_dev":           "black-forest-labs-flux-1-dev",
		"flux_schnell":       "black-forest-labs-flux-1-schnell",
		"qwen_image":         "qwen-image-t2i",
		"seedream_4_t2i":     "seedream-v4-t2i",
		"wan_2_2_i2v":        "wan-2-2-i2v-720",
		"sora_2_pro_i2v":     "sora-2-pro-i2v",
		"granite_4":          "granite-4-0-h-small",
		"z_image_turbo":      "z-image-turbo",
		"infinitetalk":       "infinitetalk",
		"nano_banana_2_edit": "google-nano-banana-2-edit",
	}
	for id, endpoint := range want {
		m, err := c.Get(id)
		if err != nil {
			t.Errorf("missing model %s: %v", id, err)
			continue
		}
		if m.Endpoint != endpoint {
			t.Errorf("%s endpoint = %q, want %q", id, m.Endpoint, endpoint)
		}
	}
}
