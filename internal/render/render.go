// Package render prints human tables and machine JSON.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// NoColor disables ANSI styling. Set from --no-color, NO_COLOR, or a non-TTY.
var NoColor bool

func init() {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("BARA_NO_COLOR") != "" {
		NoColor = true
	}
}

const (
	ansiReset = "\033[0m"
	ansiBold  = "\033[1m"
	ansiDim   = "\033[2m"
	ansiRed   = "\033[31m"
	ansiGreen = "\033[32m"
	ansiAmber = "\033[33m"
	ansiCyan  = "\033[36m"
)

func paint(code, s string) string {
	if NoColor || s == "" {
		return s
	}
	return code + s + ansiReset
}

// Bold styles a string as bold.
func Bold(s string) string { return paint(ansiBold, s) }

// Dim styles a string as dim.
func Dim(s string) string { return paint(ansiDim, s) }

// Cyan styles a string cyan.
func Cyan(s string) string { return paint(ansiCyan, s) }

// StatusColor tints a job status by outcome.
func StatusColor(status string) string {
	switch strings.ToUpper(status) {
	case "COMPLETED":
		return paint(ansiGreen, status)
	case "FAILED", "CANCELLED", "TIMED_OUT":
		return paint(ansiRed, status)
	case "IN_PROGRESS", "QUEUED":
		return paint(ansiAmber, status)
	}
	return status
}

// JSON writes v as indented JSON.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// Table writes an aligned table. Empty rows print a placeholder line.
func Table(w io.Writer, headers []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Fprintln(w, Dim("(none)"))
		return
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	if len(headers) > 0 {
		styled := make([]string, len(headers))
		for i, h := range headers {
			styled[i] = Bold(strings.ToUpper(h))
		}
		fmt.Fprintln(tw, strings.Join(styled, "\t"))
	}
	for _, r := range rows {
		fmt.Fprintln(tw, strings.Join(r, "\t"))
	}
	tw.Flush()
}

// KeyValues writes an aligned key/value block.
func KeyValues(w io.Writer, pairs [][2]string) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	for _, p := range pairs {
		if p[1] == "" {
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\n", Bold(p[0]), p[1])
	}
	tw.Flush()
}

// Truncate shortens s to n runes, adding an ellipsis.
func Truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
