// Command generated prints the import-path-qualified names of generated Go
// files, one per line, so the coverage gate can leave them out.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/rossedman/canon/tools/internal/gofiles"
)

// module is this repo's module path, the prefix of every coverage line.
const module = "github.com/rossedman/canon"

// main prints generated files under the packages named on the command line.
func main() {
	files, err := gofiles.Walk(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for _, line := range generated(files) {
		fmt.Println(line)
	}
}

// generated returns module-qualified paths of generated files. It always
// prints at least a sentinel line, because grep -f with an empty pattern
// file would drop every line.
func generated(files []gofiles.File) []string {
	out := []string{module + "/__no_generated_files__"}
	for _, f := range files {
		if f.Generated {
			out = append(out, module+"/"+strings.TrimPrefix(f.Path, "./"))
		}
	}
	return out
}
