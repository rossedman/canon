// Command canon serves engineering standards as tools: an MCP server, a CLI
// and a hook binary over one engine.
package main

import "github.com/rossedman/canon/cmd"

// main runs the root command.
func main() {
	cmd.Execute()
}
