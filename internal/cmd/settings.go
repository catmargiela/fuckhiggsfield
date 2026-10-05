package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/catmargiela/fuckhiggsfield/internal/config"
	"github.com/catmargiela/fuckhiggsfield/internal/render"
)

func newSettingsCmd(app *App) *cobra.Command {
	c := &cobra.Command{
		Use:     "settings",
		Aliases: []string{"config", "auth"},
		Short:   "Store API keys and preferences",
		Long: "Read and write bara settings.\n\n" +
			"API keys live in credentials.json with 0600 permissions; everything else\n" +
			"lives in config.json. Environment variables (OPENAI_API_KEY, BARA_*)\n" +
			"override the files without writing to disk.",
	}
	c.AddCommand(
		newSettingsListCmd(app),
		newSettingsGetCmd(app),
		newSettingsSetCmd(app),
		newSettingsUnsetCmd(app),
		newSettingsPathCmd(app),
	)
	return c
}

func newSettingsListCmd(app *App) *cobra.Command {
	var reveal bool
	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List every setting and its current value",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := app.Load(); err != nil {
				return err
			}
			type row struct {
				Name   string `json:"name"`
				Value  string `json:"value"`
				Secret bool   `json:"secret"`
				Desc   string `json:"description"`
			}
			var out []row
			var rows [][]string
			for _, s := range config.Settings() {
				v, err := config.Get(app.Cfg, app.Creds, s.Name)
				if err != nil {
					return err
				}
				shown := v
				if s.Secret && !reveal {
					shown = config.Mask(v)
				}
				if shown == "" {
					shown = render.Dim("(unset)")
				}
				out = append(out, row{Name: s.Name, Value: shown, Secret: s.Secret, Desc: s.Desc})
				rows = append(rows, []string{s.Name, shown, s.Desc})
			}
			if app.JSON {
				return render.JSON(app.Out, out)
			}
			render.Table(app.Out, []string{"setting", "value", "description"}, rows)
			return nil
		},
	}
	c.Flags().BoolVar(&reveal, "reveal", false, "print secret values in full instead of masking them")
	return c
}

func newSettingsGetCmd(app *App) *cobra.Command {
	var reveal bool
	c := &cobra.Command{
		Use:     "get <setting>",
		Short:   "Print one setting",
		Args:    cobra.ExactArgs(1),
		Example: "  bara settings get output_dir\n  bara settings get openai_api_key --reveal",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Load(); err != nil {
				return err
			}
			v, err := config.Get(app.Cfg, app.Creds, args[0])
			if err != nil {
				return err
			}
			secret := false
			for _, s := range config.Settings() {
				if s.Name == args[0] {
					secret = s.Secret
				}
			}
			if secret && !reveal {
				v = config.Mask(v)
			}
			if app.JSON {
				return render.JSON(app.Out, map[string]string{"name": args[0], "value": v})
			}
			fmt.Fprintln(app.Out, v)
			return nil
		},
	}
	c.Flags().BoolVar(&reveal, "reveal", false, "print the secret in full")
	return c
}

func newSettingsSetCmd(app *App) *cobra.Command {
	var stdin bool
	c := &cobra.Command{
		Use:   "set <setting> [value]",
		Short: "Write a setting",
		Long: "Write a setting.\n\n" +
			"Omit the value (or pass --stdin) to read it from standard input, which keeps\n" +
			"API keys out of your shell history.",
		Args:    cobra.RangeArgs(1, 2),
		Example: "  bara settings set openai_api_key --stdin\n  bara settings set upload_backend none\n  bara settings set output_dir ~/renders",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Load(); err != nil {
				return err
			}
			name := args[0]
			var value string
			switch {
			case len(args) == 2 && !stdin:
				value = args[1]
			default:
				v, err := readSecret(name)
				if err != nil {
					return err
				}
				value = v
			}
			secret, err := config.Set(app.Cfg, app.Creds, name, value)
			if err != nil {
				return err
			}
			if secret {
				if err := app.Creds.Save(); err != nil {
					return err
				}
			} else if err := app.Cfg.Save(); err != nil {
				return err
			}
			shown := value
			if secret {
				shown = config.Mask(value)
			}
			if app.JSON {
				return render.JSON(app.Out, map[string]any{"name": name, "value": shown, "secret": secret})
			}
			fmt.Fprintf(app.Out, "%s = %s\n", render.Bold(name), shown)
			return nil
		},
	}
	c.Flags().BoolVar(&stdin, "stdin", false, "read the value from standard input")
	return c
}

func newSettingsUnsetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "unset <setting>",
		Short: "Clear a setting",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Load(); err != nil {
				return err
			}
			secret, err := config.Set(app.Cfg, app.Creds, args[0], "")
			if err != nil {
				return err
			}
			if secret {
				if err := app.Creds.Save(); err != nil {
					return err
				}
			} else if err := app.Cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(app.Out, "cleared %s\n", render.Bold(args[0]))
			return nil
		},
	}
}

func newSettingsPathCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print where settings are stored",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			d, err := config.Dir()
			if err != nil {
				return err
			}
			if app.JSON {
				return render.JSON(app.Out, map[string]string{
					"dir":         d,
					"config":      d + "/config.json",
					"credentials": d + "/credentials.json",
					"jobs":        d + "/jobs.json",
				})
			}
			render.KeyValues(app.Out, [][2]string{
				{"Directory", d},
				{"Config", d + "/config.json"},
				{"Credentials", d + "/credentials.json"},
				{"Jobs", d + "/jobs.json"},
			})
			return nil
		},
	}
}

// readSecret prompts on stderr and reads one line from stdin, so the value
// never lands in shell history.
func readSecret(name string) (string, error) {
	if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		fmt.Fprintf(os.Stderr, "Enter value for %s: ", name)
	}
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("read %s from stdin: %w", name, err)
	}
	v := strings.TrimSpace(line)
	if v == "" {
		return "", fmt.Errorf("no value provided for %s", name)
	}
	return v, nil
}
