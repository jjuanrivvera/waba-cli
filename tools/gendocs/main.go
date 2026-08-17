// Command gendocs regenerates docs/commands from the live cobra tree.
//
// The command reference is generated rather than written by hand so it can never drift from
// the binary: CI runs this and fails the build if the committed docs differ.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra/doc"

	"github.com/jjuanrivvera/waba-cli/commands"
)

func main() {
	out := "docs/commands"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	n, err := generate(out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gendocs: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("gendocs: wrote %d pages to %s\n", n, out)
}

// generate writes the command reference and returns the page count.
func generate(out string) (int, error) {
	if err := os.MkdirAll(out, 0o750); err != nil { // #nosec G703 -- output dir named by the operator
		return 0, err
	}

	root := commands.NewRootCmd()
	// Without this cobra stamps a generation date into every page, which would make CI's
	// drift check fail on every run whether or not anything actually changed.
	root.DisableAutoGenTag = true

	if err := doc.GenMarkdownTree(root, out); err != nil {
		return 0, err
	}
	return fixLinks(out)
}

// fixLinks strips the ".md" from cobra's cross-references.
//
// MkDocs resolves links relative to the rendered page, so a link to "atlassian_issues.md"
// 404s on the built site; "atlassian_issues" resolves correctly. A --strict site build fails
// on the dangling links, so this is not cosmetic.
func fixLinks(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(path) // #nosec G304,G703 -- the generator reads back its own output
		if err != nil {
			return count, err
		}
		fixed := strings.ReplaceAll(string(raw), ".md)", ")")
		if err := os.WriteFile(path, []byte(fixed), 0o600); err != nil { // #nosec G703 -- rewrites the file it just generated
			return count, err
		}
		count++
	}
	return count, nil
}
