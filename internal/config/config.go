// Package config stores non-secret settings and API credentials on disk.
//
// Layout mirrors the split the Higgsfield CLI uses: a world-readable config
// file for preferences, and a 0600 credentials file for secrets.
//
//	$XDG_CONFIG_HOME/bara/config.json       preferences
//	$XDG_CONFIG_HOME/bara/credentials.json  API keys (0600)
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// Provider identifies a generation backend.
type Provider string

const (
	ProviderOpenAI Provider = "openai"
	ProviderRunPod Provider = "runpod"
)

// Config holds non-secret preferences.
type Config struct {
	// OutputDir is where generated media is written.
	OutputDir string `json:"output_dir,omitempty"`
	// DefaultImageModel is used when `generate create` gets no model.
	DefaultImageModel string `json:"default_image_model,omitempty"`
	// OpenAIBaseURL overrides the OpenAI API base (for proxies/gateways).
	OpenAIBaseURL string `json:"openai_base_url,omitempty"`
	// RunPodBaseURL overrides the RunPod public-endpoint base.
	RunPodBaseURL string `json:"runpod_base_url,omitempty"`

	// UploadBackend selects where artifacts are published: none, catbox,
	// litterbox, or s3. Empty means none.
	UploadBackend string `json:"upload_backend,omitempty"`
	// UploadBaseURL is the public base URL objects are served from (s3).
	UploadBaseURL string `json:"upload_base_url,omitempty"`
	// LitterboxTime is the retention window: 1h, 12h, 24h, or 72h.
	LitterboxTime string `json:"litterbox_time,omitempty"`

	// S3-compatible bucket settings (Cloudflare R2, AWS S3, B2, MinIO).
	S3Endpoint  string `json:"s3_endpoint,omitempty"`
	S3Bucket    string `json:"s3_bucket,omitempty"`
	S3Region    string `json:"s3_region,omitempty"`
	S3Prefix    string `json:"s3_prefix,omitempty"`
	S3PathStyle *bool  `json:"s3_path_style,omitempty"`
}

// S3UsePathStyle reports whether to address the bucket as a path segment.
// Path style is the default because R2, B2, and MinIO all require it.
func (c *Config) S3UsePathStyle() bool {
	if c.S3PathStyle == nil {
		return true
	}
	return *c.S3PathStyle
}

// Credentials holds secrets. Persisted with 0600.
type Credentials struct {
	OpenAIAPIKey   string `json:"openai_api_key,omitempty"`
	RunPodAPIKey   string `json:"runpod_api_key,omitempty"`
	S3AccessKey    string `json:"s3_access_key_id,omitempty"`
	S3SecretKey    string `json:"s3_secret_access_key,omitempty"`
	CatboxUserHash string `json:"catbox_userhash,omitempty"`
}

const (
	// DefaultOpenAIBaseURL is the public OpenAI API root.
	DefaultOpenAIBaseURL = "https://api.openai.com/v1"
	// DefaultRunPodBaseURL is the RunPod public-endpoints root.
	DefaultRunPodBaseURL = "https://api.runpod.ai/v2"
)

// Dir returns the directory holding config and credentials.
// BARA_CONFIG_DIR overrides it entirely.
func Dir() (string, error) {
	if d := os.Getenv("BARA_CONFIG_DIR"); d != "" {
		return d, nil
	}
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "bara"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".config", "bara"), nil
}

func configPath() (string, error) {
	if p := os.Getenv("BARA_CONFIG_PATH"); p != "" {
		return p, nil
	}
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

func credentialsPath() (string, error) {
	if p := os.Getenv("BARA_CREDENTIALS_PATH"); p != "" {
		return p, nil
	}
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "credentials.json"), nil
}

func readJSON(path string, dst any) error {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return nil
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func writeJSON(path string, src any, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	b, err := json.MarshalIndent(src, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	// Write to a temp file in the same dir, then rename, so a crash never
	// leaves a half-written credentials file behind.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(perm); err != nil && runtime.GOOS != "windows" {
		tmp.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("install %s: %w", path, err)
	}
	return nil
}

// Load reads preferences, applying defaults and environment overrides.
func Load() (*Config, error) {
	p, err := configPath()
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := readJSON(p, cfg); err != nil {
		return nil, err
	}
	if v := os.Getenv("BARA_OUTPUT_DIR"); v != "" {
		cfg.OutputDir = v
	}
	if v := os.Getenv("BARA_OPENAI_BASE_URL"); v != "" {
		cfg.OpenAIBaseURL = v
	}
	if v := os.Getenv("BARA_RUNPOD_BASE_URL"); v != "" {
		cfg.RunPodBaseURL = v
	}
	if cfg.OpenAIBaseURL == "" {
		cfg.OpenAIBaseURL = DefaultOpenAIBaseURL
	}
	if cfg.RunPodBaseURL == "" {
		cfg.RunPodBaseURL = DefaultRunPodBaseURL
	}
	if cfg.OutputDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("locate home directory: %w", err)
		}
		cfg.OutputDir = filepath.Join(home, "bara-output")
	}
	return cfg, nil
}

