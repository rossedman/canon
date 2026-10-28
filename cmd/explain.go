package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/canon/internal/ui"
)

// explainCmd traces a rule or record to its reasoning.
var explainCmd = &cobra.Command{
	Use:   "explain <rule-or-record>",
	Short: "Show the record behind a rule or finding, and what enforces it",
	Example: `  canon explain canon-final-stage-base
  canon explain go-errors-wrap`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		e, err := loadEngine()
		if err != nil {
			return err
		}
		res, err := e.Explain(e.Resolve(args[0]))
		if err != nil {
			return fmt.Errorf("explain: %w", err)
		}
		if viper.GetBool("explain.json") {
			return writeJSON(cmd.OutOrStdout(), res)
		}
		if err := ui.Explain(ui.Writer(cmd.OutOrStdout()), res); err != nil {
			return fmt.Errorf("print explanation: %w", err)
		}
		return nil
	},
}

// init registers the explain command and its flags.
func init() {
	explainCmd.Flags().Bool("json", false, "print the result as JSON")
	mustBind("explain.json", explainCmd.Flags().Lookup("json"))
	rootCmd.AddCommand(explainCmd)
}
