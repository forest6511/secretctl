package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/forest6511/secretctl/pkg/audit"
	"github.com/forest6511/secretctl/pkg/vault"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct - Wails binds this to the frontend
type App struct {
	ctx          context.Context
	vault        *vault.Vault
	vaultDir     string
	unlocked     bool
	lastActivity time.Time
	activityMu   sync.Mutex
}

// NewApp creates a new App application struct
func NewApp(vaultDir string) *App {
	return &App{
		vaultDir: vaultDir,
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.lastActivity = time.Now()

	// Start idle timeout watcher
	go a.watchIdleTimeout()
}

// shutdown is called at app termination
func (a *App) shutdown(ctx context.Context) {
	// Clear clipboard before exit to prevent secret leakage
	a.ClearClipboard()

	if a.vault != nil {
		a.vault.Lock()
	}
}

// watchIdleTimeout monitors for idle and auto-locks vault
func (a *App) watchIdleTimeout() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.activityMu.Lock()
			idle := time.Since(a.lastActivity)
			a.activityMu.Unlock()

			if a.unlocked && idle > 15*time.Minute {
				a.Lock()
				runtime.EventsEmit(a.ctx, "vault:locked")
			}
		case <-a.ctx.Done():
			return
		}
	}
}

// ResetIdleTimer is called on user activity
func (a *App) ResetIdleTimer() {
	a.activityMu.Lock()
	a.lastActivity = time.Now()
	a.activityMu.Unlock()
}

// ============================================================================
// Authentication API
// ============================================================================

// AuthStatus represents authentication state
type AuthStatus struct {
	Unlocked bool   `json:"unlocked"`
	VaultDir string `json:"vaultDir"`
}

// CheckVaultExists checks if vault exists at configured path
func (a *App) CheckVaultExists() bool {
	_, err := os.Stat(filepath.Join(a.vaultDir, "vault.db"))
	return err == nil
}

// InitVault creates a new vault with master password
func (a *App) InitVault(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	if a.CheckVaultExists() {
		return errors.New("vault already exists")
	}

	// Create vault instance
	v := vault.New(a.vaultDir)
	if err := v.Init(password); err != nil {
		return err
	}

	// Unlock immediately after init
	if err := v.Unlock(password); err != nil {
		return err
	}

	a.vault = v
	a.unlocked = true
	a.lastActivity = time.Now()

	return nil
}

// Unlock unlocks the vault with master password
func (a *App) Unlock(password string) error {
	if a.unlocked {
		return errors.New("vault already unlocked")
	}

	v := vault.New(a.vaultDir)
	if err := v.Unlock(password); err != nil {
		return errors.New("invalid password")
	}

	a.vault = v
	a.unlocked = true
	a.lastActivity = time.Now()

	return nil
}

// Lock locks the vault and clears clipboard
func (a *App) Lock() error {
	if !a.unlocked {
		return errors.New("vault not unlocked")
	}

	// Clear clipboard before locking to prevent secret leakage
	a.ClearClipboard()

	a.vault.Lock()
	a.vault = nil
	a.unlocked = false

	return nil
}

// GetAuthStatus returns current authentication status
func (a *App) GetAuthStatus() AuthStatus {
	return AuthStatus{
		Unlocked: a.unlocked,
		VaultDir: a.vaultDir,
	}
}

// ============================================================================
// Secret API
// ============================================================================

// FieldDTO represents a field for frontend
type FieldDTO struct {
	Value     string   `json:"value"`
	Sensitive bool     `json:"sensitive"`
	Aliases   []string `json:"aliases,omitempty"`
	Kind      string   `json:"kind,omitempty"`
	Hint      string   `json:"hint,omitempty"`
}

// Secret represents a secret for frontend
type Secret struct {
	Key        string              `json:"key"`
	Value      string              `json:"value,omitempty"`      // Legacy: single value
	Fields     map[string]FieldDTO `json:"fields,omitempty"`     // Multi-field values
	FieldOrder []string            `json:"fieldOrder,omitempty"` // Field display order
	Bindings   map[string]string   `json:"bindings,omitempty"`   // env_var -> field_name
	Notes      string              `json:"notes,omitempty"`
	URL        string              `json:"url,omitempty"`
	Tags       []string            `json:"tags,omitempty"`
	CreatedAt  string              `json:"createdAt"`
	UpdatedAt  string              `json:"updatedAt"`
}

