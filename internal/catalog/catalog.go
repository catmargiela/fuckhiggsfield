// Package catalog holds the model registry: which models exist, which backend
// serves them, and what parameters each accepts.
//
// The registry is data, not code — see models.json. Adding a model is a JSON
// edit, which is what lets `bara model get` print a real schema and lets
// `generate create` validate flags before spending a credit.
package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed models.json
var modelsJSON []byte

// Kind is the media class a model produces.
type Kind string

const (
	KindImage Kind = "image"
	KindVideo Kind = "video"
	KindAudio Kind = "audio"
	KindText  Kind = "text"
)

// ParamType is the wire type of a parameter value.
type ParamType string

const (
	TypeString ParamType = "string"
	TypeInt    ParamType = "int"
	TypeFloat  ParamType = "float"
	TypeBool   ParamType = "bool"
	// TypeStringArray is a repeatable/comma-separated list of strings.
	TypeStringArray ParamType = "string[]"
	// TypeJSON is raw JSON, used for structured params such as LoRA lists.
	TypeJSON ParamType = "json"
)

// Media marks a parameter that accepts a media reference. The CLI resolves
// local paths for these before sending the request.
type Media string

const (
	MediaNone  Media = ""
	MediaImage Media = "image"
	MediaVideo Media = "video"
	MediaAudio Media = "audio"
)

// Param describes one model parameter.
type Param struct {
	Name     string    `json:"name"`
	Type     ParamType `json:"type"`
	Required bool      `json:"required,omitempty"`
	Default  any       `json:"default,omitempty"`
	Enum     []string  `json:"enum,omitempty"`
	Min      *float64  `json:"min,omitempty"`
	Max      *float64  `json:"max,omitempty"`
	Media    Media     `json:"media,omitempty"`
	Desc     string    `json:"desc,omitempty"`
}

// Model is one entry in the registry.
type Model struct {
	// ID is the name users type, e.g. "gpt_image_2".
	ID string `json:"id"`
	// Name is the human label.
	Name string `json:"name"`
	// Provider routes the request: "openai".
	Provider string `json:"provider"`
	// Kind is the output media class.
	Kind Kind `json:"kind"`
	// Endpoint is the provider-side identifier: the OpenAI model id.
	Endpoint string `json:"endpoint"`
	// Price is a human-readable pricing note straight from the provider docs.
	Price string `json:"price,omitempty"`
	// Params are the accepted inputs.
	Params []Param `json:"params"`
	// Notes is free-form guidance shown by `bara model get`.
	Notes string `json:"notes,omitempty"`
}

// Param returns the named parameter.
func (m Model) Param(name string) (Param, bool) {
	for _, p := range m.Params {
		if p.Name == name {
			return p, true
		}
	}
	return Param{}, false
}

// Catalog is the loaded registry.
type Catalog struct {
	models []Model
	byID   map[string]Model
}

type registryFile struct {
	Models []Model `json:"models"`
}

var loaded *Catalog

// Load returns the embedded registry. It is parsed once and cached.
func Load() (*Catalog, error) {
	if loaded != nil {
		return loaded, nil
	}
	var rf registryFile
	if err := json.Unmarshal(modelsJSON, &rf); err != nil {
		return nil, fmt.Errorf("parse embedded model registry: %w", err)
	}
	c := &Catalog{models: rf.Models, byID: make(map[string]Model, len(rf.Models))}
	for _, m := range rf.Models {
		if _, dup := c.byID[m.ID]; dup {
			return nil, fmt.Errorf("duplicate model id %q in registry", m.ID)
		}
		c.byID[m.ID] = m
	}
	sort.Slice(c.models, func(i, j int) bool { return c.models[i].ID < c.models[j].ID })
	loaded = c
	return c, nil
}

// All returns every model, sorted by id.
func (c *Catalog) All() []Model { return append([]Model(nil), c.models...) }

// Get looks up a model by id.
func (c *Catalog) Get(id string) (Model, error) {
	m, ok := c.byID[id]
	if !ok {
		return Model{}, fmt.Errorf("unknown model %q — run `bara model list` for the catalog%s", id, c.suggest(id))
	}
	return m, nil
}

