package cmd

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/catmargiela/fuckhiggsfield/internal/render"
	"github.com/catmargiela/fuckhiggsfield/internal/uploader"
)

func newUploadCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:   "upload",
		Short: "Publish files to a public URL",
		Long: "Publish local files to a public URL.\n\n" +
			"Uploading is off by default. Pick a backend with\n" +
			"`bara settings set upload_backend <name>`, or force one per command\n" +
			"with --backend. Run `bara upload backends` to see the options.",
	}
	c.AddCommand(newUploadFileCmd(app), newUploadBackendsCmd(app))
	return c
}

func newUploadBackendsCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:     "backends",
		Aliases: []string{"list"},
		Short:   "List upload backends",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := app.Load(); err != nil {
				return err
			}
			current := app.Cfg.UploadBackend
			if current == "" {
				current = uploader.BackendNone
			}
			if app.JSON {
				var out []map[string]string
				for _, b := range uploader.Backends() {
					out = append(out, map[string]string{"name": b[0], "description": b[1], "active": fmt.Sprint(b[0] == current)})
				}
				return render.JSON(app.Out, out)
			}
			var rows [][]string
			for _, b := range uploader.Backends() {
				marker := ""
				if b[0] == current {
					marker = "*"
				}
				rows = append(rows, []string{marker, b[0], b[1]})
			}
			render.Table(app.Out, []string{"", "backend", "description"}, rows)
			fmt.Fprintf(app.Out, "\n%s\n", render.Dim("* = active. Change it with `bara settings set upload_backend <name>`."))
			fmt.Fprintf(app.Out, "%s\n", render.Dim("For s3 also set: s3_endpoint, s3_bucket, s3_access_key_id, s3_secret_access_key, upload_base_url."))
			return nil
		},
	}
}

func newUploadFileCmd(app *App) *cobra.Command {
	var backend string
	c := &cobra.Command{
		Use:     "create <file>...",
		Aliases: []string{"file"},
		Short:   "Upload local files and print their URLs",
		Args:    cobra.MinimumNArgs(1),
		Example: "  bara upload create ./render.png\n" +
			"  bara upload create ./a.png ./b.png --backend catbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Load(); err != nil {
				return err
			}
			up, err := app.Uploader(backend)
			if err != nil {
				return err
			}
			if up == nil {
				return fmt.Errorf("uploading is disabled — run `bara settings set upload_backend catbox` (or pass --backend), see `bara upload backends`")
			}
			type result struct {
				File string `json:"file"`
				URL  string `json:"url"`
			}
			var results []result
			for _, path := range args {
				url, err := uploadPath(cmd.Context(), up, path)
				if err != nil {
					return err
				}
				results = append(results, result{File: path, URL: url})
				if !app.JSON {
					fmt.Fprintln(app.Out, url)
				}
			}
			if app.JSON {
				return render.JSON(app.Out, results)
			}
			fmt.Fprintf(app.Err, "%s\n", render.Dim(up.Describe()))
			return nil
		},
	}
	c.Flags().StringVar(&backend, "backend", "", "force a backend for this command (catbox, litterbox, s3)")
	return c
}

// uploadPath reads a local file and publishes it.
func uploadPath(ctx context.Context, up uploader.Uploader, path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return up.Upload(ctx, filepath.Base(path), data, contentTypeFor(path, data))
}

// contentTypeFor picks a MIME type from the extension, falling back to sniffing.
func contentTypeFor(path string, data []byte) string {
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(path))); ct != "" {
		if i := strings.Index(ct, ";"); i >= 0 {
			ct = ct[:i]
		}
		return ct
	}
	ct := http.DetectContentType(data)
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = ct[:i]
	}
	return ct
}
