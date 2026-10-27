package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/canon/internal/engine"
	"github.com/rossedman/canon/internal/ui"
)

// scaffoldCmd generates a project or component from a template.
var scaffoldCmd = &cobra.Command{
	Use:   "scaffold <kind> <name>",
	Short: "Generate a project or component that already follows the canon",
	Long: `Scaffold renders a template from the canon. Kinds: go-cli, go-service,
py-service, tf-module, dockerfile and gha-workflow. Without --write it lists
the files it would create. With --write it creates them (never overwriting)
and checks the result.`,
	Example: `  canon scaffold go-cli hello
  canon scaffold go-cli hello --var module=github.com/me/hello --write
  canon scaffold dockerfile api --dest services/api --write`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		e, err := loadEngine()
		if err != nil {
			return err
		}
		res, err := e.Scaffold(cmd.Context(), engine.ScaffoldRequest{
			Root:   repoRoot(),
			Kind:   args[0],
			Name:   args[1],
			Dest:   viper.GetString("scaffold.dest"),
			Vars:   viper.GetStringMapString("scaffold.var"),
			DryRun: !viper.GetBool("scaffold.write"),
		})
		if err != nil {
			return err
		}
		if viper.GetBool("scaffold.json") {
			if err := writeJSON(cmd.OutOrStdout(), res); err != nil {
				return err
			}
		} else if err := ui.Scaffold(ui.Writer(cmd.OutOrStdout()), res); err != nil {
			return fmt.Errorf("print scaffold: %w", err)
		}
		if !res.Passed {
			return exitError{code: 1}
		}
		return nil
	},
}

// init registers the scaffold command and its flags.
func init() {
	f := scaffoldCmd.Flags()
	f.String("dest", ".", "directory to write into, relative to --root")
	f.StringToString("var", nil, "template variable as key=value (repeatable)")
	f.Bool("write", false, "write the files instead of listing them")
	f.Bool("json", false, "print the result as JSON")
	for _, n := range []string{"dest", "var", "write", "json"} {
		mustBind("scaffold."+n, f.Lookup(n))
	}
	rootCmd.AddCommand(scaffoldCmd)
}
