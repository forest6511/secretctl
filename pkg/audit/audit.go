// Package audit provides audit logging with HMAC chain for tamper detection.
// Implements requirements-ja.md §7 and security-design-ja.md §8.
package audit

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/hkdf"
)

// Disk space constants
const (
	MinAuditDiskSpace = 1024 * 1024 // 1 MB minimum for audit logs
)

// Operation types for audit logging
const (
	// Vault operations
	OpVaultInit         = "vault.init"
	OpVaultUnlock       = "vault.unlock"
	OpVaultUnlockFailed = "vault.unlock_failed"
	OpVaultLock         = "vault.lock"

	// Secret operations
	OpSecretGet    = "secret.get"
	OpSecretSet    = "secret.set"
	OpSecretUpdate = "secret.update"
	OpSecretDelete = "secret.delete"
	OpSecretList   = "secret.list"

	// MCP operations (Phase 2)
	OpSecretExists    = "secret.exists"
	OpSecretGetMasked = "secret.get_masked"
	OpSecretRun       = "secret.run"
	OpSecretRunDenied = "secret.run_denied"
	OpSecretExport    = "secret.export"

	// Session operations (Phase 3+)
	OpSessionStart = "session.start"
	OpSessionEnd   = "session.end"
)

// Source identifies where the operation originated
const (
	SourceCLI = "cli"
	SourceMCP = "mcp"
	SourceUI  = "ui"
	SourceAPI = "api"
)

// Result indicates the outcome of an operation
const (
	ResultSuccess = "success"
	ResultError   = "error"
	ResultDenied  = "denied"
)

// AuditEvent represents a single audit log record (requirements-ja.md §7.3)
type AuditEvent struct {
	// Basic information
	Version   int    `json:"v"`  // Schema version (1)
	ID        string `json:"id"` // Event ID (ULID)
	Timestamp string `json:"ts"` // RFC 3339 nanosecond precision

	// Operation information
	Operation string `json:"op"`                 // Operation type
	Key       string `json:"key,omitempty"`      // Key name (if applicable)
	KeyHMAC   string `json:"key_hmac,omitempty"` // Key name HMAC (optional)

	// Actor information
	Actor Actor `json:"actor"`

	// Organization information (Phase 3+)
	Org *Org `json:"org,omitempty"`

	// Result
	Result string     `json:"result"`          // success | error | denied
	Error  *ErrorInfo `json:"error,omitempty"` // Error details

	// Context (operation-dependent)
	Context map[string]interface{} `json:"ctx,omitempty"`

	// Tamper detection
	Chain Chain `json:"chain"`
}

// Actor represents who performed the operation
type Actor struct {
	Type      string `json:"type"`                 // user | service | system
	ID        string `json:"id,omitempty"`         // User ID (Phase 3+)
	Email     string `json:"email,omitempty"`      // Email (Phase 3+)
	Source    string `json:"source"`               // cli | mcp | ui | api
	ClientID  string `json:"client_id,omitempty"`  // Client identifier
	SessionID string `json:"session_id"`           // Session ID
	IP        string `json:"ip,omitempty"`         // IP address (Phase 3+)
	UserAgent string `json:"user_agent,omitempty"` // User-Agent (Phase 3+)
}

