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
	m := testModel(t, "gpt_image_2")
	opts := createOpts{}
	got, err := parseParams(m, []string{
		"--prompt", "a quiet beach at sunrise",
		"--n", "2",
		"--size", "1536x1024",
		"--quality", "high",
	}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	want := map[string]any{
		"prompt":  "a quiet beach at sunrise",
		"n":       int64(2),
		"size":    "1536x1024",
		"quality": "high",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parsed = %#v, want %#v", got, want)
	}
}

// The value-consuming closure advances the loop index; if that ever stopped
// working, every second token would be misread as a flag.
func TestParseParamsDoesNotMisreadValuesAsFlags(t *testing.T) {
	m := testModel(t, "gpt_image_2")
	opts := createOpts{}
	got, err := parseParams(m, []string{"--prompt", "cat", "--n", "3"}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	if got["prompt"] != "cat" {
		t.Errorf("prompt = %v, want cat", got["prompt"])
	}
	if got["n"] != int64(3) {
		t.Errorf("n = %v, want 3", got["n"])
	}
	if len(got) != 2 {
		t.Errorf("parsed %d params, want 2: %#v", len(got), got)
	}
}

func TestParseParamsInlineEquals(t *testing.T) {
	m := testModel(t, "gpt_image_2")
	opts := createOpts{}
	got, err := parseParams(m, []string{"--prompt=hello world", "--n=4"}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	if got["prompt"] != "hello world" {
		t.Errorf("prompt = %v", got["prompt"])
	}
	if got["n"] != int64(4) {
		t.Errorf("n = %v", got["n"])
	}
}

func TestParseParamsDashesNormalizeToUnderscores(t *testing.T) {
	m := testModel(t, "gpt_image_2")
	opts := createOpts{}
	got, err := parseParams(m, []string{"--prompt", "x", "--output-format", "webp"}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	if got["output_format"] != "webp" {
		t.Errorf("output_format = %v", got["output_format"])
	}
}

// gpt_image_2 has no boolean parameter, so the bool path runs against a
// minimal model.
var boolModel = catalog.Model{ID: "bools", Params: []catalog.Param{
	{Name: "prompt", Type: catalog.TypeString},
	{Name: "safe", Type: catalog.TypeBool},
}}

func TestParseParamsBoolBare(t *testing.T) {
	opts := createOpts{}
	got, err := parseParams(boolModel, []string{"--prompt", "x", "--safe"}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	if got["safe"] != true {
		t.Errorf("bare bool = %v, want true", got["safe"])
	}
}

func TestParseParamsBoolExplicit(t *testing.T) {
	opts := createOpts{}
	got, err := parseParams(boolModel, []string{"--prompt", "x", "--safe", "false"}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	if got["safe"] != false {
		t.Errorf("explicit bool = %v, want false", got["safe"])
	}
}

func TestParseParamsRepeatedListAccumulates(t *testing.T) {
	m := testModel(t, "gpt_image_2")
	opts := createOpts{}
	got, err := parseParams(m, []string{
		"--prompt", "combine",
		"--image", "a.png",
		"--image", "b.png",
	}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	want := []string{"a.png", "b.png"}
	if !reflect.DeepEqual(got["image"], want) {
		t.Fatalf("image = %#v, want %#v", got["image"], want)
	}
}

func TestParseParamsCommaSeparatedList(t *testing.T) {
	m := testModel(t, "gpt_image_2")
	opts := createOpts{}
	got, err := parseParams(m, []string{"--prompt", "x", "--image", "a.png,b.png"}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	want := []string{"a.png", "b.png"}
	if !reflect.DeepEqual(got["image"], want) {
		t.Fatalf("image = %#v, want %#v", got["image"], want)
	}
}

// A URL may contain commas but is a single value.
func TestParseParamsURLStaysWhole(t *testing.T) {
	m := testModel(t, "gpt_image_2")
	opts := createOpts{}
	url := "https://files.catbox.moe/a,b.png"
	got, err := parseParams(m, []string{"--prompt", "x", "--image", url}, &opts)
	if err != nil {
		t.Fatalf("parseParams: %v", err)
	}
	want := []string{url}
	if !reflect.DeepEqual(got["image"], want) {
		t.Fatalf("image = %#v, want %#v", got["image"], want)
	}
}

func TestParseParamsControlFlags(t *testing.T) {
	m := testModel(t, "gpt_image_2")
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
	m := testModel(t, "gpt_image_2")
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
	m := testModel(t, "gpt_image_2")
	opts := createOpts{}
	_, err := parseParams(m, []string{"--nope", "1"}, &opts)
	if err == nil {
		t.Fatal("expected an error for an unknown parameter")
	}
}

func TestParseParamsEnumRejected(t *testing.T) {
	m := testModel(t, "gpt_image_2")
	opts := createOpts{}
	_, err := parseParams(m, []string{"--prompt", "x", "--output_format", "tiff"}, &opts)
	if err == nil {
		t.Fatal("expected an error for a value outside the enum")
	}
}

func TestParseParamsRangeRejected(t *testing.T) {
	m := testModel(t, "gpt_image_2")
	opts := createOpts{}
	_, err := parseParams(m, []string{"--prompt", "x", "--n", "11"}, &opts)
	if err == nil {
		t.Fatal("expected an error for a value above max")
	}
}

func TestParseParamsMissingValue(t *testing.T) {
	m := testModel(t, "gpt_image_2")
	opts := createOpts{}
	_, err := parseParams(m, []string{"--prompt"}, &opts)
	if err == nil {
		t.Fatal("expected an error when a value-taking flag has no value")
	}
}

func TestEstimateCostPerImage(t *testing.T) {
	m := catalog.Model{ID: "flat", Price: "$0.02 per image"}
	usd, basis, ok := estimateCost(m, map[string]any{"prompt": "x", "n": int64(3)})
	if !ok {
		t.Fatal("expected an estimate for a flat per-image rate")
	}
	if usd != 0.06 {
		t.Errorf("usd = %v, want 0.06", usd)
	}
	if basis == "" {
		t.Error("basis should explain the computation")
	}
}

// Tiered prices ("$0.14 - $0.24") must not be collapsed into one number.
func TestEstimateCostRefusesTieredPricing(t *testing.T) {
	m := catalog.Model{ID: "tiered", Price: "$0.14 - $0.24 per image"}
	if _, _, ok := estimateCost(m, map[string]any{"prompt": "x"}); ok {
		t.Fatal("tiered pricing should not produce a single estimate")
	}
}

// gpt_image_2 is billed by size and quality, which no flat rate captures.
func TestEstimateCostGPTImage2IsNotComputable(t *testing.T) {
	m := testModel(t, "gpt_image_2")
	if _, _, ok := estimateCost(m, map[string]any{"prompt": "x"}); ok {
		t.Fatal("gpt_image_2 pricing should not produce an estimate")
	}
}

func TestSanitizeJobID(t *testing.T) {
	if got := sanitize("sync-0965/43..4e?x"); got != "sync-0965-43..4e-x" {
		t.Errorf("sanitize = %q", got)
	}
}
