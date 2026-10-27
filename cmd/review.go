package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/canon/internal/engine"
	"github.com/rossedman/canon/internal/findings"
	"github.com/rossedman/canon/internal/ui"
)

// reviewCmd grades a diff against the records that apply.
var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review a change: findings on changed files plus concerns linters miss",
	Long: `Review checks the files a change touches and classifies the change for
concerns no linter reports: new dependencies, new Dockerfiles, Terraform
provider bumps, exported API changes and new packages without tests. The diff
is computed against --base, or read from --diff-file (- for stdin).`,
	Example: `  canon review
  git diff main | canon review --diff-file -`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		min, err := findings.ParseSeverity(viper.GetString("review.min-severity"))
		if err != nil {
			return err
		}
		diff, err := readDiff(cmd.InOrStdin(), viper.GetString("review.diff-file"))
		if err != nil {
			return err
		}
		e, err := loadEngine()
		if err != nil {
			return err
		}
		res, err := e.Review(cmd.Context(), engine.ReviewRequest{Root: repoRoot(), Base: viper.GetString("review.base"), Diff: diff, MinSeverity: min})
		if err != nil {
			return err
		}
		if viper.GetBool("review.json") {
			if err := writeJSON(cmd.OutOrStdout(), res); err != nil {
				return err
			}
		} else if err := ui.Review(ui.Writer(cmd.OutOrStdout()), res); err != nil {
			return fmt.Errorf("print review: %w", err)
		}
		return failOn(viper.GetString("review.fail-on"), res.Findings)
	},
}

// init registers the review command and its flags.
func init() {
	f := reviewCmd.Flags()
	f.String("base", "origin/main", "branch to compare the working tree against")
	f.String("diff-file", "", "read a unified diff from this file, or - for stdin")
	f.String("min-severity", "warning", "lowest severity to report")
	f.String("fail-on", "none", "exit 1 when a finding is at least this severe: error, warning, suggestion or none")
	f.Bool("json", false, "print the result as JSON")
	for _, n := range []string{"base", "diff-file", "min-severity", "fail-on", "json"} {
		mustBind("review."+n, f.Lookup(n))
	}
	rootCmd.AddCommand(reviewCmd)
}

// readDiff reads a diff from a file, from stdin for "-", or returns "" so the
// engine computes one.
func readDiff(stdin io.Reader, name string) (string, error) {
	var (
		data []byte
		err  error
	)
	switch name {
	case "":
		return "", nil
	case "-":
		data, err = io.ReadAll(stdin)
	default:
		data, err = os.ReadFile(name) //nolint:gosec // the user names the diff file to read
	}
	if err != nil {
		return "", fmt.Errorf("read diff: %w", err)
	}
	return string(data), nil
}
