package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// Run command flags
var (
	runKeys       []string
	runTimeout    time.Duration
	runNoSanitize bool
	runEnvPrefix  string
)

// Exit codes per requirements-ja.md §1.3
const (
	ExitSecretNotFound = 2
	ExitTimeout        = 124
	ExitCommandNotFound = 127
	ExitSignalBase     = 128
)

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringArrayVarP(&runKeys, "key", "k", nil, "Secret keys to inject (glob pattern supported)")
	runCmd.Flags().DurationVarP(&runTimeout, "timeout", "t", 5*time.Minute, "Command timeout")
	runCmd.Flags().BoolVar(&runNoSanitize, "no-sanitize", false, "Disable output sanitization")
	runCmd.Flags().StringVar(&runEnvPrefix, "env-prefix", "", "Environment variable name prefix")

	runCmd.MarkFlagRequired("key")
}

// runCmd executes a command with secrets injected as environment variables
var runCmd = &cobra.Command{
	Use:   "run [flags] -- command [args...]",
	Short: "Run a command with secrets as environment variables",
	Long: `Run a command with specified secrets injected as environment variables.

Secrets are converted to environment variable names using these rules:
  - '/' is replaced with '_'
  - '-' is replaced with '_'
  - Names are converted to UPPERCASE

Examples:
  secretctl run -k API_KEY -- curl https://api.example.com
  secretctl run -k DB_HOST -k DB_USER -k DB_PASS -- psql
  secretctl run -k "aws/prod/*" -- aws s3 ls
  secretctl run -k API_KEY --timeout=30s -- ./script.sh`,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Find the command after "--"
		dashIndex := cmd.ArgsLenAtDash()
		if dashIndex == -1 || dashIndex >= len(args) {
			return fmt.Errorf("no command specified; use: secretctl run -k KEY -- command [args...]")
		}

		commandArgs := args[dashIndex:]
		if len(commandArgs) == 0 {
			return fmt.Errorf("no command specified after '--'")
		}

		return executeRun(commandArgs)
	},
}

// executeRun performs the main run command logic
func executeRun(commandArgs []string) error {
	// 1. Unlock vault
	if err := ensureUnlocked(); err != nil {
		return err
	}
	defer v.Lock()

	// 2. Expand key patterns and collect secrets
	secrets, err := collectSecrets(runKeys)
	if err != nil {
		return err
	}

	if len(secrets) == 0 {
		return fmt.Errorf("no secrets matched the specified patterns")
	}

	// 3. Build environment variables
	env, err := buildEnvironment(secrets)
	if err != nil {
		return err
	}

	// 4. Execute command with timeout
	return executeCommand(commandArgs, env, secrets)
}

// secretData holds a secret key and its decrypted value
type secretData struct {
	key   string
	value []byte
}

// collectSecrets expands patterns and fetches secret values
func collectSecrets(patterns []string) ([]secretData, error) {
	// Get all available keys for pattern matching
	allKeys, err := v.ListSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	// Expand patterns and collect unique keys
	seen := make(map[string]bool)
	var matchedKeys []string

	for _, pattern := range patterns {
		matches, err := expandPattern(pattern, allKeys)
		if err != nil {
			return nil, err
		}

		for _, key := range matches {
			if !seen[key] {
				seen[key] = true
				matchedKeys = append(matchedKeys, key)
			}
		}
	}

	if len(matchedKeys) == 0 {
		return nil, fmt.Errorf("no secrets match the specified patterns")
	}

	// Fetch secret values
	var secrets []secretData
	for _, key := range matchedKeys {
		entry, err := v.GetSecret(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret '%s': %w", key, err)
		}

		// Check if secret is expired
		if entry.ExpiresAt != nil && entry.ExpiresAt.Before(time.Now()) {
			return nil, fmt.Errorf("secret '%s' has expired", key)
		}

		secrets = append(secrets, secretData{
			key:   key,
			value: entry.Value,
		})
	}

	return secrets, nil
}

// expandPattern expands a glob pattern against available keys
func expandPattern(pattern string, availableKeys []string) ([]string, error) {
	// Validate pattern syntax
	if _, err := filepath.Match(pattern, ""); err != nil {
		return nil, fmt.Errorf("invalid pattern '%s': %w", pattern, err)
	}

	// Check if pattern contains glob characters
	hasGlob := strings.ContainsAny(pattern, "*?[")

	if !hasGlob {
		// Exact match - verify key exists
		for _, key := range availableKeys {
			if key == pattern {
				return []string{pattern}, nil
			}
		}
		return nil, fmt.Errorf("secret '%s' not found", pattern)
	}

	// Glob matching
	var matches []string
	for _, key := range availableKeys {
		matched, err := filepath.Match(pattern, key)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, key)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no secrets match pattern '%s'", pattern)
	}

	return matches, nil
}

// buildEnvironment creates environment variables from secrets
func buildEnvironment(secrets []secretData) ([]string, error) {
	// Start with current environment
	env := os.Environ()

	// Add secrets as environment variables
	for _, secret := range secrets {
		envName := keyToEnvName(secret.key)

		// Apply prefix if specified
		if runEnvPrefix != "" {
			envName = runEnvPrefix + envName
		}

		// Validate environment variable name
		if err := validateEnvName(envName); err != nil {
			return nil, fmt.Errorf("invalid environment variable name for key '%s': %w", secret.key, err)
		}

		// Check for NUL bytes in value
		if err := validateNoNulBytes(envName, secret.value); err != nil {
			return nil, err
		}

		// Warn if overwriting reserved variables
		warnReservedEnvVar(envName)

		env = append(env, fmt.Sprintf("%s=%s", envName, string(secret.value)))
	}

	return env, nil
}