// Save persists preferences (0644).
func (c *Config) Save() error {
	p, err := configPath()
	if err != nil {
		return err
	}
	return writeJSON(p, c, 0o644)
}

// LoadCredentials reads secrets. Environment variables win over the file so a
// CI run can inject keys without touching disk.
func LoadCredentials() (*Credentials, error) {
	p, err := credentialsPath()
	if err != nil {
		return nil, err
	}
	cr := &Credentials{}
	if err := readJSON(p, cr); err != nil {
		return nil, err
	}
	if v := firstNonEmpty(os.Getenv("BARA_OPENAI_API_KEY"), os.Getenv("OPENAI_API_KEY")); v != "" {
		cr.OpenAIAPIKey = v
	}
	if v := firstNonEmpty(os.Getenv("BARA_RUNPOD_API_KEY"), os.Getenv("RUNPOD_API_KEY")); v != "" {
		cr.RunPodAPIKey = v
	}
	// AWS_* is honoured so an existing S3 profile works without re-entry.
	if v := firstNonEmpty(os.Getenv("BARA_S3_ACCESS_KEY_ID"), os.Getenv("AWS_ACCESS_KEY_ID")); v != "" {
		cr.S3AccessKey = v
	}
	if v := firstNonEmpty(os.Getenv("BARA_S3_SECRET_ACCESS_KEY"), os.Getenv("AWS_SECRET_ACCESS_KEY")); v != "" {
		cr.S3SecretKey = v
	}
	return cr, nil
}

// Save persists secrets with 0600.
func (c *Credentials) Save() error {
	p, err := credentialsPath()
	if err != nil {
		return err
	}
	return writeJSON(p, c, 0o600)
}

// KeyFor returns the API key for a provider, or an error naming the command
// that would set it.
func (c *Credentials) KeyFor(p Provider) (string, error) {
	switch p {
	case ProviderOpenAI:
		if c.OpenAIAPIKey == "" {
			return "", errors.New("no OpenAI API key configured — run `bara settings set openai_api_key <key>` or export OPENAI_API_KEY")
		}
		return c.OpenAIAPIKey, nil
	case ProviderRunPod:
		if c.RunPodAPIKey == "" {
			return "", errors.New("no RunPod API key configured — run `bara settings set runpod_api_key <key>` or export RUNPOD_API_KEY")
		}
		return c.RunPodAPIKey, nil
	}
	return "", fmt.Errorf("unknown provider %q", p)
}

// Setting is one addressable key in `bara settings`.
type Setting struct {
	Name   string
	Secret bool
	Desc   string
}

// Settings enumerates every key `bara settings set` accepts.
func Settings() []Setting {
	s := []Setting{
		{Name: "openai_api_key", Secret: true, Desc: "OpenAI API key (gpt-image-* models)"},
		{Name: "runpod_api_key", Secret: true, Desc: "RunPod API key (public endpoints)"},
		{Name: "output_dir", Desc: "directory generated media is written to"},
		{Name: "default_image_model", Desc: "model used when none is given"},
		{Name: "openai_base_url", Desc: "override the OpenAI API base URL"},
		{Name: "runpod_base_url", Desc: "override the RunPod API base URL"},
		{Name: "upload_backend", Desc: "publish outputs to a URL: none, catbox, litterbox, s3"},
		{Name: "upload_base_url", Desc: "public base URL objects are served from (s3)"},
		{Name: "litterbox_time", Desc: "litterbox retention: 1h, 12h, 24h, 72h"},
		{Name: "catbox_userhash", Secret: true, Desc: "catbox account hash (optional; uploads are anonymous without it)"},
		{Name: "s3_endpoint", Desc: "S3 API endpoint, e.g. https://<account>.r2.cloudflarestorage.com"},
		{Name: "s3_bucket", Desc: "bucket name"},
		{Name: "s3_region", Desc: "region (use 'auto' for Cloudflare R2)"},
		{Name: "s3_prefix", Desc: "key prefix inside the bucket"},
		{Name: "s3_path_style", Desc: "address the bucket as a path segment (true for R2/B2/MinIO)"},
		{Name: "s3_access_key_id", Secret: true, Desc: "S3 access key id"},
		{Name: "s3_secret_access_key", Secret: true, Desc: "S3 secret access key"},
	}
	sort.Slice(s, func(i, j int) bool { return s[i].Name < s[j].Name })
	return s
}