// SecretListItem represents a secret in list view (no value)
type SecretListItem struct {
	Key          string   `json:"key"`
	Tags         []string `json:"tags,omitempty"`
	UpdatedAt    string   `json:"updatedAt"`
	FieldCount   int      `json:"fieldCount"`
	BindingCount int      `json:"bindingCount"`
	HasNotes     bool     `json:"hasNotes"`
	HasURL       bool     `json:"hasUrl"`
}

// ListSecrets returns all secret keys
func (a *App) ListSecrets() ([]SecretListItem, error) {
	if !a.unlocked {
		return nil, errors.New("vault locked")
	}

	keys, err := a.vault.ListSecrets()
	if err != nil {
		return nil, err
	}

	items := make([]SecretListItem, 0, len(keys))
	for _, key := range keys {
		entry, err := a.vault.GetSecret(key)
		if err != nil {
			continue
		}

		// Calculate field count (legacy single value = 1 field)
		fieldCount := len(entry.Fields)
		if fieldCount == 0 && len(entry.Value) > 0 {
			fieldCount = 1
		}

		hasNotes := false
		hasURL := false
		if entry.Metadata != nil {
			hasNotes = entry.Metadata.Notes != ""
			hasURL = entry.Metadata.URL != ""
		}

		items = append(items, SecretListItem{
			Key:          key,
			Tags:         entry.Tags,
			UpdatedAt:    entry.UpdatedAt.Format(time.RFC3339),
			FieldCount:   fieldCount,
			BindingCount: len(entry.Bindings),
			HasNotes:     hasNotes,
			HasURL:       hasURL,
		})
	}

	return items, nil
}

