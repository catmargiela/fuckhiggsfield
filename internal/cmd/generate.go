package cmd

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/catmargiela/fuckhiggsfield/internal/catalog"
	"github.com/catmargiela/fuckhiggsfield/internal/jobs"
	"github.com/catmargiela/fuckhiggsfield/internal/media"
	"github.com/catmargiela/fuckhiggsfield/internal/provider"
	"github.com/catmargiela/fuckhiggsfield/internal/render"
	"github.com/catmargiela/fuckhiggsfield/internal/uploader"
)

func newGenerateCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen"},
		Short:   "Create, cost, list, and wait on generation jobs",
	}
	c.AddCommand(
		newGenerateCreateCmd(app),
		newGenerateGetCmd(app),
		newGenerateListCmd(app),
		newGenerateWaitCmd(app),
		newGenerateCancelCmd(app),
		newGenerateCostCmd(app),
	)
	return c
}

// createOpts are the control flags of `generate create`, kept separate from
// model parameters so a model can never shadow them.
type createOpts struct {
	wait     bool
	timeout  time.Duration
	interval time.Duration
	outDir   string
	quiet    bool
	noSave   bool
	json     bool
	help     bool
	upload   string
	noUpload bool
}

func newGenerateCreateCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:   "create <model> [--param value]...",
		Short: "Create a generation job",
		Long: "Create an image, video, audio, or text job.\n\n" +
			"Model parameters are passed as --name value; run `bara model get <model>`\n" +
			"to see what a model accepts. Media parameters take a local file path or a\n" +
			"URL — local files are inlined automatically.\n\n" +
			"Control flags: --wait, --wait-timeout, --wait-interval, --out, --quiet,\n" +
			"--no-save, --json, --upload [backend], --no-upload.\n\n" +
			"--upload republishes each result to a public URL (see `bara upload backends`).\n" +
			"It also applies to local media inputs, which lifts the inline size limit.",
		Example: "  bara generate create gpt_image_2 --prompt \"a cat in a spaceship\" --wait\n" +
			"  bara generate create flux_dev --prompt \"golden hour city\" --width 1280 --height 720 --wait\n" +
			"  bara generate create qwen_image_edit --prompt \"make it snow\" --image ./photo.png --wait\n" +
			"  bara generate create wan_2_6_i2v --prompt \"slow push in\" --image ./still.png --duration 5 --wait",
		// Model parameters are discovered at runtime, so cobra must not try to
		// parse them as its own flags.
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := createOpts{timeout: 10 * time.Minute, interval: 3 * time.Second}
			if len(args) == 0 || hasHelpFlag(args) {
				return cmd.Help()
			}
			if err := app.Load(); err != nil {
				return err
			}
			modelID := args[0]
			if strings.HasPrefix(modelID, "-") {
				return fmt.Errorf("expected a model name first, got %q\n\nUsage: bara generate create <model> [--param value]...", modelID)
			}
			m, err := app.Catalog.Get(modelID)
			if err != nil {
				return err
			}
			values, err := parseParams(m, args[1:], &opts)
			if err != nil {
				return err
			}
			if opts.json {
				app.JSON = true
			}
			return runCreate(cmd.Context(), app, m, values, opts)
		},
	}
	return c
}

func hasHelpFlag(args []string) bool {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			return true
		}
	}
	return false
}

