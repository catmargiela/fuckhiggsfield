package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("BARA_CONFIG_DIR", dir)
	t.Setenv("BARA_CONFIG_PATH", "")
	t.Setenv("BARA_CREDENTIALS_PATH", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("BARA_OPENAI_API_KEY", "")
	return dir
}

func TestCredentialsRoundTrip(t *testing.T) {
	isolate(t)
	cr := &Credentials{OpenAIAPIKey: "sk-abc", CatboxUserHash: "hash"}
	if err := cr.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if got.OpenAIAPIKey != "sk-abc" || got.CatboxUserHash != "hash" {
		t.Fatalf("round trip lost keys: %+v", got)
	}
}

// Secrets on a shared machine must not be world-readable.
func TestCredentialsFileIsPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions do not apply on Windows")
	}
	dir := isolate(t)
	cr := &Credentials{OpenAIAPIKey: "sk-secret"}
	if err := cr.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "credentials.json"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("credentials mode = %o, want 600", perm)
	}
}

func TestEnvironmentOverridesFile(t *testing.T) {
	isolate(t)
	cr := &Credentials{OpenAIAPIKey: "from-file"}
	if err := cr.Save(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENAI_API_KEY", "from-env")
	got, err := LoadCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if got.OpenAIAPIKey != "from-env" {
		t.Fatalf("env should win, got %q", got.OpenAIAPIKey)
	}
}

func TestBaraPrefixedEnvWinsOverGeneric(t *testing.T) {
	isolate(t)
	t.Setenv("OPENAI_API_KEY", "generic")
	t.Setenv("BARA_OPENAI_API_KEY", "specific")
	got, err := LoadCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if got.OpenAIAPIKey != "specific" {
		t.Fatalf("BARA_OPENAI_API_KEY should win, got %q", got.OpenAIAPIKey)
	}
}

func TestConfigDefaults(t *testing.T) {
	isolate(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OpenAIBaseURL != DefaultOpenAIBaseURL {
		t.Errorf("openai base = %q", cfg.OpenAIBaseURL)
	}
	if got := cfg.EffectiveUploadBackend(); got != "catbox" {
		t.Errorf("upload backend = %q, want catbox", got)
	}
	if cfg.OutputDir == "" {
		t.Error("output dir should default to something")
	}
}

// Opting out with none must survive a reload instead of snapping back to the
// default backend.
func TestExplicitNoneUploadBackendIsKept(t *testing.T) {
	isolate(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Set(cfg, &Credentials{}, "upload_backend", "none"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.EffectiveUploadBackend() != "none" {
		t.Fatalf("upload backend = %q, want none", got.EffectiveUploadBackend())
	}
}

// Saving any setting must not freeze the default backend into config.json,
// or unsetting upload_backend later would no longer restore the default.
func TestSaveDoesNotPersistDefaultUploadBackend(t *testing.T) {
	dir := isolate(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Set(cfg, &Credentials{}, "default_image_model", "gpt_image_2"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "upload_backend") {
		t.Fatalf("config.json persisted the default backend: %s", b)
	}
}

func TestSetRoutesSecretsAndPreferences(t *testing.T) {
	isolate(t)
	cfg, _ := Load()
	cr, _ := LoadCredentials()

	secret, err := Set(cfg, cr, "openai_api_key", "sk-1")
	if err != nil || !secret {
		t.Fatalf("openai_api_key should be a secret: secret=%v err=%v", secret, err)
	}
	secret, err = Set(cfg, cr, "output_dir", "/tmp/out")
	if err != nil || secret {
		t.Fatalf("output_dir should not be a secret: secret=%v err=%v", secret, err)
	}
	if cfg.OutputDir != "/tmp/out" || cr.OpenAIAPIKey != "sk-1" {
		t.Fatalf("values landed in the wrong store: cfg=%+v cr=%+v", cfg, cr)
	}
	if _, err := Set(cfg, cr, "nonsense", "x"); err == nil {
		t.Error("expected an error for an unknown setting")
	}
}

func TestKeyForReportsHowToFix(t *testing.T) {
	cr := &Credentials{}
	_, err := cr.KeyFor(ProviderOpenAI)
	if err == nil {
		t.Fatal("expected an error when no key is configured")
	}
	if !strings.Contains(err.Error(), "settings set openai_api_key") {
		t.Errorf("error should name the fix, got %q", err.Error())
	}
	if _, err := cr.KeyFor(Provider("runpod")); err == nil {
		t.Error("expected an error for a provider that no longer exists")
	}
}

func TestMaskHidesTheMiddle(t *testing.T) {
	got := Mask("sk-proj-1234567890abcdef")
	if strings.Contains(got, "1234567890") {
		t.Fatalf("mask leaked the key body: %q", got)
	}
	if !strings.HasPrefix(got, "sk-p") || !strings.HasSuffix(got, "cdef") {
		t.Errorf("mask should keep short head/tail markers, got %q", got)
	}
	if Mask("") != "" {
		t.Error("empty stays empty")
	}
	// A key too short to keep head/tail markers must be fully hidden.
	if got := Mask("short"); got != "*****" {
		t.Errorf("Mask(\"short\") = %q, want it fully masked", got)
	}
}

func TestGetUnknownSetting(t *testing.T) {
	isolate(t)
	cfg, _ := Load()
	cr, _ := LoadCredentials()
	if _, err := Get(cfg, cr, "bogus"); err == nil {
		t.Error("expected an error")
	}
}

func TestSettingsListIsComplete(t *testing.T) {
	isolate(t)
	cfg, _ := Load()
	cr, _ := LoadCredentials()
	for _, s := range Settings() {
		if s.Desc == "" {
			t.Errorf("setting %s has no description", s.Name)
		}
		if _, err := Get(cfg, cr, s.Name); err != nil {
			t.Errorf("setting %s is listed but not gettable: %v", s.Name, err)
		}
		if _, err := Set(cfg, cr, s.Name, ""); err != nil {
			t.Errorf("setting %s is listed but not settable: %v", s.Name, err)
		}
	}
}