// Org represents organization context (Phase 3+)
type Org struct {
	ID      string `json:"id,omitempty"`       // Organization ID
	VaultID string `json:"vault_id,omitempty"` // Vault ID
	TeamID  string `json:"team_id,omitempty"`  // Team ID
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// Chain provides HMAC chain for tamper detection
type Chain struct {
	Sequence int64  `json:"seq"`  // Sequence number
	PrevHash string `json:"prev"` // Previous record hash
	HMAC     string `json:"hmac"` // This record's HMAC
}

// Logger handles audit log writing with HMAC chain
type Logger struct {
	path       string     // Audit log directory path
	hmacKey    []byte     // HMAC key derived from master key
	mu         sync.Mutex // Protects concurrent writes
	sequence   int64      // Current sequence number
	prevHash   string     // Previous record hash
	sessionID  string     // Current session ID
	hmacKeySet bool       // Whether HMAC key has been set
}

// Config holds audit logger configuration
type Config struct {
	Path            string // Directory for audit logs (default: ~/.secretctl/audit)
	HMACKeyNames    bool   // Whether to HMAC key names
	IncludeContext  bool   // Whether to include context information
	RetentionMonths int    // Retention period in months (default: 12)
}

// NewLogger creates a new audit logger
func NewLogger(path string) *Logger {
	return &Logger{
		path:      path,
		prevHash:  "genesis", // Initial chain value
		sessionID: generateSessionID(),
	}
}

// SetHMACKey derives and sets the HMAC key from the master key using HKDF
func (l *Logger) SetHMACKey(masterKey []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Derive HMAC key using HKDF-SHA256
	hkdfReader := hkdf.New(sha256.New, masterKey, nil, []byte("audit-log-v1"))
	l.hmacKey = make([]byte, 32)
	if _, err := hkdfReader.Read(l.hmacKey); err != nil {
		return fmt.Errorf("audit: failed to derive HMAC key: %w", err)
	}
	l.hmacKeySet = true

	// Load existing chain state
	if err := l.loadChainState(); err != nil {
		// Not a fatal error - may be first run
		l.sequence = 0
		l.prevHash = "genesis"
	}

	return nil
}

// Log records an audit event
func (l *Logger) Log(op, source, result string, keyName string, errInfo *ErrorInfo, ctx map[string]interface{}) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.hmacKeySet {
		return fmt.Errorf("audit: HMAC key not set")
	}

	// Ensure directory exists
	if err := os.MkdirAll(l.path, 0700); err != nil {
		return fmt.Errorf("audit: failed to create directory: %w", err)
	}

	// Check disk space before write (per Codex review)
	if err := l.checkDiskSpace(); err != nil {
		return err
	}

	// Build event
	event := AuditEvent{
		Version:   1,
		ID:        generateULID(),
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Operation: op,
		Actor: Actor{
			Type:      "user",
			Source:    source,
			SessionID: l.sessionID,
		},
		Result:  result,
		Error:   errInfo,
		Context: ctx,
	}

	// Add key HMAC if key name provided (using HMAC instead of SHA-256 per Codex review)
	if keyName != "" {
		mac := hmac.New(sha256.New, l.hmacKey)
		mac.Write([]byte(keyName))
		event.Key = hex.EncodeToString(mac.Sum(nil))
		event.KeyHMAC = event.Key // Also set KeyHMAC field for clarity
	}

	// Build chain
	l.sequence++
	event.Chain.Sequence = l.sequence
	event.Chain.PrevHash = l.prevHash

	// Calculate HMAC for this record
	recordData := l.buildRecordData(&event)
	mac := hmac.New(sha256.New, l.hmacKey)
	mac.Write(recordData)
	event.Chain.HMAC = hex.EncodeToString(mac.Sum(nil))

	// Update previous hash for next record
	l.prevHash = event.Chain.HMAC

	// Write to file
	if err := l.writeEvent(&event); err != nil {
		return err
	}

	// Save chain state
	return l.saveChainState()
}

// LogSuccess is a convenience method for successful operations
func (l *Logger) LogSuccess(op, source, keyName string) error {
	return l.Log(op, source, ResultSuccess, keyName, nil, nil)
}

// LogError is a convenience method for failed operations
func (l *Logger) LogError(op, source, keyName string, errCode, errMsg string) error {
	return l.Log(op, source, ResultError, keyName, &ErrorInfo{Code: errCode, Message: errMsg}, nil)
}

// LogDenied is a convenience method for denied operations
func (l *Logger) LogDenied(op, source, keyName string, reason string) error {
	return l.Log(op, source, ResultDenied, keyName, nil, map[string]interface{}{"reason": reason})
}

// buildRecordData creates the data to be HMACed
// Per Codex review: includes ALL significant fields (Actor, Error, Context)
func (l *Logger) buildRecordData(event *AuditEvent) []byte {
	// Build Actor fields
	actorData := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		event.Actor.Type,
		event.Actor.ID,
		event.Actor.Email,
		event.Actor.Source,
		event.Actor.ClientID,
		event.Actor.SessionID,
		event.Actor.IP,
		event.Actor.UserAgent,
	)

	// Build Error fields (if present)
	errorData := ""
	if event.Error != nil {
		errorData = fmt.Sprintf("%s|%s", event.Error.Code, event.Error.Message)
	}

	// Build Context fields (sorted keys for deterministic HMAC)
	contextData := ""
	if event.Context != nil {
		// Sort context keys for deterministic ordering
		keys := make([]string, 0, len(event.Context))
		for k := range event.Context {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			contextData += fmt.Sprintf("%s=%v|", k, event.Context[k])
		}
	}

	// Include all significant fields in HMAC calculation
	data := fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s|%s|%s|%d|%s",
		event.Version,
		event.ID,
		event.Timestamp,
		event.Operation,
		event.Key,
		actorData,
		event.Result,
		errorData,
		contextData,
		event.Chain.Sequence,
		event.Chain.PrevHash,
	)
	return []byte(data)
}

