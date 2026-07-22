// Package cmd wires the bara command tree.
package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/catmargiela/fuckhiggsfield/internal/catalog"
	"github.com/catmargiela/fuckhiggsfield/internal/config"
	"github.com/catmargiela/fuckhiggsfield/internal/provider"
	"github.com/catmargiela/fuckhiggsfield/internal/provider/openai"
	"github.com/catmargiela/fuckhiggsfield/internal/provider/runpod"
	"github.com/catmargiela/fuckhiggsfield/internal/render"
	"github.com/catmargiela/fuckhiggsfield/internal/uploader"
)

// Build metadata, injected via -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// App carries everything a command needs.
type App struct {
	Cfg     *config.Config
	Creds   *config.Credentials
	Catalog *catalog.Catalog
	HTTP    *http.Client

	JSON bool
	Out  io.Writer
	Err  io.Writer
}

// Load builds the app context, reading config and credentials from disk.
func (a *App) Load() error {
	if a.Catalog != nil {
		return nil
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cr, err := config.LoadCredentials()
	if err != nil {
		return err
	}
	cat, err := catalog.Load()
	if err != nil {
		return err
	}
	a.Cfg, a.Creds, a.Catalog = cfg, cr, cat
	if a.HTTP == nil {
		a.HTTP = &http.Client{Timeout: 300 * time.Second}
	}
	if a.Out == nil {
		a.Out = os.Stdout
	}
	if a.Err == nil {
		a.Err = os.Stderr
	}
	return nil
}

// ProviderFor returns a ready client for the model's backend.
func (a *App) ProviderFor(m catalog.Model) (provider.Provider, error) {
	switch config.Provider(m.Provider) {
	case config.ProviderOpenAI:
		key, err := a.Creds.KeyFor(config.ProviderOpenAI)
		if err != nil {
			return nil, err
		}
		return openai.New(a.Cfg.OpenAIBaseURL, key, a.HTTP), nil
	case config.ProviderRunPod:
		key, err := a.Creds.KeyFor(config.ProviderRunPod)
		if err != nil {
			return nil, err
		}
		return runpod.New(a.Cfg.RunPodBaseURL, key, a.HTTP), nil
	}
	return nil, fmt.Errorf("model %s has unknown provider %q", m.ID, m.Provider)
}

// Uploader builds the configured uploader. It returns (nil, nil) when
// uploading is disabled, which is the default. Passing override forces a
// backend for a single command.
func (a *App) Uploader(override string) (uploader.Uploader, error) {
	backend := a.Cfg.UploadBackend
	if override != "" {
		backend = override
	}
	return uploader.New(uploader.Config{
		Backend:        backend,
		CatboxUserHash: a.Creds.CatboxUserHash,
		LitterboxTime:  a.Cfg.LitterboxTime,
		S3Endpoint:     a.Cfg.S3Endpoint,
		S3Bucket:       a.Cfg.S3Bucket,
		S3Region:       a.Cfg.S3Region,
		S3AccessKey:    a.Creds.S3AccessKey,
		S3SecretKey:    a.Creds.S3SecretKey,
		S3Prefix:       a.Cfg.S3Prefix,
		S3PathStyle:    a.Cfg.S3UsePathStyle(),
		PublicBaseURL:  a.Cfg.UploadBaseURL,
		HTTP:           a.HTTP,
	})
}

// ProviderByName returns a client for a provider key, for polling jobs whose
// model is no longer in the catalog.
func (a *App) ProviderByName(name string) (provider.Provider, error) {
	switch config.Provider(name) {
	case config.ProviderOpenAI:
		key, err := a.Creds.KeyFor(config.ProviderOpenAI)
		if err != nil {
			return nil, err
		}
		return openai.New(a.Cfg.OpenAIBaseURL, key, a.HTTP), nil
	case config.ProviderRunPod:
		key, err := a.Creds.KeyFor(config.ProviderRunPod)
		if err != nil {
			return nil, err
		}
		return runpod.New(a.Cfg.RunPodBaseURL, key, a.HTTP), nil
	}
	return nil, fmt.Errorf("unknown provider %q", name)
}

// NewRoot builds the command tree.
func NewRoot() *cobra.Command {
	app := &App{Out: os.Stdout, Err: os.Stderr}

	root := &cobra.Command{
		Use:     "bara",
		Aliases: []string{"br"},
		Short:   "Generate images, video, and audio from the terminal with your own API keys",
		Long: "Run image, video, audio, and text generation from the terminal.\n\n" +
			"Models are served by OpenAI (gpt-image-*) and RunPod Public Endpoints,\n" +
			"billed directly to the API keys you configure with `bara settings set`.",
		Example: "  bara settings set openai_api_key sk-...\n" +
			"  bara model list --image\n" +
			"  bara generate create gpt_image_2 --prompt \"a cat in a spaceship\" --wait\n" +
			"  bara generate create flux_dev --prompt \"a quiet beach at sunrise\" --wait",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
	}
	root.SetOut(os.Stdout)
	root.PersistentFlags().BoolVar(&app.JSON, "json", false, "print raw JSON responses")
	root.PersistentFlags().BoolVar(&render.NoColor, "no-color", render.NoColor, "disable color output")
	root.SetVersionTemplate(fmt.Sprintf("bara %s (%s) built %s\n", Version, Commit, Date))

	root.AddCommand(
		newSettingsCmd(app),
		newModelCmd(app),
		newGenerateCmd(app),
		newUploadCmd(app),
		newMCPCmd(app),
		newVersionCmd(app),
	)
	return root
}

// Execute runs the CLI and returns a process exit code.
func Execute() int {
	root := NewRoot()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "bara: "+err.Error())
		return 1
	}
	return 0
}