// parseParams walks `--name value` tokens against the model schema, peeling off
// control flags as it goes.
func parseParams(m catalog.Model, args []string, opts *createOpts) (map[string]any, error) {
	values := map[string]any{}
	lists := map[string][]string{}

	for i := 0; i < len(args); i++ {
		tok := args[i]
		if tok == "-q" {
			opts.quiet = true
			continue
		}
		if !strings.HasPrefix(tok, "--") {
			return nil, fmt.Errorf("unexpected argument %q — parameters are passed as --name value", tok)
		}
		name := strings.TrimPrefix(tok, "--")
		inlineVal := ""
		hasInline := false
		if eq := strings.Index(name, "="); eq >= 0 {
			inlineVal, hasInline = name[eq+1:], true
			name = name[:eq]
		}
		name = strings.ReplaceAll(name, "-", "_")

		// next returns the following token as a value, if there is one.
		next := func() (string, bool) {
			if hasInline {
				return inlineVal, true
			}
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				i++
				return args[i], true
			}
			return "", false
		}

		switch name {
		case "wait":
			if v, ok := next(); ok {
				b, err := strconv.ParseBool(v)
				if err != nil {
					return nil, fmt.Errorf("--wait expects true or false, got %q", v)
				}
				opts.wait = b
			} else {
				opts.wait = true
			}
			continue
		case "wait_timeout", "timeout":
			v, ok := next()
			if !ok {
				return nil, fmt.Errorf("--wait-timeout needs a duration, e.g. 20m")
			}
			d, err := time.ParseDuration(v)
			if err != nil {
				return nil, fmt.Errorf("--wait-timeout %q is not a duration (try 20m): %w", v, err)
			}
			opts.timeout = d
			continue
		case "wait_interval", "interval":
			v, ok := next()
			if !ok {
				return nil, fmt.Errorf("--wait-interval needs a duration, e.g. 5s")
			}
			d, err := time.ParseDuration(v)
			if err != nil {
				return nil, fmt.Errorf("--wait-interval %q is not a duration (try 5s): %w", v, err)
			}
			opts.interval = d
			continue
		case "out", "output_dir":
			v, ok := next()
			if !ok {
				return nil, fmt.Errorf("--out needs a directory")
			}
			opts.outDir = v
			continue
		case "quiet":
			opts.quiet = true
			continue
		case "no_save":
			opts.noSave = true
			continue
		case "upload":
			// Bare --upload means "use the configured backend".
			if v, ok := next(); ok {
				opts.upload = v
			} else {
				opts.upload = "-"
			}
			continue
		case "no_upload":
			opts.noUpload = true
			continue
		case "json":
			opts.json = true
			continue
		case "no_color":
			render.NoColor = true
			continue
		}

		p, ok := m.Param(name)
		if !ok {
			return nil, unknownParamError(m, name)
		}
		if p.Type == catalog.TypeBool {
			if v, ok := next(); ok {
				b, err := strconv.ParseBool(v)
				if err != nil {
					return nil, fmt.Errorf("--%s expects true or false, got %q", name, v)
				}
				values[name] = b
			} else {
				values[name] = true
			}
			continue
		}
		v, ok := next()
		if !ok {
			return nil, fmt.Errorf("--%s needs a value", name)
		}
		if p.Type == catalog.TypeStringArray {
			lists[name] = append(lists[name], splitCommas(v)...)
			continue
		}
		coerced, err := catalog.Coerce(p, v)
		if err != nil {
			return nil, err
		}
		values[name] = coerced
	}

	for name, items := range lists {
		p, _ := m.Param(name)
		if err := validateList(p, items); err != nil {
			return nil, err
		}
		values[name] = items
	}
	return values, nil
}

func validateList(p catalog.Param, items []string) error {
	if len(p.Enum) == 0 {
		return nil
	}
	for _, it := range items {
		if _, err := catalog.Coerce(catalog.Param{Name: p.Name, Type: catalog.TypeString, Enum: p.Enum}, it); err != nil {
			return err
		}
	}
	return nil
}

func splitCommas(s string) []string {
	// A data URI or a URL with a comma must survive intact.
	if strings.HasPrefix(s, "data:") || strings.Contains(s, "://") {
		return []string{s}
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{s}
	}
	return out
}

func unknownParamError(m catalog.Model, name string) error {
	var names []string
	for _, p := range m.Params {
		names = append(names, "--"+p.Name)
	}
	sort.Strings(names)
	return fmt.Errorf("%s has no parameter --%s\n\nAccepted: %s\n\nRun `bara model get %s` for details",
		m.ID, name, strings.Join(names, " "), m.ID)
}

