package cmd

import (
	"fmt"
	"log/slog"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/canon/internal/mcp"
)

// serveCmd runs the MCP server.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the MCP server over stdio, or streamable HTTP with --http",
	Example: `  claude mcp add canon -- canon serve
  canon serve --http :8080`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		e, err := loadEngine()
		if err != nil {
			return err
		}
		srv, err := mcp.New(e, repoRoot(), version)
		if err != nil {
			return fmt.Errorf("start mcp server: %w", err)
		}
		if addr := viper.GetString("serve.http"); addr != "" {
			slog.Info("serving MCP over streamable HTTP", "addr", addr)
			return mcp.ServeHTTP(cmd.Context(), srv, addr)
		}
		if err := srv.Run(cmd.Context(), &sdk.StdioTransport{}); err != nil {
			return fmt.Errorf("serve stdio: %w", err)
		}
		return nil
	},
}

// init registers the serve command and its flags.
func init() {
	serveCmd.Flags().String("http", "", "listen address for streamable HTTP (e.g. :8080); stdio when empty")
	mustBind("serve.http", serveCmd.Flags().Lookup("http"))
	rootCmd.AddCommand(serveCmd)
}
