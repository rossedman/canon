package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/canon/internal/engine"
	"github.com/rossedman/canon/internal/findings"
	"github.com/rossedman/canon/internal/ui"
)

// checkCmd runs the rules in force against paths.
var checkCmd = &cobra.Command{
	Use:   "check [paths...]",
	Short: "Check files against the canon and report findings",
	Long: `Check runs every rule in force for this repo's profile against the given
files or directories (default: the whole repo). Each finding names its rule,
the record it enforces, and why. Exits 1 when a finding reaches --fail-on.`,
	Example: `  canon check
  canon check --changed-only --sarif > canon.sarif
  canon check cmd/ --scope go --min-severity suggestion`,
	RunE: func(cmd *cobra.Command, args []string) error {
		minSev, err := findings.ParseSeverity(viper.GetString("check.min-severity"))
		if err != nil {
			return fmt.Errorf("--min-severity: %w", err)
		}
		e, err := loadEngine()
		if err != nil {
			return err
		}
		res, err := e.Check(cmd.Context(), engine.CheckRequest{
			Root:        repoRoot(),
			Paths:       args,
			Scope:       viper.GetStringSlice("check.scope"),
			MinSeverity: minSev,
			ChangedOnly: viper.GetBool("check.changed-only"),
			Base:        viper.GetString("check.base"),
			Profile:     viper.GetString("check.profile"),
		})
		if err != nil {
			return fmt.Errorf("check: %w", err)
		}
		out := cmd.OutOrStdout()
		switch {
		case viper.GetBool("check.sarif"):
			if err := writeSARIF(out, e, res.Findings); err != nil {
				return err
			}
		case viper.GetBool("check.json"):
			if err := writeJSON(out, res); err != nil {
				return err
			}
		default:
			if err := ui.Check(ui.Writer(out), res); err != nil {
				return fmt.Errorf("print findings: %w", err)
			}
		}
		return failOn(viper.GetString("check.fail-on"), res.Findings)
	},
}

// init registers the check command and its flags.
func init() {
	f := checkCmd.Flags()
	f.StringSlice("scope", nil, "limit to scopes: go, python, containers, terraform, yaml, ci, oss")
	f.String("min-severity", "warning", "lowest severity to report: error, warning or suggestion")
	f.Bool("changed-only", false, "check only files changed since --base")
	f.String("base", "origin/main", "base branch for --changed-only")
	f.String("profile", "", "use this profile instead of detecting one")
	f.Bool("json", false, "print the result as JSON")
	f.Bool("sarif", false, "print findings as SARIF 2.1.0")
	f.String("fail-on", "error", "exit 1 when a finding is at least this severe: error, warning, suggestion or none")
	for _, n := range []string{"scope", "min-severity", "changed-only", "base", "profile", "json", "sarif", "fail-on"} {
		mustBind("check."+n, f.Lookup(n))
	}
	rootCmd.AddCommand(checkCmd)
}

// writeSARIF prints findings as SARIF with every rule in the canon described.
func writeSARIF(w interface{ Write([]byte) (int, error) }, e *engine.Engine, fs []findings.Finding) error {
	var infos []findings.RuleInfo
	for _, r := range e.Rules.All() {
		info := findings.RuleInfo{ID: r.ID, Record: r.Record, Description: r.Description}
		if rec, ok := e.Records.Get(r.Record); ok {
			info.Why = rec.Why()
		}
		infos = append(infos, info)
	}
	b, err := findings.SARIF(fs, infos, version)
	if err != nil {
		return fmt.Errorf("build sarif: %w", err)
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("write sarif: %w", err)
	}
	return nil
}

// failOn returns an exit error when any finding reaches the threshold.
func failOn(threshold string, fs []findings.Finding) error {
	if threshold == "none" {
		return nil
	}
	minSev, err := findings.ParseSeverity(threshold)
	if err != nil {
		return fmt.Errorf("--fail-on: %w", err)
	}
	if len(findings.Filter(fs, minSev)) > 0 {
		return exitError{code: 1}
	}
	return nil
}