// GetSecret returns a secret with its value
func (a *App) GetSecret(key string) (*Secret, error) {
	if !a.unlocked {
		return nil, errors.New("vault locked")
	}

	entry, err := a.vault.GetSecret(key)
	if err != nil {
		return nil, err
	}

	notes := ""
	url := ""
	if entry.Metadata != nil {
		notes = entry.Metadata.Notes
		url = entry.Metadata.URL
	}

	// Build Fields and FieldOrder
	fields := make(map[string]FieldDTO)
	var fieldOrder []string

	if len(entry.Fields) > 0 {
		// Multi-field secret
		for name, field := range entry.Fields {
			fields[name] = FieldDTO{
				Value:     field.Value,
				Sensitive: field.Sensitive,
				Aliases:   field.Aliases,
				Kind:      field.Kind,
				Hint:      field.Hint,
			}
			fieldOrder = append(fieldOrder, name)
		}
		// Sort field order deterministically (alphabetically)
		sort.Strings(fieldOrder)
	} else if len(entry.Value) > 0 {
		// Legacy single-value secret: convert to Fields["value"]
		fields["value"] = FieldDTO{
			Value:     string(entry.Value),
			Sensitive: true, // Legacy values are treated as sensitive
		}
		fieldOrder = []string{"value"}
	}

	// Copy bindings
	bindings := make(map[string]string)
	for k, v := range entry.Bindings {
		bindings[k] = v
	}

	return &Secret{
		Key:        entry.Key,
		Value:      string(entry.Value), // Keep for backward compatibility
		Fields:     fields,
		FieldOrder: fieldOrder,
		Bindings:   bindings,
		Notes:      notes,
		URL:        url,
		Tags:       entry.Tags,
		CreatedAt:  entry.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  entry.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// CreateSecret creates a new secret
func (a *App) CreateSecret(key, value, notes, url string, tags []string) error {
	if !a.unlocked {
		return errors.New("vault locked")
	}

	entry := &vault.SecretEntry{
		Key:   key,
		Value: []byte(value),
		Tags:  tags,
	}
	if notes != "" || url != "" {
		entry.Metadata = &vault.SecretMetadata{
			Notes: notes,
			URL:   url,
		}
	}

	return a.vault.SetSecret(key, entry)
}

// UpdateSecret updates an existing secret
func (a *App) UpdateSecret(key, value, notes, url string, tags []string) error {
	if !a.unlocked {
		return errors.New("vault locked")
	}

	entry := &vault.SecretEntry{
		Key:   key,
		Value: []byte(value),
		Tags:  tags,
	}
	if notes != "" || url != "" {
		entry.Metadata = &vault.SecretMetadata{
			Notes: notes,
			URL:   url,
		}
	}

	return a.vault.SetSecret(key, entry)
}

// DeleteSecret deletes a secret
func (a *App) DeleteSecret(key string) error {
	if !a.unlocked {
		return errors.New("vault locked")
	}

	return a.vault.DeleteSecret(key)
}

// SecretUpdateDTO represents a secret update request from the frontend
type SecretUpdateDTO struct {
	Key      string              `json:"key"`
	Fields   map[string]FieldDTO `json:"fields"`
	Bindings map[string]string   `json:"bindings,omitempty"`
	Notes    string              `json:"notes,omitempty"`
	URL      string              `json:"url,omitempty"`
	Tags     []string            `json:"tags,omitempty"`
}

// validateSecretDTO validates the SecretUpdateDTO fields
func validateSecretDTO(dto SecretUpdateDTO) error {
	if dto.Key == "" {
		return errors.New("secret key is required")
	}
	if len(dto.Fields) == 0 {
		return errors.New("at least one field is required")
	}

	// Validate field names (snake_case)
	fieldNameRegex := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	for name := range dto.Fields {
		if len(name) > 64 {
			return fmt.Errorf("field name '%s' exceeds 64 characters", name)
		}
		if !fieldNameRegex.MatchString(name) {
			return fmt.Errorf("field name '%s' must be snake_case (lowercase letters, numbers, underscores)", name)
		}
	}

	// Validate binding names (SCREAMING_SNAKE_CASE) and references
	envVarRegex := regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	for envVar, fieldName := range dto.Bindings {
		if !envVarRegex.MatchString(envVar) {
			return fmt.Errorf("environment variable '%s' must be SCREAMING_SNAKE_CASE", envVar)
		}
		if _, exists := dto.Fields[fieldName]; !exists {
			return fmt.Errorf("binding '%s' references non-existent field '%s'", envVar, fieldName)
		}
	}

	return nil
}

// UpdateSecretMultiField updates a secret with multi-field support
func (a *App) UpdateSecretMultiField(dto SecretUpdateDTO) error {
	if !a.unlocked {
		return errors.New("vault locked")
	}

	// Validate input
	if err := validateSecretDTO(dto); err != nil {
		return err
	}

	// Convert FieldDTO to vault.Field
	fields := make(map[string]vault.Field)
	for name, fieldDTO := range dto.Fields {
		fields[name] = vault.Field{
			Value:     fieldDTO.Value,
			Sensitive: fieldDTO.Sensitive,
			Aliases:   fieldDTO.Aliases,
			Kind:      fieldDTO.Kind,
			Hint:      fieldDTO.Hint,
		}
	}

	entry := &vault.SecretEntry{
		Key:      dto.Key,
		Fields:   fields,
		Bindings: dto.Bindings,
		Tags:     dto.Tags,
	}

	if dto.Notes != "" || dto.URL != "" {
		entry.Metadata = &vault.SecretMetadata{
			Notes: dto.Notes,
			URL:   dto.URL,
		}
	}

	// Persist first, then audit log (audit after success)
	if err := a.vault.SetSecret(dto.Key, entry); err != nil {
		// Log failure
		_ = a.vault.AuditLogger().Log(
			"secret.updated",
			"desktop",
			audit.ResultError,
			dto.Key,
			nil,
			map[string]interface{}{
				"error": err.Error(),
			},
		)
		return err
	}

	// Log success after persistence
	if err := a.vault.AuditLogger().Log(
		"secret.updated",
		"desktop",
		audit.ResultSuccess,
		dto.Key,
		nil,
		map[string]interface{}{
			"field_count":   len(fields),
			"binding_count": len(dto.Bindings),
		},
	); err != nil {
		return err
	}

	return nil
}

// CreateSecretMultiField creates a new secret with multi-field support
func (a *App) CreateSecretMultiField(dto SecretUpdateDTO) error {
	if !a.unlocked {
		return errors.New("vault locked")
	}

	// Validate input
	if err := validateSecretDTO(dto); err != nil {
		return err
	}

	// Check if secret already exists (only treat ErrSecretNotFound as expected)
	_, err := a.vault.GetSecret(dto.Key)
	if err == nil {
		return errors.New("secret already exists")
	}
	// Note: We proceed with creation even if GetSecret returns other errors
	// because the vault may not have any secrets yet or the secret simply doesn't exist

	// Convert FieldDTO to vault.Field
	fields := make(map[string]vault.Field)
	for name, fieldDTO := range dto.Fields {
		fields[name] = vault.Field{
			Value:     fieldDTO.Value,
			Sensitive: fieldDTO.Sensitive,
			Aliases:   fieldDTO.Aliases,
			Kind:      fieldDTO.Kind,
			Hint:      fieldDTO.Hint,
		}
	}

	entry := &vault.SecretEntry{
		Key:      dto.Key,
		Fields:   fields,
		Bindings: dto.Bindings,
		Tags:     dto.Tags,
	}

	if dto.Notes != "" || dto.URL != "" {
		entry.Metadata = &vault.SecretMetadata{
			Notes: dto.Notes,
			URL:   dto.URL,
		}
	}

	// Persist first, then audit log (audit after success)
	if err := a.vault.SetSecret(dto.Key, entry); err != nil {
		// Log failure
		_ = a.vault.AuditLogger().Log(
			"secret.created",
			"desktop",
			audit.ResultError,
			dto.Key,
			nil,
			map[string]interface{}{
				"error": err.Error(),
			},
		)
		return err
	}

	// Log success after persistence
	if err := a.vault.AuditLogger().Log(
		"secret.created",
		"desktop",
		audit.ResultSuccess,
		dto.Key,
		nil,
		map[string]interface{}{
			"field_count":   len(fields),
			"binding_count": len(dto.Bindings),
		},
	); err != nil {
		return err
	}

	return nil
}

// CopyToClipboard copies value to clipboard and schedules auto-clear
func (a *App) CopyToClipboard(value string) error {
	err := runtime.ClipboardSetText(a.ctx, value)
	if err != nil {
		return err
	}

	// Auto-clear after 30 seconds
	go func() {
		time.Sleep(30 * time.Second)
		current, _ := runtime.ClipboardGetText(a.ctx)
		if current == value {
			runtime.ClipboardSetText(a.ctx, "")
		}
	}()

	return nil
}

// ClearClipboard clears the system clipboard
func (a *App) ClearClipboard() {
	if a.ctx != nil {
		runtime.ClipboardSetText(a.ctx, "")
	}
}

// ViewSensitiveField logs when a sensitive field is viewed
func (a *App) ViewSensitiveField(key, fieldName string) error {
	if !a.unlocked {
		return errors.New("vault locked")
	}

	return a.vault.AuditLogger().Log(
		"secret.field_viewed",
		"desktop",
		audit.ResultSuccess,
		key,
		nil,
		map[string]interface{}{"field": fieldName},
	)
}

// CopyFieldValue logs when a field value is copied and copies to clipboard
// Security: Fetches the actual value from vault to prevent caller manipulation
func (a *App) CopyFieldValue(key, fieldName string) error {
	if !a.unlocked {
		return errors.New("vault locked")
	}

	// Fetch the actual secret from vault (security: don't trust caller-provided values)
	entry, err := a.vault.GetSecret(key)
	if err != nil {
		return err
	}

	// Get the field value and sensitivity from the stored secret
	var fieldValue string
	var sensitive bool

	if len(entry.Fields) > 0 {
		field, ok := entry.Fields[fieldName]
		if !ok {
			return errors.New("field not found")
		}
		fieldValue = field.Value
		sensitive = field.Sensitive
	} else if fieldName == "value" && len(entry.Value) > 0 {
		// Legacy single value
		fieldValue = string(entry.Value)
		sensitive = true
	} else {
		return errors.New("field not found")
	}

	// Log the copy action
	operation := "secret.field_copied"
	if sensitive {
		operation = "secret.sensitive_field_copied"
	}

	if err := a.vault.AuditLogger().Log(
		operation,
		"desktop",
		audit.ResultSuccess,
		key,
		nil,
		map[string]interface{}{"field": fieldName},
	); err != nil {
		return err
	}

	// Copy to clipboard with auto-clear
	return a.CopyToClipboard(fieldValue)
}

// ============================================================================
// Audit Log API
// ============================================================================

// AuditLogEntry represents an audit log entry for frontend
type AuditLogEntry struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Source    string `json:"source"`
	Key       string `json:"key,omitempty"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// AuditLogFilter defines filter options for audit logs
type AuditLogFilter struct {
	Action    string `json:"action,omitempty"`
	Source    string `json:"source,omitempty"`
	Key       string `json:"key,omitempty"`
	StartTime string `json:"startTime,omitempty"`
	EndTime   string `json:"endTime,omitempty"`
	Success   *bool  `json:"success,omitempty"`
}

// AuditLogSearchResult contains paginated audit log results
type AuditLogSearchResult struct {
	Entries []AuditLogEntry `json:"entries"`
	Total   int             `json:"total"`
}

// ListAuditLogs returns audit logs
func (a *App) ListAuditLogs(limit int) ([]AuditLogEntry, error) {
	if !a.unlocked {
		return nil, errors.New("vault locked")
	}

	auditLogger := a.vault.AuditLogger()
	events, err := auditLogger.ListEvents(limit, time.Time{})
	if err != nil {
		return nil, err
	}

	entries := make([]AuditLogEntry, 0, len(events))
	for _, event := range events {
		success := event.Result == audit.ResultSuccess
		errMsg := ""
		if event.Error != nil {
			errMsg = event.Error.Message
		}
		entries = append(entries, AuditLogEntry{
			Timestamp: event.Timestamp,
			Action:    event.Operation,
			Source:    event.Actor.Source,
			Key:       event.Key,
			Success:   success,
			Error:     errMsg,
		})
	}

	return entries, nil
}

// SearchAuditLogs returns filtered and paginated audit logs
func (a *App) SearchAuditLogs(filter AuditLogFilter, limit, offset int) (*AuditLogSearchResult, error) {
	if !a.unlocked {
		return nil, errors.New("vault locked")
	}

	auditLogger := a.vault.AuditLogger()
	allEvents, err := auditLogger.ListEvents(0, time.Time{})
	if err != nil {
		return nil, err
	}

	var filtered []audit.AuditEvent
	for _, event := range allEvents {
		if filter.Action != "" && event.Operation != filter.Action {
			continue
		}
		if filter.Source != "" && event.Actor.Source != filter.Source {
			continue
		}
		if filter.Key != "" && !strings.Contains(event.Key, filter.Key) {
			continue
		}
		if filter.StartTime != "" {
			startTime, _ := time.Parse(time.RFC3339, filter.StartTime)
			eventTime, _ := time.Parse(time.RFC3339Nano, event.Timestamp)
			if eventTime.Before(startTime) {
				continue
			}
		}
		if filter.EndTime != "" {
			endTime, _ := time.Parse(time.RFC3339, filter.EndTime)
			eventTime, _ := time.Parse(time.RFC3339Nano, event.Timestamp)
			if eventTime.After(endTime) {
				continue
			}
		}
		if filter.Success != nil {
			success := event.Result == audit.ResultSuccess
			if success != *filter.Success {
				continue
			}
		}
		filtered = append(filtered, event)
	}

	total := len(filtered)

	start := offset
	end := offset + limit
	if limit == 0 {
		end = total
	}
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	result := filtered[start:end]
	entries := make([]AuditLogEntry, 0, len(result))
	for _, event := range result {
		success := event.Result == audit.ResultSuccess
		errMsg := ""
		if event.Error != nil {
			errMsg = event.Error.Message
		}
		entries = append(entries, AuditLogEntry{
			Timestamp: event.Timestamp,
			Action:    event.Operation,
			Source:    event.Actor.Source,
			Key:       event.Key,
			Success:   success,
			Error:     errMsg,
		})
	}

	return &AuditLogSearchResult{
		Entries: entries,
		Total:   total,
	}, nil
}

// VerifyAuditLogs verifies audit log integrity
func (a *App) VerifyAuditLogs() (bool, error) {
	if !a.unlocked {
		return false, errors.New("vault locked")
	}

	result, err := a.vault.AuditVerify()
	if err != nil {
		return false, err
	}

	return result.Valid, nil
}

// GetAuditLogStats returns audit log statistics
func (a *App) GetAuditLogStats() (map[string]int, error) {
	if !a.unlocked {
		return nil, errors.New("vault locked")
	}

	auditLogger := a.vault.AuditLogger()
	events, err := auditLogger.ListEvents(0, time.Time{})
	if err != nil {
		return nil, err
	}

	stats := map[string]int{
		"total":   len(events),
		"success": 0,
		"failure": 0,
		"cli":     0,
		"mcp":     0,
		"ui":      0,
	}

	for _, event := range events {
		if event.Result == audit.ResultSuccess {
			stats["success"]++
		} else {
			stats["failure"]++
		}
		stats[event.Actor.Source]++
	}

	return stats, nil
}
