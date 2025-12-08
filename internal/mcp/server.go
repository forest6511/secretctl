// Package mcp implements the MCP (Model Context Protocol) server for secretctl.
// This implements the Option D+ design where AI agents never receive plaintext secrets.
package mcp

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/forest6511/secretctl/pkg/vault"
)

// maxConcurrentRuns is the maximum number of concurrent secret_run executions per §6.4
const maxConcurrentRuns = 5

// Server represents the MCP server for secretctl.
type Server struct {
	server    *mcp.Server
	vault     *vault.Vault
	vaultPath string
	policy    *Policy
	runSem    chan struct{} // Semaphore for limiting concurrent secret_run operations
}

// ServerOptions contains configuration options for the MCP server.
type ServerOptions struct {
	// VaultPath is the path to the vault directory.
	// If empty, defaults to ~/.secretctl
	VaultPath string

	// Password is the master password for the vault.
	// If empty, the server will attempt to read from SECRETCTL_PASSWORD environment variable.
	Password string
}

// NewServer creates a new MCP server instance.
func NewServer(opts *ServerOptions) (*Server, error) {
	if opts == nil {
		opts = &ServerOptions{}
	}

	// Determine vault path
	vaultPath := opts.VaultPath
	if vaultPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		vaultPath = filepath.Join(home, ".secretctl")
	}

	// Load policy
	policy, err := LoadPolicy(vaultPath)
	if err != nil {
		// Policy load failure is not fatal - we'll operate in restricted mode
		log.Printf("warning: failed to load MCP policy: %v", err)
		policy = nil
	}

	// Create vault instance
	v := vault.New(vaultPath)

	// Get password from options or environment
	password := opts.Password
	if password == "" {
		password = os.Getenv("SECRETCTL_PASSWORD")
		// Clear the environment variable after reading for security
		os.Unsetenv("SECRETCTL_PASSWORD")
	}

	if password == "" {
		return nil, fmt.Errorf("no password provided: set SECRETCTL_PASSWORD environment variable")
	}

	// Unlock the vault
	if err := v.Unlock(password); err != nil {
		return nil, fmt.Errorf("failed to unlock vault: %w", err)
	}

	// Create the MCP server
	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "secretctl",
			Version: "0.5.0",
		},
		nil,
	)

	s := &Server{
		server:    mcpServer,
		vault:     v,
		vaultPath: vaultPath,
		policy:    policy,
		runSem:    make(chan struct{}, maxConcurrentRuns),
	}

	// Register tools
	s.registerTools()

	return s, nil
}

// registerTools registers all MCP tools with the server.
func (s *Server) registerTools() {
	// secret_list - List secret keys with metadata (no values)
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "secret_list",
		Description: "List all secret keys with metadata. Returns key names, tags, expiration, and flags for notes/url presence. Does NOT return secret values.",
	}, s.handleSecretList)

	// secret_exists - Check if a secret exists and return metadata
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "secret_exists",
		Description: "Check if a secret key exists and return its metadata. Does NOT return the secret value.",
	}, s.handleSecretExists)

	// secret_get_masked - Get masked secret value
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "secret_get_masked",
		Description: "Get a masked version of a secret value (e.g., '****WXYZ'). Useful for verifying secret format without exposing the actual value.",
	}, s.handleSecretGetMasked)

	// secret_run - Execute command with secrets as environment variables
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "secret_run",
		Description: "Execute a command with specified secrets injected as environment variables. Output is automatically sanitized to prevent secret leakage. Requires policy approval.",
	}, s.handleSecretRun)
}

// Run starts the MCP server using stdio transport.
func (s *Server) Run(ctx context.Context) error {
	defer s.vault.Lock()

	return s.server.Run(ctx, &mcp.StdioTransport{})
}

// Close closes the server and locks the vault.
func (s *Server) Close() error {
	s.vault.Lock()
	return nil
}
