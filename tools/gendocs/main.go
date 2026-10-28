// Command gendocs writes the generated reference pages: one page per CLI
// command from cobra, and one page per decision record from the canon, so
// the docs never drift from the binary or the records.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rossedman/canon/cmd"
	"github.com/rossedman/canon/internal/docgen"
	"github.com/rossedman/canon/internal/engine"
)

// main generates docs into -out (default docs/reference).
func main() {
	out := flag.String("out", "docs/reference", "directory to write reference pages into")
	flag.Parse()
	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run writes the command and record reference pages under out.
func run(out string) error {
	e, err := engine.Default()
	if err != nil {
		return fmt.Errorf("load canon: %w", err)
	}
	if err := docgen.Commands(cmd.Root(), out+"/commands"); err != nil {
		return fmt.Errorf("command docs: %w", err)
	}
	if err := docgen.Records(e, out+"/records"); err != nil {
		return fmt.Errorf("record docs: %w", err)
	}
	return nil
}
