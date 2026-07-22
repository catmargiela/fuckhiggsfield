package cmd

import (
	"reflect"
	"testing"
	"time"

	"github.com/catmargiela/fuckhiggsfield/internal/catalog"
)

func testModel(t *testing.T, id string) catalog.Model {
	t.Helper()
	c, err := catalog.Load()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	m, err := c.Get(id)
	if err != nil {
		t.Fatalf("get model %s: %v", id, err)
	}
	return m
}

func TestParseParamsScalars(t *testing.T) {
	m := testModel(t, "flux_dev")
	opts := createOpts{}
	got, err := parseParams(m, []string{
		"--prompt", "a quiet beach at sunrise",
		"--width", "1280",
		"--height", "720",
		"--guidance", "3.5",
	}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	want := map[string]any{
		"prompt":   "a quiet beach at sunrise",
		"width":    int64(1280),
		"height":   int64(720),
		"guidance": 3.5,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parsed = %#v, want %#v", got, want)
	}
}

// The value-consuming closure advances the loop index; if that ever stopped
// working, every second token would be misread as a flag.
func TestParseParamsDoesNotMisreadValuesAsFlags(t *testing.T) {
	m := testModel(t, "flux_dev")
	opts := createOpts{}
	got, err := parseParams(m, []string{"--prompt", "cat", "--seed", "42"}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	if got["prompt"] != "cat" {
		t.Errorf("prompt = %v, want cat", got["prompt"])
	}
	if got["seed"] != int64(42) {
		t.Errorf("seed = %v, want 42", got["seed"])
	}
	if len(got) != 2 {
		t.Errorf("parsed %d params, want 2: %#v", len(got), got)
	}
}

func TestParseParamsInlineEquals(t *testing.T) {
	m := testModel(t, "flux_dev")
	opts := createOpts{}
	got, err := parseParams(m, []string{"--prompt=hello world", "--width=512"}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	if got["prompt"] != "hello world" {
		t.Errorf("prompt = %v", got["prompt"])
	}
	if got["width"] != int64(512) {
		t.Errorf("width = %v", got["width"])
	}
}

func TestParseParamsDashesNormalizeToUnderscores(t *testing.T) {
	m := testModel(t, "flux_dev")
	opts := createOpts{}
	got, err := parseParams(m, []string{"--prompt", "x", "--negative-prompt", "blurry"}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	if got["negative_prompt"] != "blurry" {
		t.Errorf("negative_prompt = %v", got["negative_prompt"])
	}
}

func TestParseParamsBoolBare(t *testing.T) {
	m := testModel(t, "qwen_image")
	opts := createOpts{}
	got, err := parseParams(m, []string{"--prompt", "x", "--enable_safety_checker"}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	if got["enable_safety_checker"] != true {
		t.Errorf("bare bool = %v, want true", got["enable_safety_checker"])
	}
}

func TestParseParamsBoolExplicit(t *testing.T) {
	m := testModel(t, "qwen_image")
	opts := createOpts{}
	got, err := parseParams(m, []string{"--prompt", "x", "--enable_safety_checker", "false"}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	if got["enable_safety_checker"] != false {
		t.Errorf("explicit bool = %v, want false", got["enable_safety_checker"])
	}
}

func TestParseParamsRepeatedListAccumulates(t *testing.T) {
	m := testModel(t, "nano_banana_edit")
	opts := createOpts{}
	got, err := parseParams(m, []string{
		"--prompt", "combine",
		"--images", "a.png",
		"--images", "b.png",
	}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	want := []string{"a.png", "b.png"}
	if !reflect.DeepEqual(got["images"], want) {
		t.Fatalf("images = %#v, want %#v", got["images"], want)
	}
}

func TestParseParamsCommaSeparatedList(t *testing.T) {
	m := testModel(t, "nano_banana_edit")
	opts := createOpts{}
	got, err := parseParams(m, []string{"--prompt", "x", "--images", "a.png,b.png"}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	want := []string{"a.png", "b.png"}
	if !reflect.DeepEqual(got["images"], want) {
		t.Fatalf("images = %#v, want %#v", got["images"], want)
	}
}

// A data URI contains commas but is a single value.
func TestParseParamsDataURIStaysWhole(t *testing.T) {
	m := testModel(t, "nano_banana_edit")
	opts := createOpts{}
	uri := "data:image/png;base64,iVBORw0KGgo,AAAA"
	got, err := parseParams(m, []string{"--prompt", "x", "--images", uri}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	want := []string{uri}
	if !reflect.DeepEqual(got["images"], want) {
		t.Fatalf("images = %#v, want %#v", got["images"], want)
	}
}

func TestParseParamsControlFlags(t *testing.T) {
	m := testModel(t, "flux_dev")
	opts := createOpts{timeout: 10 * time.Minute, interval: 3 * time.Second}
	got, err := parseParams(m, []string{
		"--prompt", "x",
		"--wait",
		"--wait-timeout", "20m",
		"--wait-interval", "5s",
		"--out", "/tmp/renders",
		"--quiet",
		"--json",
	}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	if !opts.wait {
		t.Error("wait not set")
	}
	if opts.timeout != 20*time.Minute {
		t.Errorf("timeout = %v, want 20m", opts.timeout)
	}
	if opts.interval != 5*time.Second {
		t.Errorf("interval = %v, want 5s", opts.interval)
	}
	if opts.outDir != "/tmp/renders" {
		t.Errorf("outDir = %v", opts.outDir)
	}
	if !opts.quiet || !opts.json {
		t.Errorf("quiet=%v json=%v, want both true", opts.quiet, opts.json)
	}
	// Control flags must not leak into the model parameters.
	if len(got) != 1 || got["prompt"] != "x" {
		t.Fatalf("params = %#v, want only prompt", got)
	}
}

// --wait immediately before another flag must not swallow it.
func TestParseParamsBareWaitDoesNotEatNextFlag(t *testing.T) {
	m := testModel(t, "flux_dev")
	opts := createOpts{}
	got, err := parseParams(m, []string{"--wait", "--prompt", "x"}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	if !opts.wait {
		t.Error("wait not set")
	}
	if got["prompt"] != "x" {
		t.Errorf("prompt = %v, want x", got["prompt"])
	}
}

func TestParseParamsUnknownParam(t *testing.T) {
	m := testModel(t, "flux_dev")
	opts := createOpts{}
	_, err := parseParams(m, []string{"--nope", "1"}, &opts)
	if err == nil {
		t.Fatal("expected an error for an unknown parameter")
	}
}

func TestParseParamsEnumRejected(t *testing.T) {
	m := testModel(t, "flux_dev")
	opts := createOpts{}
	_, err := parseParams(m, []string{"--prompt", "x", "--image_format", "tiff"}, &opts)
	if err == nil {
		t.Fatal("expected an error for a value outside the enum")
	}
}

func TestParseParamsRangeRejected(t *testing.T) {
	m := testModel(t, "flux_dev")
	opts := createOpts{}
	_, err := parseParams(m, []string{"--prompt", "x", "--num_inference_steps", "99"}, &opts)
	if err == nil {
		t.Fatal("expected an error for a value above max")
	}
}

func TestParseParamsMissingValue(t *testing.T) {
	m := testModel(t, "flux_dev")
	opts := createOpts{}
	_, err := parseParams(m, []string{"--prompt"}, &opts)
	if err == nil {
		t.Fatal("expected an error when a value-taking flag has no value")
	}
}

func TestEstimateCostPerImage(t *testing.T) {
	m := testModel(t, "qwen_image")
	usd, basis, ok := estimateCost(m, map[string]any{"prompt": "x"})
	if !ok {
		t.Fatal("expected an estimate for a flat per-image rate")
	}
	if usd != 0.02 {
		t.Errorf("usd = %v, want 0.02", usd)
	}
	if basis == "" {
		t.Error("basis should explain the computation")
	}
}

func TestEstimateCostPerSecond(t *testing.T) {
	m := testModel(t, "vidu_q3_t2v")
	usd, _, ok := estimateCost(m, map[string]any{"prompt": "x", "duration": int64(10)})
	if !ok {
		t.Fatal("expected an estimate for a per-second rate with a duration")
	}
	if usd != 1.5 {
		t.Errorf("usd = %v, want 1.5", usd)
	}
}

func TestEstimateCostPerMegapixel(t *testing.T) {
	m := testModel(t, "flux_dev")
	usd, _, ok := estimateCost(m, map[string]any{"width": int64(1000), "height": int64(1000)})
	if !ok {
		t.Fatal("expected an estimate for a per-megapixel rate")
	}
	if usd != 0.02 {
		t.Errorf("usd = %v, want 0.02", usd)
	}
}

// Tiered prices ("$0.14 - $0.24") must not be collapsed into one number.
func TestEstimateCostRefusesTieredPricing(t *testing.T) {
	m := testModel(t, "nano_banana_pro_edit")
	if _, _, ok := estimateCost(m, map[string]any{"prompt": "x"}); ok {
		t.Fatal("tiered pricing should not produce a single estimate")
	}
}

func TestSanitizeJobID(t *testing.T) {
	if got := sanitize("sync-0965/43..4e?x"); got != "sync-0965-43..4e-x" {
		t.Errorf("sanitize = %q", got)
	}
}
