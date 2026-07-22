// Command bara generates images, video, audio, and text from the terminal
// using your own OpenAI and RunPod API keys.
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
