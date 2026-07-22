package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/catmargiela/fuckhiggsfield/internal/render"
)

func newVersionCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build info",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := map[string]string{
				"version": Version,
				"commit":  Commit,
				"date":    Date,
				"go":      runtime.Version(),
				"os":      runtime.GOOS,
				"arch":    runtime.GOARCH,
			}
			if app.JSON {
				return render.JSON(app.Out, info)
			}
			fmt.Fprintf(app.Out, "bara %s (%s) built %s\n%s %s/%s\n",
				Version, Commit, Date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
			return nil
		},
	}
}
