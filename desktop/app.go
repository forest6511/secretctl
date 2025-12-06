package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

// Lock locks the vault
func (a *App) Lock() error {
	if !a.unlocked {
		return errors.New("vault not unlocked")
	}

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

// Secret represents a secret for frontend
type Secret struct {
	Key       string   `json:"key"`
	Value     string   `json:"value,omitempty"`
	Notes     string   `json:"notes,omitempty"`
	URL       string   `json:"url,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

// SecretListItem represents a secret in list view (no value)
type SecretListItem struct {
	Key       string   `json:"key"`
	Tags      []string `json:"tags,omitempty"`
	UpdatedAt string   `json:"updatedAt"`
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
		items = append(items, SecretListItem{
			Key:       key,
			Tags:      entry.Tags,
			UpdatedAt: entry.UpdatedAt.Format(time.RFC3339),
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

	return &Secret{
		Key:       entry.Key,
		Value:     string(entry.Value),
		Notes:     notes,
		URL:       url,
		Tags:      entry.Tags,
		CreatedAt: entry.CreatedAt.Format(time.RFC3339),
		UpdatedAt: entry.UpdatedAt.Format(time.RFC3339),
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
func (a *App) SearchAuditLogs(filter AuditLogFilter, limit, offset int) ([]AuditLogEntry, int, error) {
	if !a.unlocked {
		return nil, 0, errors.New("vault locked")
	}

	auditLogger := a.vault.AuditLogger()
	allEvents, err := auditLogger.ListEvents(0, time.Time{})
	if err != nil {
		return nil, 0, err
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

	return entries, total, nil
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
