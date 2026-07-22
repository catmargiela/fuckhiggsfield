package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/catmargiela/fuckhiggsfield/internal/catalog"
	"github.com/catmargiela/fuckhiggsfield/internal/mcpserver"
	"github.com/catmargiela/fuckhiggsfield/internal/media"
	"github.com/catmargiela/fuckhiggsfield/internal/provider"
	"github.com/catmargiela/fuckhiggsfield/internal/render"
)

func newMCPCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:   "mcp",
		Short: "Run the Model Context Protocol server",
		Long: "Expose the model catalog and generation to MCP clients.\n\n" +
			"Register it with an MCP client by pointing the client at `bara mcp serve`.",
	}
	c.AddCommand(newMCPServeCmd(app), newMCPConfigCmd(app))
	return c
}

func newMCPServeCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Serve MCP over stdio",
		Long: "Serve MCP over stdio (newline-delimited JSON-RPC 2.0).\n\n" +
			"Tools: list_models, get_model, generate, get_job, list_jobs.\n" +
			"Nothing is written to stdout except protocol messages.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := app.Load(); err != nil {
				return err
			}
			up, err := app.Uploader("")
			if err != nil {
				return err
			}
			deps := mcpserver.Deps{
				Catalog: app.Catalog,
				ProviderFor: func(m catalog.Model) (provider.Provider, error) {
					return app.ProviderFor(m)
				},
				OutputDir: app.Cfg.OutputDir,
				Download: func(ctx context.Context, dir, name, ext, url string) (string, error) {
					return media.Download(ctx, app.HTTP, dir, name, ext, url)
				},
				UploadBackend: app.Cfg.UploadBackend,
				ServerName:    "bara",
				ServerVersion: Version,
			}
			if up != nil {
				deps.Upload = func(ctx context.Context, path string) (string, error) {
					return uploadPath(ctx, up, path)
				}
			}
			srv := mcpserver.New(deps, os.Stdout)
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			return srv.Serve(ctx, os.Stdin)
		},
	}
}

func newMCPConfigCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Print an MCP client config snippet",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			exe, err := os.Executable()
			if err != nil || exe == "" {
				exe = "bara"
			}
			snippet := map[string]any{
				"mcpServers": map[string]any{
					"bara": map[string]any{
						"command": exe,
						"args":    []string{"mcp", "serve"},
					},
				},
			}
			if app.JSON {
				return render.JSON(app.Out, snippet)
			}
			fmt.Fprintln(app.Out, render.Dim("Add this to your MCP client config (e.g. Claude Desktop):"))
			fmt.Fprintln(app.Out)
			if err := render.JSON(app.Out, snippet); err != nil {
				return err
			}
			fmt.Fprintln(app.Out)
			fmt.Fprintln(app.Out, render.Dim("Claude Code: claude mcp add bara -- "+exe+" mcp serve"))
			return nil
		},
	}
}