func runCreate(ctx context.Context, app *App, m catalog.Model, values map[string]any, opts createOpts) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := catalog.Validate(m, values); err != nil {
		return err
	}
	values = catalog.ApplyDefaults(m, values)

	var up uploader.Uploader
	if !opts.noUpload {
		override := opts.upload
		if override == "-" {
			// Bare --upload: fall back to the configured backend, and if none
			// is set, say so rather than silently doing nothing.
			override = ""
			if app.Cfg.UploadBackend == "" || app.Cfg.UploadBackend == uploader.BackendNone {
				return fmt.Errorf("--upload needs a backend: pass one (--upload catbox) or set a default with `bara settings set upload_backend <name>`\n\nRun `bara upload backends` to see the options")
			}
		}
		var err error
		up, err = app.Uploader(override)
		if err != nil {
			return err
		}
	}

	// RunPod fetches media by URL, so local paths must be resolved first.
	// OpenAI takes multipart uploads, so its provider handles paths directly.
	if m.Provider == "runpod" {
		if err := resolveMediaParams(cmdContext(ctx), m, values, up); err != nil {
			return err
		}
	}

	p, err := app.ProviderFor(m)
	if err != nil {
		return err
	}
	if !opts.quiet && !app.JSON {
		fmt.Fprintf(app.Err, "%s %s via %s…\n", render.Dim("submitting"), render.Bold(m.ID), m.Provider)
	}
	job, err := p.Create(ctx, m, values)
	if err != nil {
		return err
	}
	job.Model = m.ID

	store, err := jobs.Open()
	if err != nil {
		return err
	}
	store.Put(job)
	if err := store.Save(); err != nil {
		return err
	}

	if opts.wait && !job.Status.Terminal() {
		job, err = waitForJob(ctx, app, store, job, opts.timeout, opts.interval, opts.quiet)
		if err != nil {
			return err
		}
	}

	if job.Status.OK() && !opts.noSave {
		if err := saveOutputs(ctx, app, job, opts.outDir); err != nil {
			return err
		}
		if up != nil {
			if err := uploadOutputs(ctx, up, job); err != nil {
				return err
			}
			if !opts.quiet && !app.JSON {
				fmt.Fprintf(app.Err, "%s\n", render.Dim("published via "+up.Describe()))
			}
		}
		store.Put(job)
		if err := store.Save(); err != nil {
			return err
		}
	}
	return printJob(app, job, opts.quiet)
}

func cmdContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// resolveMediaParams turns local paths into something RunPod can fetch. With an
// uploader configured the file is published and referenced by URL, which avoids
// the size ceiling that inline data URIs impose.
func resolveMediaParams(ctx context.Context, m catalog.Model, values map[string]any, up uploader.Uploader) error {
	resolve := func(ref string) (string, error) {
		if up == nil || media.IsURL(ref) {
			return media.Resolve(ref)
		}
		return uploadPath(ctx, up, ref)
	}
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
			resolved, err := resolve(t)
			if err != nil {
				return fmt.Errorf("--%s: %w", p.Name, err)
			}
			values[p.Name] = resolved
		case []string:
			out := make([]string, 0, len(t))
			for _, ref := range t {
				resolved, err := resolve(ref)
				if err != nil {
					return fmt.Errorf("--%s: %w", p.Name, err)
				}
				out = append(out, resolved)
			}
			values[p.Name] = out
		}
	}
	return nil
}

// uploadOutputs republishes each saved artifact to a durable URL.
func uploadOutputs(ctx context.Context, up uploader.Uploader, job *provider.Job) error {
	for i := range job.Outputs {
		o := &job.Outputs[i]
		if o.PublicURL != "" || o.LocalPath == "" {
			continue
		}
		url, err := uploadPath(cmdContext(ctx), up, o.LocalPath)
		if err != nil {
			return err
		}
		o.PublicURL = url
	}
	return nil
}

func waitForJob(ctx context.Context, app *App, store *jobs.Store, job *provider.Job, timeout, interval time.Duration, quiet bool) (*provider.Job, error) {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	deadline := time.Now().Add(timeout)
	p, err := app.ProviderByName(job.Provider)
	if err != nil {
		return job, err
	}
	spin := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	for i := 0; ; i++ {
		if time.Now().After(deadline) {
			return job, fmt.Errorf("timed out after %s waiting for job %s (it may still finish; run `bara generate get %s`)", timeout, job.ID, job.ID)
		}
		select {
		case <-ctx.Done():
			return job, ctx.Err()
		case <-time.After(interval):
		}
		updated, err := p.Get(ctx, job.Endpoint, job.ID)
		if err != nil {
			return job, err
		}
		updated.Model = job.Model
		updated.Created = job.Created
		job = updated
		store.Put(job)
		_ = store.Save()
		if !quiet && !app.JSON {
			fmt.Fprintf(app.Err, "\r%s %s %s", spin[i%len(spin)], render.Dim(job.ID), render.StatusColor(string(job.Status)))
		}
		if job.Status.Terminal() {
			if !quiet && !app.JSON {
				fmt.Fprintln(app.Err)
			}
			if !job.Status.OK() {
				msg := job.Error
				if msg == "" {
					msg = "no error detail returned"
				}
				return job, fmt.Errorf("job %s ended with status %s: %s", job.ID, job.Status, msg)
			}
			return job, nil
		}
	}
}

