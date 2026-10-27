package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/canon/internal/ui"
)

// profileCmd shows which profile and records apply to a repo.
var profileCmd = &cobra.Command{
	Use:   "profile [path]",
	Short: "Detect the repo's profile, the records in force and missing checkers",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		e, err := loadEngine()
		if err != nil {
			return err
		}
		p := repoRoot()
		if len(args) == 1 {
			p = args[0]
			if !filepath.IsAbs(p) {
				p = filepath.Join(repoRoot(), p)
			}
		}
		res, err := e.Profile(p)
		if err != nil {
			return err
		}
		if viper.GetBool("profile.json") {
			return writeJSON(cmd.OutOrStdout(), res)
		}
		if err := ui.Profile(ui.Writer(cmd.OutOrStdout()), res); err != nil {
			return fmt.Errorf("print profile: %w", err)
		}
		return nil
	},
}

// init registers the profile command and its flags.
func init() {
	profileCmd.Flags().Bool("json", false, "print the result as JSON")
	mustBind("profile.json", profileCmd.Flags().Lookup("json"))
	rootCmd.AddCommand(profileCmd)
}
