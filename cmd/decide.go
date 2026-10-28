package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/canon/internal/engine"
	"github.com/rossedman/canon/internal/ui"
)

// decideCmd answers a question from the records.
var decideCmd = &cobra.Command{
	Use:   "decide <question...>",
	Short: `Answer a "how do we do X" question from decision records`,
	Example: `  canon decide how do we handle errors in Go
  canon decide "which base image for a Go service?" --scope containers`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		e, err := loadEngine()
		if err != nil {
			return err
		}
		res, err := e.Decide(engine.DecideRequest{
			Question: strings.Join(args, " "),
			Scope:    viper.GetStringSlice("decide.scope"),
			Limit:    viper.GetInt("decide.limit"),
		})
		if err != nil {
			return fmt.Errorf("decide: %w", err)
		}
		if viper.GetBool("decide.json") {
			return writeJSON(cmd.OutOrStdout(), res)
		}
		if err := ui.Decide(ui.Writer(cmd.OutOrStdout()), res); err != nil {
			return fmt.Errorf("print decision: %w", err)
		}
		return nil
	},
}

// init registers the decide command and its flags.
func init() {
	f := decideCmd.Flags()
	f.StringSlice("scope", nil, "limit to scopes")
	f.Int("limit", 3, "most records to return")
	f.Bool("json", false, "print the result as JSON")
	for _, n := range []string{"scope", "limit", "json"} {
		mustBind("decide."+n, f.Lookup(n))
	}
	rootCmd.AddCommand(decideCmd)
}
