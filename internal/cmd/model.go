package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/catmargiela/fuckhiggsfield/internal/catalog"
	"github.com/catmargiela/fuckhiggsfield/internal/render"
)

func newModelCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:     "model",
		Aliases: []string{"models"},
		Short:   "List models and inspect their parameters",
	}
	c.AddCommand(newModelListCmd(app), newModelGetCmd(app))
	return c
}

func newModelListCmd(app *App) *cobra.Command {
	var (
		image, video, audio, text bool
		providerFlag, query       string
	)
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available models",
		Args:    cobra.NoArgs,
		Example: "  bara model list\n  bara model list --image\n  bara model list --provider runpod --query wan",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := app.Load(); err != nil {
				return err
			}
			var kinds []catalog.Kind
			if image {
				kinds = append(kinds, catalog.KindImage)
			}
			if video {
				kinds = append(kinds, catalog.KindVideo)
			}
			if audio {
				kinds = append(kinds, catalog.KindAudio)
			}
			if text {
				kinds = append(kinds, catalog.KindText)
			}
			if len(kinds) == 0 {
				kinds = []catalog.Kind{""}
			}
			var models []catalog.Model
			seen := map[string]bool{}
			for _, k := range kinds {
				for _, m := range app.Catalog.Filter(k, providerFlag, query) {
					if !seen[m.ID] {
						seen[m.ID] = true
						models = append(models, m)
					}
				}
			}
			if app.JSON {
				return render.JSON(app.Out, models)
			}
			rows := make([][]string, 0, len(models))
			for _, m := range models {
				rows = append(rows, []string{m.ID, string(m.Kind), m.Provider, m.Name, render.Dim(m.Price)})
			}
			render.Table(app.Out, []string{"model", "kind", "provider", "name", "price"}, rows)
			fmt.Fprintf(app.Out, "\n%s\n", render.Dim(fmt.Sprintf("%d model(s). Run `bara model get <model>` for parameters.", len(models))))
			return nil
		},
	}
	c.Flags().BoolVar(&image, "image", false, "only image models")
	c.Flags().BoolVar(&video, "video", false, "only video models")
	c.Flags().BoolVar(&audio, "audio", false, "only audio models")
	c.Flags().BoolVar(&text, "text", false, "only text models")
	c.Flags().StringVar(&providerFlag, "provider", "", "filter by provider (openai, runpod)")
	c.Flags().StringVar(&query, "query", "", "substring match on id or name")
	return c
}

func newModelGetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:     "get <model>",
		Short:   "Show a model's parameter schema",
		Args:    cobra.ExactArgs(1),
		Example: "  bara model get gpt_image_2\n  bara model get wan_2_6_i2v --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Load(); err != nil {
				return err
			}
			m, err := app.Catalog.Get(args[0])
			if err != nil {
				return err
			}
			if app.JSON {
				return render.JSON(app.Out, m)
			}
			render.KeyValues(app.Out, [][2]string{
				{"Model", m.ID},
				{"Name", m.Name},
				{"Kind", string(m.Kind)},
				{"Provider", m.Provider},
				{"Endpoint", m.Endpoint},
				{"Price", m.Price},
			})
			if m.Notes != "" {
				fmt.Fprintf(app.Out, "\n%s\n", m.Notes)
			}
			fmt.Fprintf(app.Out, "\n%s\n", render.Bold("Parameters"))
			rows := make([][]string, 0, len(m.Params))
			for _, p := range m.Params {
				req := ""
				if p.Required {
					req = "yes"
				}
				rows = append(rows, []string{
					"--" + p.Name,
					string(p.Type),
					req,
					formatDefault(p),
					constraints(p),
					render.Dim(p.Desc),
				})
			}
			render.Table(app.Out, []string{"flag", "type", "required", "default", "allowed", "description"}, rows)
			fmt.Fprintf(app.Out, "\n%s\n", render.Dim(fmt.Sprintf("Example: bara generate create %s --prompt \"...\" --wait", m.ID)))
			return nil
		},
	}
}

func formatDefault(p catalog.Param) string {
	if p.Default == nil {
		return ""
	}
	switch v := p.Default.(type) {
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%v", v)
	case string:
		return v
	}
	return fmt.Sprintf("%v", p.Default)
}

func constraints(p catalog.Param) string {
	var parts []string
	if len(p.Enum) > 0 {
		parts = append(parts, strings.Join(p.Enum, "|"))
	}
	if p.Min != nil || p.Max != nil {
		lo, hi := "", ""
		if p.Min != nil {
			lo = trimFloat(*p.Min)
		}
		if p.Max != nil {
			hi = trimFloat(*p.Max)
		}
		parts = append(parts, lo+".."+hi)
	}
	if p.Media != "" {
		parts = append(parts, string(p.Media)+" path or URL")
	}
	return strings.Join(parts, " ")
}

func trimFloat(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%v", f)
}