// Get returns the current value of a setting. Secrets are returned in full;
// callers decide whether to mask.
func Get(cfg *Config, cr *Credentials, name string) (string, error) {
	switch name {
	case "openai_api_key":
		return cr.OpenAIAPIKey, nil
	case "runpod_api_key":
		return cr.RunPodAPIKey, nil
	case "output_dir":
		return cfg.OutputDir, nil
	case "default_image_model":
		return cfg.DefaultImageModel, nil
	case "openai_base_url":
		return cfg.OpenAIBaseURL, nil
	case "runpod_base_url":
		return cfg.RunPodBaseURL, nil
	case "upload_backend":
		return cfg.UploadBackend, nil
	case "upload_base_url":
		return cfg.UploadBaseURL, nil
	case "litterbox_time":
		return cfg.LitterboxTime, nil
	case "catbox_userhash":
		return cr.CatboxUserHash, nil
	case "s3_endpoint":
		return cfg.S3Endpoint, nil
	case "s3_bucket":
		return cfg.S3Bucket, nil
	case "s3_region":
		return cfg.S3Region, nil
	case "s3_prefix":
		return cfg.S3Prefix, nil
	case "s3_path_style":
		if cfg.S3PathStyle == nil {
			return "", nil
		}
		return strconv.FormatBool(*cfg.S3PathStyle), nil
	case "s3_access_key_id":
		return cr.S3AccessKey, nil
	case "s3_secret_access_key":
		return cr.S3SecretKey, nil
	}
	return "", fmt.Errorf("unknown setting %q (run `bara settings list`)", name)
}

// Set writes a setting into the right store. It reports whether the value
// landed in the credentials file, so the caller knows what to persist.
func Set(cfg *Config, cr *Credentials, name, value string) (secret bool, err error) {
	switch name {
	case "openai_api_key":
		cr.OpenAIAPIKey = value
		return true, nil
	case "runpod_api_key":
		cr.RunPodAPIKey = value
		return true, nil
	case "output_dir":
		cfg.OutputDir = value
	case "default_image_model":
		cfg.DefaultImageModel = value
	case "openai_base_url":
		cfg.OpenAIBaseURL = value
	case "runpod_base_url":
		cfg.RunPodBaseURL = value
	case "upload_backend":
		if value != "" && !validBackend(value) {
			return false, fmt.Errorf("upload_backend must be one of: none, catbox, litterbox, s3 (got %q)", value)
		}
		cfg.UploadBackend = value
	case "upload_base_url":
		cfg.UploadBaseURL = value
	case "litterbox_time":
		if value != "" && !validLitterboxTime(value) {
			return false, fmt.Errorf("litterbox_time must be one of: 1h, 12h, 24h, 72h (got %q)", value)
		}
		cfg.LitterboxTime = value
	case "catbox_userhash":
		cr.CatboxUserHash = value
		return true, nil
	case "s3_endpoint":
		cfg.S3Endpoint = value
	case "s3_bucket":
		cfg.S3Bucket = value
	case "s3_region":
		cfg.S3Region = value
	case "s3_prefix":
		cfg.S3Prefix = value
	case "s3_path_style":
		if value == "" {
			cfg.S3PathStyle = nil
			break
		}
		b, err := strconv.ParseBool(value)
		if err != nil {
			return false, fmt.Errorf("s3_path_style expects true or false (got %q)", value)
		}
		cfg.S3PathStyle = &b
	case "s3_access_key_id":
		cr.S3AccessKey = value
		return true, nil
	case "s3_secret_access_key":
		cr.S3SecretKey = value
		return true, nil
	default:
		return false, fmt.Errorf("unknown setting %q (run `bara settings list`)", name)
	}
	return false, nil
}

func validBackend(v string) bool {
	switch v {
	case "none", "catbox", "litterbox", "s3":
		return true
	}
	return false
}

func validLitterboxTime(v string) bool {
	switch v {
	case "1h", "12h", "24h", "72h":
		return true
	}
	return false
}

// Mask renders a secret as a short prefix plus a fixed tail, so users can tell
// two keys apart without the value being copyable from a screenshot.
func Mask(v string) string {
	if v == "" {
		return ""
	}
	if len(v) <= 8 {
		return strings.Repeat("*", len(v))
	}
	return v[:4] + strings.Repeat("*", 8) + v[len(v)-4:]
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
