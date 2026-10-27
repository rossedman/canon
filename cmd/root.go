// Package cmd wires canon's commands. Each file defines exactly one command;
// the logic lives in internal/.
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/rossedman/canon/internal/engine"
	"github.com/rossedman/canon/internal/ui"
)

// cfgFile is the --config flag.
var cfgFile string

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:   "canon",
	Short: "Engineering standards as tools an agent can call",
	Long: `canon turns engineering opinions into tools: check code against rules,
answer "how do we do X" from decision records, review diffs, and scaffold
projects that already follow the standards. It runs as an MCP server
(canon serve), a CLI, and a Claude Code hook (canon hook).`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// init registers persistent flags and config loading.
func init() {
	cobra.OnInitialize(initConfig)
	pf := rootCmd.PersistentFlags()
	pf.StringVar(&cfgFile, "config", "", "config file (default: $XDG_CONFIG_HOME/canon/config.yaml, ~/.canon.yaml or ./.canon.yaml)")
	pf.String("canon-dir", "", "load the canon from this directory instead of the embedded one")
	pf.String("root", ".", "repository root that paths are relative to")
	pf.String("log-level", "warn", "log level: debug, info, warn or error")
	for _, name := range []string{"canon-dir", "root", "log-level"} {
		mustBind(name, pf.Lookup(name))
	}
}

// mustBind binds a flag to a viper key. A missing flag is a programming
// error, so it panics at startup rather than silently ignoring config.
func mustBind(key string, f *pflag.Flag) {
	if f == nil {
		panic("bind " + key + ": no such flag")
	}
	if err := viper.BindPFlag(key, f); err != nil {
		panic("bind " + key + ": " + err.Error())
	}
}

// initConfig resolves config as flag, then env, then file.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		if dir, err := os.UserConfigDir(); err == nil {
			viper.AddConfigPath(filepath.Join(dir, "canon"))
		}
		if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(home)
		}
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}
	viper.SetEnvPrefix("CANON")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && cfgFile != "" {
			fmt.Fprintf(os.Stderr, "canon: read config: %v\n", err)
		}
	}
	slog.SetDefault(ui.NewLogger(os.Stderr, logLevel(viper.GetString("log-level"))))
}

// logLevel parses a log level name, defaulting to warn.
func logLevel(s string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return slog.LevelWarn
	}
	return l
}

// exitError carries a process exit code without printing an error, for
// commands whose output already says what went wrong.
type exitError struct{ code int }

// Error implements error.
func (e exitError) Error() string { return fmt.Sprintf("exit status %d", e.code) }

// Execute runs the root command through fang and exits with the right code.
func Execute() {
	err := fang.Execute(context.Background(), rootCmd,
		fang.WithVersion(version), fang.WithCommit(commit),
		fang.WithErrorHandler(func(w io.Writer, styles fang.Styles, err error) {
			var ee exitError
			if errors.As(err, &ee) {
				return
			}
			fang.DefaultErrorHandler(w, styles, err)
		}))
	var ee exitError
	switch {
	case errors.As(err, &ee):
		os.Exit(ee.code)
	case err != nil:
		os.Exit(1)
	}
}

// loadEngine loads the canon from --canon-dir or the embedded copy.
func loadEngine() (*engine.Engine, error) {
	if dir := viper.GetString("canon-dir"); dir != "" {
		e, err := engine.LoadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("load canon from %s: %w", dir, err)
		}
		return e, nil
	}
	return engine.Default()
}

// repoRoot returns the --root flag.
func repoRoot() string {
	if r := viper.GetString("root"); r != "" {
		return r
	}
	return "."
}

// writeJSON prints v as indented JSON.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

// Root returns the root command, for generating reference docs.
func Root() *cobra.Command {
	return rootCmd
}
