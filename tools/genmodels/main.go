// Command genmodels renders MODELS.md from the embedded model registry, so the
// documentation cannot drift from what the CLI actually accepts.
//
//	go run ./tools/genmodels > MODELS.md
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/catmargiela/fuckhiggsfield/internal/catalog"
)

func main() {
	c, err := catalog.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "genmodels:", err)
		os.Exit(1)
	}
	var b strings.Builder

	b.WriteString("# Models\n\n")
	b.WriteString("Generated from `internal/catalog/models.json` by `go run ./tools/genmodels`.\n")
	b.WriteString("Do not edit by hand.\n\n")
	b.WriteString("Run `bara model get <model>` for the same schema in your terminal.\n\n")

	kinds := []struct {
		kind  catalog.Kind
		title string
	}{
		{catalog.KindImage, "Image"},
		{catalog.KindVideo, "Video"},
		{catalog.KindAudio, "Audio"},
		{catalog.KindText, "Text"},
	}

	b.WriteString("## Index\n\n")
	for _, k := range kinds {
		models := c.Filter(k.kind, "", "")
		if len(models) == 0 {
			continue
		}
		fmt.Fprintf(&b, "### %s (%d)\n\n", k.title, len(models))
		b.WriteString("| model | provider | name | price |\n|---|---|---|---|\n")
		for _, m := range models {
			fmt.Fprintf(&b, "| [`%s`](#%s) | %s | %s | %s |\n",
				m.ID, anchor(m.ID), m.Provider, m.Name, escape(m.Price))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Reference\n\n")
	for _, k := range kinds {
		for _, m := range c.Filter(k.kind, "", "") {
			fmt.Fprintf(&b, "### %s\n\n", m.ID)
			fmt.Fprintf(&b, "**%s** — %s via %s\n\n", m.Name, m.Kind, m.Provider)
			fmt.Fprintf(&b, "- Endpoint: `%s`\n", m.Endpoint)
			if m.Price != "" {
				fmt.Fprintf(&b, "- Price: %s\n", escape(m.Price))
			}
			b.WriteString("\n")
			if m.Notes != "" {
				fmt.Fprintf(&b, "%s\n\n", m.Notes)
			}
			b.WriteString("| parameter | type | required | default | allowed | description |\n")
			b.WriteString("|---|---|---|---|---|---|\n")
			for _, p := range m.Params {
				req := ""
				if p.Required {
					req = "yes"
				}
				fmt.Fprintf(&b, "| `--%s` | %s | %s | %s | %s | %s |\n",
					p.Name, p.Type, req, escape(def(p)), escape(allowed(p)), escape(p.Desc))
			}
			fmt.Fprintf(&b, "\n```bash\nbara generate create %s --prompt \"...\" --wait\n```\n\n", m.ID)
		}
	}
	fmt.Print(b.String())
}

func def(p catalog.Param) string {
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
	case bool:
		return fmt.Sprintf("%t", v)
	}
	return fmt.Sprintf("%v", p.Default)
}

func allowed(p catalog.Param) string {
	var parts []string
	if len(p.Enum) > 0 {
		q := make([]string, len(p.Enum))
		for i, e := range p.Enum {
			q[i] = "`" + e + "`"
		}
		sort.SliceStable(q, func(i, j int) bool { return false })
		parts = append(parts, strings.Join(q, ", "))
	}
	if p.Min != nil || p.Max != nil {
		lo, hi := "", ""
		if p.Min != nil {
			lo = num(*p.Min)
		}
		if p.Max != nil {
			hi = num(*p.Max)
		}
		parts = append(parts, lo+"–"+hi)
	}
	if p.Media != "" {
		parts = append(parts, string(p.Media)+" path or URL")
	}
	return strings.Join(parts, "; ")
}

func num(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%v", f)
}

// escape neutralizes the pipe character, which would break table cells.
func escape(s string) string { return strings.ReplaceAll(s, "|", "\\|") }

func anchor(id string) string {
	return strings.ToLower(strings.ReplaceAll(id, "_", ""))
}
