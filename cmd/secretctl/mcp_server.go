package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/forest6511/secretctl/internal/mcp"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(mcpServerCmd)
}

// mcpServerCmd starts the MCP server for AI coding assistant integration
var mcpServerCmd = &cobra.Command{
	Use:   "mcp-server",
	Short: "Start the MCP server for AI coding assistant integration",
	Long: `Start the MCP server that provides secure secret access to AI coding assistants.

The server implements the Model Context Protocol (MCP) over stdio transport,
following the Option D+ design where AI agents never receive plaintext secrets.

Available tools:
  - secret_list:       List secret keys with metadata (no values)
  - secret_exists:     Check if a secret exists with metadata
  - secret_get_masked: Get masked secret value (e.g., "****WXYZ")
  - secret_run:        Execute command with secrets as environment variables

Authentication:
  Set SECRETCTL_PASSWORD environment variable before starting the server.
  The password is read once and immediately cleared from the environment.

  SECURITY NOTE: On Linux, the environment variable may briefly be visible
  via /proc/<pid>/environ before it is cleared. For maximum security,
  consider using a secrets manager or setting the variable immediately
  before execution in a subshell.

Policy:
  Create ~/.secretctl/mcp-policy.yaml to configure allowed commands for secret_run.
  Without a policy file, secret_run is disabled (deny-by-default).

Example MCP configuration for Claude Code (~/.claude.json):
  {
    "mcpServers": {
      "secretctl": {
        "type": "stdio",
        "command": "/path/to/secretctl",
        "args": ["mcp-server"],
        "env": {
          "SECRETCTL_PASSWORD": "your-master-password"
        }
      }
    }
  }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMCPServer()
	},
}

func runMCPServer() error {
	// Create server with default options
	server, err := mcp.NewServer(nil)
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
		server.Close()
	}()

	// Run the server
	if err := server.Run(ctx); err != nil {
		// Don't report context canceled as an error
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("MCP server error: %w", err)
	}

	return nil
}