// keyToEnvName converts a secret key to an environment variable name
// per requirements-ja.md §6.3: / -> _, - -> _, UPPERCASE
func keyToEnvName(key string) string {
	name := strings.ReplaceAll(key, "/", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return strings.ToUpper(name)
}

// validateEnvName validates that a name is a valid POSIX environment variable name
// Pattern: ^[A-Za-z_][A-Za-z0-9_]*$
func validateEnvName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("environment variable name cannot be empty")
	}

	// First character: A-Z, a-z, _
	first := name[0]
	if !((first >= 'A' && first <= 'Z') ||
		(first >= 'a' && first <= 'z') || first == '_') {
		return fmt.Errorf("must start with a letter or underscore")
	}

	// Subsequent characters: A-Z, a-z, 0-9, _
	for i := 1; i < len(name); i++ {
		c := name[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '_') {
			return fmt.Errorf("contains invalid character '%c'", c)
		}
	}

	return nil
}

// validateNoNulBytes checks for NUL bytes which are security risks
func validateNoNulBytes(name string, value []byte) error {
	if strings.ContainsRune(name, '\x00') {
		return fmt.Errorf("NUL byte in environment variable name: %q", name)
	}
	if bytes.ContainsRune(value, '\x00') {
		return fmt.Errorf("NUL byte in secret value for: %q", name)
	}
	return nil
}

// reservedEnvVars are variables that should not be overwritten
var reservedEnvVars = map[string]bool{
	"PATH": true, "HOME": true, "USER": true, "SHELL": true,
	"PWD": true, "OLDPWD": true, "TERM": true, "LANG": true,
	"IFS": true, "PS1": true, "PS2": true,
}

// warnReservedEnvVar prints a warning if overwriting a reserved variable
func warnReservedEnvVar(name string) {
	if reservedEnvVars[name] {
		fmt.Fprintf(os.Stderr, "warning: overwriting reserved environment variable: %s\n", name)
	}
	// Also warn for LC_* variables
	if strings.HasPrefix(name, "LC_") {
		fmt.Fprintf(os.Stderr, "warning: overwriting locale environment variable: %s\n", name)
	}
}

// executeCommand runs the command with secrets in environment
func executeCommand(args []string, env []string, secrets []secretData) error {
	// Prevent core dumps to protect secrets in memory
	if err := disableCoreDumps(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to disable core dumps: %v\n", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	// Look up the command
	cmdPath, err := exec.LookPath(args[0])
	if err != nil {
		return &exitError{code: ExitCommandNotFound, err: fmt.Errorf("command not found: %s", args[0])}
	}

	// Create command
	cmd := exec.CommandContext(ctx, cmdPath, args[1:]...)
	cmd.Env = env

	// Set up graceful shutdown
	cmd.Cancel = func() error {
		// Send SIGTERM first for graceful shutdown
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	cmd.WaitDelay = 5 * time.Second // Wait 5s after SIGTERM before SIGKILL

	// Set up I/O
	if runNoSanitize {
		// Direct passthrough
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// Capture and sanitize output
		cmd.Stdin = os.Stdin

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdout pipe: %w", err)
		}

		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("failed to create stderr pipe: %w", err)
		}

		// Build sanitizer
		sanitizer := newOutputSanitizer(secrets)

		// Start output copying goroutines
		go sanitizer.copy(os.Stdout, stdoutPipe)
		go sanitizer.copy(os.Stderr, stderrPipe)
	}

	// Set up signal forwarding
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigChan)

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Forward signals to child process
	done := make(chan struct{})
	go func() {
		for {
			select {
			case sig := <-sigChan:
				if cmd.Process != nil {
					cmd.Process.Signal(sig)
				}
			case <-done:
				return
			}
		}
	}()

	// Wait for command to complete
	err = cmd.Wait()
	close(done)

	// Handle exit status
	if err != nil {
		// Check if it was a timeout
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return &exitError{code: ExitTimeout, err: fmt.Errorf("command timed out after %v", runTimeout)}
		}

		// Check for exit error
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return &exitError{code: exitErr.ExitCode(), err: nil}
		}

		return err
	}

	return nil
}

// disableCoreDumps sets RLIMIT_CORE to 0 to prevent core dumps
func disableCoreDumps() error {
	var rLimit syscall.Rlimit
	rLimit.Cur = 0
	rLimit.Max = 0
	return syscall.Setrlimit(syscall.RLIMIT_CORE, &rLimit)
}

// exitError represents a command exit with a specific code
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return fmt.Sprintf("exit status %d", e.code)
}

func (e *exitError) ExitCode() int {
	return e.code
}

// outputSanitizer sanitizes output by replacing secret values
type outputSanitizer struct {
	secrets []secretData
}

func newOutputSanitizer(secrets []secretData) *outputSanitizer {
	return &outputSanitizer{secrets: secrets}
}

// copy reads from src, sanitizes, and writes to dst
func (s *outputSanitizer) copy(dst io.Writer, src io.Reader) {
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := src.Read(buf)
		if n > 0 {
			data := buf[:n]

			// Skip sanitization for binary output (contains NUL)
			if !bytes.Contains(data, []byte{0x00}) {
				data = s.sanitize(data)
			}

			dst.Write(data)
		}
		if err != nil {
			break
		}
	}
}

// sanitize replaces secret values with [REDACTED:key]
func (s *outputSanitizer) sanitize(data []byte) []byte {
	result := data
	for _, secret := range s.secrets {
		// Only sanitize values >= 4 bytes to avoid false positives
		if len(secret.value) >= 4 {
			placeholder := []byte(fmt.Sprintf("[REDACTED:%s]", keyToEnvName(secret.key)))
			result = bytes.ReplaceAll(result, secret.value, placeholder)
		}
	}
	return result
}