// suggest offers a near-miss hint for a mistyped model id.
func (c *Catalog) suggest(id string) string {
	norm := func(s string) string {
		return strings.NewReplacer("-", "", "_", "", ".", "", " ", "").Replace(strings.ToLower(s))
	}
	want := norm(id)
	var hits []string
	for _, m := range c.models {
		got := norm(m.ID)
		if got == want || strings.Contains(got, want) || strings.Contains(want, got) {
			hits = append(hits, m.ID)
		}
	}
	if len(hits) == 0 {
		return ""
	}
	if len(hits) > 5 {
		hits = hits[:5]
	}
	return "\n\nDid you mean: " + strings.Join(hits, ", ")
}

// Filter narrows the catalog. Empty values mean "any".
func (c *Catalog) Filter(kind Kind, provider, query string) []Model {
	var out []Model
	q := strings.ToLower(query)
	for _, m := range c.models {
		if kind != "" && m.Kind != kind {
			continue
		}
		if provider != "" && m.Provider != provider {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(m.ID), q) && !strings.Contains(strings.ToLower(m.Name), q) {
			continue
		}
		out = append(out, m)
	}
	return out
}

// Providers lists the distinct providers present in the registry.
func (c *Catalog) Providers() []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range c.models {
		if !seen[m.Provider] {
			seen[m.Provider] = true
			out = append(out, m.Provider)
		}
	}
	sort.Strings(out)
	return out
}

// Coerce converts a raw command-line string into the parameter's wire type and
// enforces enum/range constraints.
func Coerce(p Param, raw string) (any, error) {
	switch p.Type {
	case TypeString:
		if err := checkEnum(p, raw); err != nil {
			return nil, err
		}
		return raw, nil
	case TypeInt:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("--%s expects an integer, got %q", p.Name, raw)
		}
		if err := checkEnum(p, raw); err != nil {
			return nil, err
		}
		if err := checkRange(p, float64(n)); err != nil {
			return nil, err
		}
		return n, nil
	case TypeFloat:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("--%s expects a number, got %q", p.Name, raw)
		}
		if err := checkRange(p, f); err != nil {
			return nil, err
		}
		return f, nil
	case TypeBool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("--%s expects true or false, got %q", p.Name, raw)
		}
		return b, nil
	case TypeStringArray:
		parts := splitList(raw)
		if err := checkEnumList(p, parts); err != nil {
			return nil, err
		}
		return parts, nil
	case TypeJSON:
		var v any
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return nil, fmt.Errorf("--%s expects JSON, got %q: %w", p.Name, raw, err)
		}
		return v, nil
	}
	return nil, fmt.Errorf("parameter %q has unsupported type %q", p.Name, p.Type)
}

func splitList(raw string) []string {
	var out []string
	for _, s := range strings.Split(raw, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func checkEnum(p Param, v string) error {
	if len(p.Enum) == 0 {
		return nil
	}
	for _, e := range p.Enum {
		if e == v {
			return nil
		}
	}
	return fmt.Errorf("--%s must be one of: %s (got %q)", p.Name, strings.Join(p.Enum, ", "), v)
}

func checkEnumList(p Param, vs []string) error {
	for _, v := range vs {
		if err := checkEnum(p, v); err != nil {
			return err
		}
	}
	return nil
}

func checkRange(p Param, v float64) error {
	if p.Min != nil && v < *p.Min {
		return fmt.Errorf("--%s must be >= %v (got %v)", p.Name, *p.Min, v)
	}
	if p.Max != nil && v > *p.Max {
		return fmt.Errorf("--%s must be <= %v (got %v)", p.Name, *p.Max, v)
	}
	return nil
}

// Validate reports missing required parameters and unknown keys.
func Validate(m Model, values map[string]any) error {
	known := map[string]bool{}
	for _, p := range m.Params {
		known[p.Name] = true
	}
	var unknown []string
	for k := range values {
		if !known[k] {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf("unknown parameter(s) for %s: %s\n\nRun `bara model get %s` to see accepted parameters", m.ID, strings.Join(unknown, ", "), m.ID)
	}
	var missing []string
	for _, p := range m.Params {
		if !p.Required {
			continue
		}
		if _, ok := values[p.Name]; !ok {
			missing = append(missing, "--"+p.Name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("missing required parameter(s) for %s: %s", m.ID, strings.Join(missing, " "))
	}
	return nil
}

// ApplyDefaults fills unset parameters that declare a default.
func ApplyDefaults(m Model, values map[string]any) map[string]any {
	out := make(map[string]any, len(values)+len(m.Params))
	for k, v := range values {
		out[k] = v
	}
	for _, p := range m.Params {
		if p.Default == nil {
			continue
		}
		if _, ok := out[p.Name]; !ok {
			out[p.Name] = p.Default
		}
	}
	return out
}
