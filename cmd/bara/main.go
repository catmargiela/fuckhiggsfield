// Command bara generates images with GPT Image 2 from the terminal using your
// own OpenAI API key.
package main

import (
	"os"

	"github.com/catmargiela/fuckhiggsfield/internal/cmd"
)

// Injected at build time via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.Version, cmd.Commit, cmd.Date = version, commit, date
	os.Exit(cmd.Execute())
}
