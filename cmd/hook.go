package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/canon/internal/findings"
	"github.com/rossedman/canon/internal/hook"
)

// hookCmd is the hook front door for Claude Code and git hooks.
var hookCmd = &cobra.Command{
	Use:   "hook [paths...]",
	Short: "Run as a Claude Code or git hook on the files just changed",
	Long: `As a Claude Code PostToolUse hook, canon hook reads the hook payload on
stdin, checks the file the agent edited, and exits 2 with the findings on
stderr so the agent fixes them. Given paths instead, it checks those, which
suits git pre-commit hooks.`,
	Example: `  # .claude/settings.json
  {"hooks": {"PostToolUse": [{"matcher": "Edit|Write|MultiEdit",
    "hooks": [{"type": "command", "command": "canon hook"}]}]}}

  # .git/hooks/pre-commit
  canon hook $(git diff --cached --name-only --diff-filter=ACM)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		min, err := findings.ParseSeverity(viper.GetString("hook.min-severity"))
		if err != nil {
			return err
		}
		paths := args
		root := repoRoot()
		if len(paths) == 0 {
			in, err := hook.ParseInput(cmd.InOrStdin())
			if err != nil {
				return err
			}
			paths = in.Paths()
			if in.Cwd != "" && !cmd.Flags().Changed("root") && os.Getenv("CANON_ROOT") == "" {
				root = in.Cwd
			}
		}
		e, err := loadEngine()
		if err != nil {
			return err
		}
		res, err := hook.Run(cmd.Context(), e, root, paths, min)
		if err != nil {
			return err
		}
		if res.Code != 0 {
			if _, err := fmt.Fprint(cmd.ErrOrStderr(), res.Message); err != nil {
				return fmt.Errorf("write hook output: %w", err)
			}
			return exitError{code: res.Code}
		}
		return nil
	},
}

// init registers the hook command and its flags.
func init() {
	hookCmd.Flags().String("min-severity", "error", "lowest severity that blocks: error, warning or suggestion")
	mustBind("hook.min-severity", hookCmd.Flags().Lookup("min-severity"))
	rootCmd.AddCommand(hookCmd)
}