func saveOutputs(ctx context.Context, app *App, job *provider.Job, outDir string) error {
	if outDir == "" {
		outDir = app.Cfg.OutputDir
	}
	for i := range job.Outputs {
		o := &job.Outputs[i]
		if o.LocalPath != "" || o.Text != "" {
			continue
		}
		name := fmt.Sprintf("%s-%d", sanitize(job.ID), i+1)
		var (
			path string
			err  error
		)
		switch {
		case o.B64 != "":
			path, err = media.SaveBase64(outDir, name, o.Ext, o.B64)
		case o.URL != "":
			path, err = media.Download(ctx, app.HTTP, outDir, name, o.Ext, o.URL)
		default:
			continue
		}
		if err != nil {
			return err
		}
		o.LocalPath = path
	}
	return nil
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func sanitize(s string) string {
	s = unsafeName.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func printJob(app *App, job *provider.Job, quiet bool) error {
	if app.JSON {
		return render.JSON(app.Out, job)
	}
	if quiet {
		// One line per artifact, most useful reference first, so the output
		// pipes straight into another command.
		for _, o := range job.Outputs {
			switch {
			case o.PublicURL != "":
				fmt.Fprintln(app.Out, o.PublicURL)
			case o.LocalPath != "":
				fmt.Fprintln(app.Out, o.LocalPath)
			case o.URL != "":
				fmt.Fprintln(app.Out, o.URL)
			case o.Text != "":
				fmt.Fprintln(app.Out, o.Text)
			}
		}
		return nil
	}
	pairs := [][2]string{
		{"Job", job.ID},
		{"Model", job.Model},
		{"Provider", job.Provider},
		{"Status", render.StatusColor(string(job.Status))},
	}
	if job.Cost > 0 {
		pairs = append(pairs, [2]string{"Cost", fmt.Sprintf("$%.4f", job.Cost)})
	}
	if job.Error != "" {
		pairs = append(pairs, [2]string{"Error", job.Error})
	}
	render.KeyValues(app.Out, pairs)
	if len(job.Outputs) == 0 {
		return nil
	}
	fmt.Fprintln(app.Out)
	for _, o := range job.Outputs {
		switch {
		case o.Text != "":
			fmt.Fprintln(app.Out, o.Text)
		case o.LocalPath != "":
			fmt.Fprintf(app.Out, "%s %s\n", render.Bold("saved"), o.LocalPath)
			if o.PublicURL != "" {
				fmt.Fprintf(app.Out, "%s  %s\n", render.Bold("url"), render.Cyan(o.PublicURL))
			}
			if o.URL != "" {
				// The provider's own link, which typically expires.
				fmt.Fprintf(app.Out, "%s %s\n", render.Dim("  src"), render.Dim(o.URL))
			}
		case o.PublicURL != "":
			fmt.Fprintln(app.Out, o.PublicURL)
		case o.URL != "":
			fmt.Fprintln(app.Out, o.URL)
		}
	}
	return nil
}

func newGenerateGetCmd(app *App) *cobra.Command {
	var refresh bool
	c := &cobra.Command{
		Use:     "get <job_id>",
		Short:   "Show a job",
		Args:    cobra.ExactArgs(1),
		Example: "  bara generate get 9f3c...\n  bara generate get 9f3c... --refresh",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Load(); err != nil {
				return err
			}
			store, err := jobs.Open()
			if err != nil {
				return err
			}
			job, err := store.Get(args[0])
			if err != nil {
				return err
			}
			if refresh && !job.Status.Terminal() && job.Provider == "runpod" {
				p, err := app.ProviderByName(job.Provider)
				if err != nil {
					return err
				}
				updated, err := p.Get(cmd.Context(), job.Endpoint, job.ID)
				if err != nil {
					return err
				}
				updated.Model, updated.Created = job.Model, job.Created
				job = updated
				store.Put(job)
				if err := store.Save(); err != nil {
					return err
				}
			}
			return printJob(app, job, false)
		},
	}
	c.Flags().BoolVar(&refresh, "refresh", false, "poll the provider once before printing")
	return c
}

func newGenerateListCmd(app *App) *cobra.Command {
	var (
		status string
		limit  int
	)
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List recent jobs",
		Args:    cobra.NoArgs,
		Example: "  bara generate list\n  bara generate list --status COMPLETED --limit 10",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := app.Load(); err != nil {
				return err
			}
			store, err := jobs.Open()
			if err != nil {
				return err
			}
			list := store.List(status, limit)
			if app.JSON {
				return render.JSON(app.Out, list)
			}
			rows := make([][]string, 0, len(list))
			for _, j := range list {
				out := ""
				if len(j.Outputs) > 0 {
					out = j.Outputs[0].PublicURL
					if out == "" {
						out = j.Outputs[0].LocalPath
					}
					if out == "" {
						out = j.Outputs[0].URL
					}
					if out == "" {
						out = render.Truncate(j.Outputs[0].Text, 40)
					}
				}
				rows = append(rows, []string{
					render.Truncate(j.ID, 24),
					j.Model,
					render.StatusColor(string(j.Status)),
					j.Created.Local().Format("2006-01-02 15:04"),
					render.Truncate(out, 52),
				})
			}
			render.Table(app.Out, []string{"job", "model", "status", "created", "output"}, rows)
			return nil
		},
	}
	c.Flags().StringVar(&status, "status", "", "filter by status (QUEUED, IN_PROGRESS, COMPLETED, FAILED)")
	c.Flags().IntVar(&limit, "limit", 20, "maximum rows (0 for all)")
	return c
}

