package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/canon/internal/ui"
)

// lintCmd validates the canon itself.
var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Validate the canon: records, rules, profiles and templates",
	Long: `Lint checks every record has its required sections and valid frontmatter,
every rule ID resolves in both directions, supersedes has no cycles, profiles
name real scopes and records, and every template renders.`,
	Example: `  canon lint
  canon lint --canon-dir ./canon`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		e, err := loadEngine()
		if err != nil {
			return err
		}
		problems := e.Lint()
		if viper.GetBool("lint.json") {
			if err := writeJSON(cmd.OutOrStdout(), problems); err != nil {
				return err
			}
		} else if err := ui.Lint(ui.Writer(cmd.OutOrStdout()), problems, len(e.Records.All())); err != nil {
			return fmt.Errorf("print problems: %w", err)
		}
		if len(problems) > 0 {
			return exitError{code: 1}
		}
		return nil
	},
}

// init registers the lint command and its flags.
func init() {
	lintCmd.Flags().Bool("json", false, "print problems as JSON")
	mustBind("lint.json", lintCmd.Flags().Lookup("json"))
	rootCmd.AddCommand(lintCmd)
}