// sortStrings sorts a slice of strings in place (simple insertion sort)
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// writeEvent writes an event to the current month's log file
func (l *Logger) writeEvent(event *AuditEvent) error {
	// Get current month's filename
	filename := time.Now().UTC().Format("2006-01") + ".jsonl"
	filepath := filepath.Join(l.path, filename)

	// Open file for append
	f, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("audit: failed to open log file: %w", err)
	}
	defer f.Close()

	// Marshal and write
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("audit: failed to marshal event: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("audit: failed to write event: %w", err)
	}

	return nil
}

// ChainState holds the persistent chain state
type ChainState struct {
	Sequence int64  `json:"seq"`
	PrevHash string `json:"prev"`
}

// loadChainState loads the chain state from metadata file
func (l *Logger) loadChainState() error {
	metaPath := filepath.Join(l.path, "audit.meta")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return err
	}

	var state ChainState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	l.sequence = state.Sequence
	l.prevHash = state.PrevHash
	return nil
}

// saveChainState saves the chain state to metadata file
func (l *Logger) saveChainState() error {
	state := ChainState{
		Sequence: l.sequence,
		PrevHash: l.prevHash,
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("audit: failed to marshal chain state: %w", err)
	}

	metaPath := filepath.Join(l.path, "audit.meta")
	if err := os.WriteFile(metaPath, data, 0600); err != nil {
		return fmt.Errorf("audit: failed to save chain state: %w", err)
	}

	return nil
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("session-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// generateULID creates a ULID-like identifier
// Using timestamp + random for time-sortable unique IDs
func generateULID() string {
	// Timestamp component (48 bits = 6 bytes)
	ts := time.Now().UnixMilli()
	tsBytes := make([]byte, 6)
	for i := 5; i >= 0; i-- {
		tsBytes[i] = byte(ts & 0xFF)
		ts >>= 8
	}

	// Random component (80 bits = 10 bytes)
	randBytes := make([]byte, 10)
	if _, err := rand.Read(randBytes); err != nil {
		// Fallback
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Combine and encode
	combined := append(tsBytes, randBytes...)
	return hex.EncodeToString(combined)
}

// Verify checks the integrity of the audit log chain
func (l *Logger) Verify() (*VerifyResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.hmacKeySet {
		return nil, fmt.Errorf("audit: HMAC key not set")
	}

	result := &VerifyResult{
		Valid:        true,
		RecordsTotal: 0,
	}

	// Read all log files in order
	files, err := filepath.Glob(filepath.Join(l.path, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("audit: failed to list log files: %w", err)
	}

	// Sort files by name (YYYY-MM.jsonl format ensures chronological order)
	sortStrings(files)

	expectedPrevHash := "genesis"
	var expectedSeq int64 = 1

	for _, file := range files {
		events, err := l.readLogFile(file)
		if err != nil {
			return nil, fmt.Errorf("audit: failed to read %s: %w", file, err)
		}

		for _, event := range events {
			result.RecordsTotal++

			// Check sequence
			if event.Chain.Sequence != expectedSeq {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf(
					"sequence gap at record %s: expected %d, got %d",
					event.ID, expectedSeq, event.Chain.Sequence))
			}

			// Check prev hash
			if event.Chain.PrevHash != expectedPrevHash {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf(
					"chain broken at record %s: expected prev %s, got %s",
					event.ID, expectedPrevHash, event.Chain.PrevHash))
			}

			// Verify HMAC
			recordData := l.buildRecordData(&event)
			mac := hmac.New(sha256.New, l.hmacKey)
			mac.Write(recordData)
			expectedHMAC := hex.EncodeToString(mac.Sum(nil))

			if event.Chain.HMAC != expectedHMAC {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf(
					"HMAC mismatch at record %s: possible tampering",
					event.ID))
			}

			expectedPrevHash = event.Chain.HMAC
			expectedSeq++
		}
	}

	result.RecordsVerified = result.RecordsTotal
	return result, nil
}

// VerifyResult contains the results of chain verification
type VerifyResult struct {
	Valid           bool     `json:"valid"`
	RecordsTotal    int      `json:"records_total"`
	RecordsVerified int      `json:"records_verified"`
	Errors          []string `json:"errors,omitempty"`
}

// readLogFile reads all events from a log file
func (l *Logger) readLogFile(path string) ([]AuditEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var events []AuditEvent
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var event AuditEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return nil, fmt.Errorf("failed to parse line: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

// splitLines splits data into lines
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

// ListEvents returns audit events with optional filtering
// limit: maximum number of events to return (0 = all)
// since: only return events after this time (zero = no filter)
func (l *Logger) ListEvents(limit int, since time.Time) ([]AuditEvent, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Read all log files
	files, err := filepath.Glob(filepath.Join(l.path, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("audit: failed to list log files: %w", err)
	}

	// Sort files by name (chronological order)
	sortStrings(files)

	var allEvents []AuditEvent
	for _, file := range files {
		events, err := l.readLogFile(file)
		if err != nil {
			return nil, fmt.Errorf("audit: failed to read %s: %w", file, err)
		}
		allEvents = append(allEvents, events...)
	}

	// Filter by time if specified
	var filtered []AuditEvent
	if !since.IsZero() {
		for _, event := range allEvents {
			eventTime, err := time.Parse(time.RFC3339Nano, event.Timestamp)
			if err != nil {
				continue // Skip events with invalid timestamps
			}
			if eventTime.After(since) {
				filtered = append(filtered, event)
			}
		}
	} else {
		filtered = allEvents
	}

	// Apply limit (return most recent events)
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}

	return filtered, nil
}

// Path returns the audit log directory path
func (l *Logger) Path() string {
	return l.path
}

// Export exports audit events in the specified format (json or csv)
// since and until filter events by timestamp (zero values mean no filter)
func (l *Logger) Export(format string, since, until time.Time) ([]byte, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Read all log files
	files, err := filepath.Glob(filepath.Join(l.path, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("audit: failed to list log files: %w", err)
	}

	// Sort files by name (chronological order)
	sortStrings(files)

	var allEvents []AuditEvent
	for _, file := range files {
		events, err := l.readLogFile(file)
		if err != nil {
			return nil, fmt.Errorf("audit: failed to read %s: %w", file, err)
		}
		allEvents = append(allEvents, events...)
	}

	// Filter by time range
	var filtered []AuditEvent
	for _, event := range allEvents {
		eventTime, err := time.Parse(time.RFC3339Nano, event.Timestamp)
		if err != nil {
			continue // Skip events with invalid timestamps
		}

		// Apply since filter
		if !since.IsZero() && eventTime.Before(since) {
			continue
		}

		// Apply until filter
		if !until.IsZero() && eventTime.After(until) {
			continue
		}

		filtered = append(filtered, event)
	}

	// Format output
	switch format {
	case "csv":
		return l.formatCSV(filtered), nil
	case "json":
		return l.formatJSON(filtered)
	default:
		return nil, fmt.Errorf("audit: unsupported format: %s", format)
	}
}

// formatJSON formats events as JSON array
func (l *Logger) formatJSON(events []AuditEvent) ([]byte, error) {
	return json.MarshalIndent(events, "", "  ")
}

// formatCSV formats events as CSV with proper escaping
func (l *Logger) formatCSV(events []AuditEvent) []byte {
	var result []byte

	// Header
	result = append(result, []byte("timestamp,operation,result,key_hash\n")...)

	// Data rows
	for _, event := range events {
		keyHash := event.Key
		if len(keyHash) > 16 {
			keyHash = keyHash[:16] + "..."
		}
		// Escape fields to prevent CSV injection
		line := fmt.Sprintf("%s,%s,%s,%s\n",
			csvEscape(event.Timestamp),
			csvEscape(event.Operation),
			csvEscape(event.Result),
			csvEscape(keyHash),
		)
		result = append(result, []byte(line)...)
	}

	return result
}

// csvEscape escapes a field for CSV output to prevent injection attacks
func csvEscape(field string) string {
	if field == "" {
		return field
	}

	// Check if field needs quoting
	// Also quote fields starting with =, +, -, @ to prevent formula injection
	needsQuoting := false
	firstChar := field[0]
	if firstChar == '=' || firstChar == '+' || firstChar == '-' || firstChar == '@' {
		needsQuoting = true
	}

	if !needsQuoting {
		for _, c := range field {
			if c == ',' || c == '"' || c == '\n' || c == '\r' {
				needsQuoting = true
				break
			}
		}
	}

	if !needsQuoting {
		return field
	}

	// Quote the field and escape any double quotes
	var escaped []byte
	escaped = append(escaped, '"')
	for _, c := range field {
		if c == '"' {
			escaped = append(escaped, '"', '"') // Escape double quote with double quote
		} else {
			escaped = append(escaped, byte(c))
		}
	}
	escaped = append(escaped, '"')
	return string(escaped)
}

// Prune deletes audit log entries older than the specified duration
// Returns the number of deleted entries
func (l *Logger) Prune(olderThan time.Duration) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)

	// Read all log files
	files, err := filepath.Glob(filepath.Join(l.path, "*.jsonl"))
	if err != nil {
		return 0, fmt.Errorf("audit: failed to list log files: %w", err)
	}

	sortStrings(files)

	deletedCount := 0

	for _, file := range files {
		events, err := l.readLogFile(file)
		if err != nil {
			return deletedCount, fmt.Errorf("audit: failed to read %s: %w", file, err)
		}

		// Check if all events in this file are older than cutoff
		allOld := true
		for _, event := range events {
			eventTime, err := time.Parse(time.RFC3339Nano, event.Timestamp)
			if err != nil {
				continue
			}
			if eventTime.After(cutoff) {
				allOld = false
				break
			}
		}

		if allOld && len(events) > 0 {
			// Delete entire file
			if err := os.Remove(file); err != nil {
				return deletedCount, fmt.Errorf("audit: failed to delete %s: %w", file, err)
			}
			deletedCount += len(events)
		} else if !allOld {
			// Need to filter events within the file
			var remaining []AuditEvent
			for _, event := range events {
				eventTime, err := time.Parse(time.RFC3339Nano, event.Timestamp)
				if err != nil {
					remaining = append(remaining, event)
					continue
				}
				if eventTime.After(cutoff) {
					remaining = append(remaining, event)
				} else {
					deletedCount++
				}
			}

			// Rewrite file with remaining events
			if len(remaining) == 0 {
				if err := os.Remove(file); err != nil {
					return deletedCount, fmt.Errorf("audit: failed to delete %s: %w", file, err)
				}
			} else {
				if err := l.rewriteLogFile(file, remaining); err != nil {
					return deletedCount, fmt.Errorf("audit: failed to rewrite %s: %w", file, err)
				}
			}
		}
	}

	return deletedCount, nil
}

// PrunePreview returns the count of entries that would be deleted
// without actually deleting them (for --dry-run)
func (l *Logger) PrunePreview(olderThan time.Duration) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)

	// Read all log files
	files, err := filepath.Glob(filepath.Join(l.path, "*.jsonl"))
	if err != nil {
		return 0, fmt.Errorf("audit: failed to list log files: %w", err)
	}

	count := 0
	for _, file := range files {
		events, err := l.readLogFile(file)
		if err != nil {
			return 0, fmt.Errorf("audit: failed to read %s: %w", file, err)
		}

		for _, event := range events {
			eventTime, err := time.Parse(time.RFC3339Nano, event.Timestamp)
			if err != nil {
				continue
			}
			if eventTime.Before(cutoff) {
				count++
			}
		}
	}

	return count, nil
}

// rewriteLogFile rewrites a log file with the given events
func (l *Logger) rewriteLogFile(path string, events []AuditEvent) error {
	// Write to temp file first
	tempPath := path + ".tmp"
	f, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			f.Close()
			os.Remove(tempPath)
			return err
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			f.Close()
			os.Remove(tempPath)
			return err
		}
	}

	if err := f.Close(); err != nil {
		os.Remove(tempPath)
		return err
	}

	// Atomic rename
	return os.Rename(tempPath, path)
}

// checkDiskSpace verifies sufficient disk space for audit log writes
func (l *Logger) checkDiskSpace() error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(l.path, &stat); err != nil {
		// If audit directory doesn't exist yet, check parent
		parentDir := filepath.Dir(l.path)
		if err := syscall.Statfs(parentDir, &stat); err != nil {
			// Log warning but don't block audit operation
			fmt.Fprintf(os.Stderr, "warning: failed to check disk space for audit: %v\n", err)
			return nil
		}
	}

	available := stat.Bavail * uint64(stat.Bsize)
	if available < MinAuditDiskSpace {
		return fmt.Errorf("audit: insufficient disk space: only %d bytes available, need at least %d",
			available, MinAuditDiskSpace)
	}

	return nil
}