func newGenerateWaitCmd(app *App) *cobra.Command {
	var (
		timeout  time.Duration
		interval time.Duration
		quiet    bool
		outDir   string
	)
	c := &cobra.Command{
		Use:     "wait <job_id>",
		Short:   "Poll a job until it finishes",
		Args:    cobra.ExactArgs(1),
		Example: "  bara generate wait 9f3c...\n  bara generate wait 9f3c... --timeout 20m --interval 5s",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Load(); err != nil {
				return err
			}
			store, err := jobs.Open()
			if err != nil {
				return err
			}
			job, err := store.Get(args[0])
			if err != nil {
				return err
			}
			if !job.Status.Terminal() {
				job, err = waitForJob(cmd.Context(), app, store, job, timeout, interval, quiet)
				if err != nil {
					return err
				}
			}
			if job.Status.OK() {
				if err := saveOutputs(cmd.Context(), app, job, outDir); err != nil {
					return err
				}
				store.Put(job)
				if err := store.Save(); err != nil {
					return err
				}
			}
			return printJob(app, job, quiet)
		},
	}
	c.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "max wait duration")
	c.Flags().DurationVar(&interval, "interval", 3*time.Second, "poll interval")
	c.Flags().BoolVarP(&quiet, "quiet", "q", false, "no progress output")
	c.Flags().StringVar(&outDir, "out", "", "directory to save outputs into")
	return c
}

func newGenerateCancelCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <job_id>",
		Short: "Cancel a queued or running job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Load(); err != nil {
				return err
			}
			store, err := jobs.Open()
			if err != nil {
				return err
			}
			job, err := store.Get(args[0])
			if err != nil {
				return err
			}
			p, err := app.ProviderByName(job.Provider)
			if err != nil {
				return err
			}
			cancel, ok := p.(provider.Canceler)
			if !ok {
				return fmt.Errorf("%s jobs cannot be cancelled", job.Provider)
			}
			if err := cancel.Cancel(cmd.Context(), job.Endpoint, job.ID); err != nil {
				return err
			}
			job.Status = provider.StatusCancelled
			store.Put(job)
			if err := store.Save(); err != nil {
				return err
			}
			fmt.Fprintf(app.Out, "cancelled %s\n", job.ID)
			return nil
		},
	}
}

func newGenerateCostCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:   "cost <model> [--param value]...",
		Short: "Show what a generation is expected to cost",
		Long: "Print the provider's documented price for a model, and — when the pricing\n" +
			"is a simple per-unit rate and the relevant parameter is supplied — a\n" +
			"computed estimate.\n\n" +
			"This reads published rates; it is not a quote from the provider.",
		DisableFlagParsing: true,
		Example:            "  bara generate cost flux_dev --width 1024 --height 1024\n  bara generate cost vidu_q3_t2v --duration 10",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || hasHelpFlag(args) {
				return cmd.Help()
			}
			if err := app.Load(); err != nil {
				return err
			}
			m, err := app.Catalog.Get(args[0])
			if err != nil {
				return err
			}
			opts := createOpts{}
			values, err := parseParams(m, args[1:], &opts)
			if err != nil {
				return err
			}
			values = catalog.ApplyDefaults(m, values)
			est, basis, ok := estimateCost(m, values)
			if opts.json {
				app.JSON = true
			}
			if app.JSON {
				out := map[string]any{"model": m.ID, "price": m.Price}
				if ok {
					out["estimate_usd"] = est
					out["basis"] = basis
				}
				return render.JSON(app.Out, out)
			}
			pairs := [][2]string{
				{"Model", m.ID},
				{"Published price", m.Price},
			}
			if ok {
				pairs = append(pairs,
					[2]string{"Estimate", fmt.Sprintf("$%.4f", est)},
					[2]string{"Basis", basis},
				)
			} else {
				pairs = append(pairs, [2]string{"Estimate", render.Dim("not computable from the published rate — see above")})
			}
			render.KeyValues(app.Out, pairs)
			return nil
		},
	}
	return c
}

var (
	rePerImage = regexp.MustCompile(`\$([0-9.]+)\s*per image`)
	rePerSec   = regexp.MustCompile(`\$([0-9.]+)\s*(?:per second|/sec)`)
	rePerMP    = regexp.MustCompile(`\$([0-9.]+)\s*per megapixel`)
	rePerKChar = regexp.MustCompile(`\$([0-9.]+)\s*per 1[,]?000 characters`)
)

// estimateCost derives a figure from the published rate when the rate is a
// single per-unit price and the driving parameter is known. Anything with
// tiered or range pricing returns ok=false rather than a guess.
func estimateCost(m catalog.Model, values map[string]any) (usd float64, basis string, ok bool) {
	price := m.Price
	// A range (e.g. "$0.14 - $0.24") means tiered pricing; refuse to pick one.
	if strings.Count(price, "$") > 1 {
		return 0, "", false
	}
	num := func(re *regexp.Regexp) (float64, bool) {
		mm := re.FindStringSubmatch(price)
		if len(mm) < 2 {
			return 0, false
		}
		f, err := strconv.ParseFloat(mm[1], 64)
		return f, err == nil
	}
	if rate, found := num(rePerImage); found {
		n := intValue(values["n"], 1)
		return rate * float64(n), fmt.Sprintf("%d image(s) x $%g", n, rate), true
	}
	if rate, found := num(rePerSec); found {
		d, has := values["duration"]
		if !has {
			return 0, "", false
		}
		secs := intValue(d, 0)
		if secs <= 0 {
			return 0, "", false
		}
		return rate * float64(secs), fmt.Sprintf("%ds x $%g/sec", secs, rate), true
	}
	if rate, found := num(rePerMP); found {
		w, hOK := values["width"], true
		h, hOK2 := values["height"]
		if !hOK || !hOK2 {
			return 0, "", false
		}
		mp := float64(intValue(w, 0)) * float64(intValue(h, 0)) / 1e6
		if mp <= 0 {
			return 0, "", false
		}
		return rate * mp, fmt.Sprintf("%.2f MP x $%g/MP", mp, rate), true
	}
	if rate, found := num(rePerKChar); found {
		p, _ := values["prompt"].(string)
		if p == "" {
			return 0, "", false
		}
		k := float64(len([]rune(p))) / 1000
		return rate * k, fmt.Sprintf("%d chars x $%g/1k", len([]rune(p)), rate), true
	}
	return 0, "", false
}

func intValue(v any, def int) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		if n, err := strconv.Atoi(t); err == nil {
			return n
		}
	}
	return def
}
