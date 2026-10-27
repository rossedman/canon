// Package cmd wires the fixture's commands.
package cmd

import "github.com/spf13/cobra"

// rootCmd is the base command.
var rootCmd = &cobra.Command{Use: "good"}

// Execute runs the root command.
func Execute() {
	_ = rootCmd.Execute()
}
