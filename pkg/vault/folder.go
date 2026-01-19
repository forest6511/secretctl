// Package vault provides secure secret storage with AES-256-GCM encryption.
package vault

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Folder validation constants per ADR-007
const (
	MaxFolderNameLength = 128
	MinFolderNameLength = 1
	MaxFolderDepth      = 10 // Maximum nesting depth to prevent infinite loops
)

// Folder errors
var (
	ErrFolderNotFound      = errors.New("vault: folder not found")
	ErrFolderNameInvalid   = errors.New("vault: folder name is invalid")
	ErrFolderNameTooLong   = errors.New("vault: folder name is too long")
	ErrFolderNameTooShort  = errors.New("vault: folder name is too short")
	ErrFolderNameSlash     = errors.New("vault: folder name cannot contain '/'")
	ErrFolderExists        = errors.New("vault: folder already exists with this name")
	ErrFolderHasChildren   = errors.New("vault: folder has children (use --force to move to unfiled)")
	ErrFolderHasSecrets    = errors.New("vault: folder contains secrets (use --force to move to unfiled)")
	ErrFolderCircular      = errors.New("vault: circular parent reference detected")
	ErrFolderDepthExceeded = errors.New("vault: maximum folder depth exceeded")
	ErrFolderPathAmbiguous = errors.New("vault: folder path is ambiguous")
	ErrFolderPathNotFound  = errors.New("vault: folder path not found")
	ErrFolderSelfParent    = errors.New("vault: folder cannot be its own parent")
	ErrFolderIDInvalid     = errors.New("vault: folder ID is invalid")
)

// Folder represents a folder for organizing secrets.
// Per ADR-007: FolderId + Folder Table design.
type Folder struct {
	ID        string    `json:"id"`                  // UUID
	Name      string    `json:"name"`                // Display name (no "/" allowed)
	ParentID  *string   `json:"parent_id,omitempty"` // NULL for root, UUID for nested
	Icon      string    `json:"icon,omitempty"`      // Emoji or icon name
	Color     string    `json:"color,omitempty"`     // Hex color code
	SortOrder int       `json:"sort_order"`          // For manual ordering
	Path      string    `json:"path,omitempty"`      // Computed: "Work/APIs" (not stored)
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FolderWithStats extends Folder with computed statistics for listing.
type FolderWithStats struct {
	Folder
	SecretCount    int `json:"secret_count"`    // Number of secrets directly in folder
	SubfolderCount int `json:"subfolder_count"` // Number of immediate subfolders
}

// colorHexRegex validates hex color codes
var colorHexRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// validateFolderName validates a folder name per ADR-007.
func validateFolderName(name string) error {
	if len(name) < MinFolderNameLength {
		return ErrFolderNameTooShort
	}
	if len(name) > MaxFolderNameLength {
		return ErrFolderNameTooLong
	}
	if strings.Contains(name, "/") {
		return ErrFolderNameSlash
	}
	// Allow most printable characters except "/"
	// The CHECK constraint in SQL enforces no "/"
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrFolderNameTooShort
	}
	return nil
}

// validateFolderColor validates a hex color code.
func validateFolderColor(color string) error {
	if color == "" {
		return nil // Empty is allowed
	}
	if !colorHexRegex.MatchString(color) {
		return fmt.Errorf("vault: invalid color format (expected #RRGGBB): %s", color)
	}
	return nil
}

// CreateFolder creates a new folder.
// Per ADR-007: PRAGMA foreign_keys = ON is set on connection.
func (v *Vault) CreateFolder(folder *Folder) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.dek == nil {
		return ErrVaultLocked
	}

	// Validate folder name
	if err := validateFolderName(folder.Name); err != nil {
		return err
	}

	// Validate color if provided
	if err := validateFolderColor(folder.Color); err != nil {
		return err
	}

	// Generate UUID if not provided
	if folder.ID == "" {
		folder.ID = uuid.New().String()
	}

	// Validate parent exists if specified
	if folder.ParentID != nil && *folder.ParentID != "" {
		var parentExists int
		err := v.db.QueryRow("SELECT COUNT(*) FROM folders WHERE id = ?", *folder.ParentID).Scan(&parentExists)
		if err != nil {
			return fmt.Errorf("vault: failed to check parent folder: %w", err)
		}
		if parentExists == 0 {
			return fmt.Errorf("%w: parent_id %s", ErrFolderNotFound, *folder.ParentID)
		}

		// Check depth doesn't exceed maximum
		depth, err := v.getFolderDepth(*folder.ParentID)
		if err != nil {
			return err
		}
		if depth >= MaxFolderDepth-1 {
			return ErrFolderDepthExceeded
		}
	}

	// Insert folder
	now := time.Now().UTC()
	_, err := v.db.Exec(`
		INSERT INTO folders (id, name, parent_id, icon, color, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, folder.ID, folder.Name, folder.ParentID, folder.Icon, folder.Color, folder.SortOrder, now, now)

	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return ErrFolderExists
		}
		return fmt.Errorf("vault: failed to create folder: %w", err)
	}

	folder.CreatedAt = now
	folder.UpdatedAt = now

	return nil
}

// GetFolder retrieves a folder by ID.
func (v *Vault) GetFolder(id string) (*Folder, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.dek == nil {
		return nil, ErrVaultLocked
	}

	folder := &Folder{}
	var parentID sql.NullString
	var icon, color sql.NullString

	err := v.db.QueryRow(`
		SELECT id, name, parent_id, icon, color, sort_order, created_at, updated_at
		FROM folders WHERE id = ?
	`, id).Scan(
		&folder.ID, &folder.Name, &parentID, &icon, &color,
		&folder.SortOrder, &folder.CreatedAt, &folder.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrFolderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("vault: failed to get folder: %w", err)
	}

	if parentID.Valid {
		folder.ParentID = &parentID.String
	}
	if icon.Valid {
		folder.Icon = icon.String
	}
	if color.Valid {
		folder.Color = color.String
	}

	// Compute path
	folder.Path, _ = v.computeFolderPathLocked(folder.ID)

	return folder, nil
}

// GetFolderByPath finds a folder by its path (e.g., "Work/APIs").
// Returns ErrFolderPathAmbiguous if multiple folders match.
func (v *Vault) GetFolderByPath(path string) (*Folder, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.dek == nil {
		return nil, ErrVaultLocked
	}

	return v.getFolderByPathLocked(path)
}

// getFolderByPathLocked finds a folder by path without locking.
func (v *Vault) getFolderByPathLocked(path string) (*Folder, error) {
	if path == "" {
		return nil, ErrFolderPathNotFound
	}

	parts := strings.Split(path, "/")
	var currentParentID *string

	for i, name := range parts {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		// Find folder with this name and parent
		var folders []Folder
		var query string
		var args []any

		if currentParentID == nil {
			query = "SELECT id, name, parent_id FROM folders WHERE name = ? COLLATE NOCASE AND parent_id IS NULL"
			args = []any{name}
		} else {
			query = "SELECT id, name, parent_id FROM folders WHERE name = ? COLLATE NOCASE AND parent_id = ?"
			args = []any{name, *currentParentID}
		}

		rows, err := v.db.Query(query, args...)
		if err != nil {
			return nil, fmt.Errorf("vault: failed to query folders: %w", err)
		}

		for rows.Next() {
			var f Folder
			var parentID sql.NullString
			if err := rows.Scan(&f.ID, &f.Name, &parentID); err != nil {
				rows.Close()
				return nil, fmt.Errorf("vault: failed to scan folder: %w", err)
			}
			if parentID.Valid {
				f.ParentID = &parentID.String
			}
			folders = append(folders, f)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("vault: error iterating folders: %w", err)
		}
		rows.Close()

		if len(folders) == 0 {
			return nil, fmt.Errorf("%w: %s", ErrFolderPathNotFound, path)
		}
		if len(folders) > 1 {
			return nil, fmt.Errorf("%w: multiple folders named '%s' at this level", ErrFolderPathAmbiguous, name)
		}

		// Move to next level
		if i == len(parts)-1 {
			// Last part - return the full folder
			return v.GetFolder(folders[0].ID)
		}
		currentParentID = &folders[0].ID
	}

	return nil, ErrFolderPathNotFound
}

// ListFolders returns all folders with optional parent filter.
// If parentID is nil, returns all folders. If parentID is empty string pointer,
// returns root folders only. Otherwise returns children of specified parent.
func (v *Vault) ListFolders(parentID *string) ([]*FolderWithStats, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.dek == nil {
		return nil, ErrVaultLocked
	}

	var query string
	var args []any

	if parentID == nil {
		// All folders
		query = `
			SELECT f.id, f.name, f.parent_id, f.icon, f.color, f.sort_order, f.created_at, f.updated_at,
			       (SELECT COUNT(*) FROM secrets s WHERE s.folder_id = f.id) as secret_count,
			       (SELECT COUNT(*) FROM folders c WHERE c.parent_id = f.id) as subfolder_count
			FROM folders f
			ORDER BY f.parent_id NULLS FIRST, f.sort_order, f.name COLLATE NOCASE
		`
	} else if *parentID == "" {
		// Root folders only
		query = `
			SELECT f.id, f.name, f.parent_id, f.icon, f.color, f.sort_order, f.created_at, f.updated_at,
			       (SELECT COUNT(*) FROM secrets s WHERE s.folder_id = f.id) as secret_count,
			       (SELECT COUNT(*) FROM folders c WHERE c.parent_id = f.id) as subfolder_count
			FROM folders f
			WHERE f.parent_id IS NULL
			ORDER BY f.sort_order, f.name COLLATE NOCASE
		`
	} else {
		// Children of specific parent
		query = `
			SELECT f.id, f.name, f.parent_id, f.icon, f.color, f.sort_order, f.created_at, f.updated_at,
			       (SELECT COUNT(*) FROM secrets s WHERE s.folder_id = f.id) as secret_count,
			       (SELECT COUNT(*) FROM folders c WHERE c.parent_id = f.id) as subfolder_count
			FROM folders f
			WHERE f.parent_id = ?
			ORDER BY f.sort_order, f.name COLLATE NOCASE
		`
		args = []any{*parentID}
	}

	rows, err := v.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to list folders: %w", err)
	}

	// Collect folder data first, then close rows before computing paths
	// (SQLite can deadlock if we query while iterating)
	var folders []*FolderWithStats
	for rows.Next() {
		f := &FolderWithStats{}
		var parentIDVal sql.NullString
		var icon, color sql.NullString

		if err := rows.Scan(
			&f.ID, &f.Name, &parentIDVal, &icon, &color,
			&f.SortOrder, &f.CreatedAt, &f.UpdatedAt,
			&f.SecretCount, &f.SubfolderCount,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("vault: failed to scan folder: %w", err)
		}

		if parentIDVal.Valid {
			f.ParentID = &parentIDVal.String
		}
		if icon.Valid {
			f.Icon = icon.String
		}
		if color.Valid {
			f.Color = color.String
		}

		folders = append(folders, f)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("vault: error iterating folders: %w", err)
	}
	rows.Close()

	// Now compute paths with rows closed
	for _, f := range folders {
		f.Path, _ = v.computeFolderPathLocked(f.ID)
	}

	return folders, nil
}

// UpdateFolder updates a folder's properties.
func (v *Vault) UpdateFolder(folder *Folder) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.dek == nil {
		return ErrVaultLocked
	}

	// Validate folder name
	if err := validateFolderName(folder.Name); err != nil {
		return err
	}

	// Validate color if provided
	if err := validateFolderColor(folder.Color); err != nil {
		return err
	}

	// Check folder exists
	var exists int
	err := v.db.QueryRow("SELECT COUNT(*) FROM folders WHERE id = ?", folder.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("vault: failed to check folder: %w", err)
	}
	if exists == 0 {
		return ErrFolderNotFound
	}

	// Prevent setting self as parent
	if folder.ParentID != nil && *folder.ParentID == folder.ID {
		return ErrFolderSelfParent
	}

	// Validate parent exists if specified
	if folder.ParentID != nil && *folder.ParentID != "" {
		var parentExists int
		err := v.db.QueryRow("SELECT COUNT(*) FROM folders WHERE id = ?", *folder.ParentID).Scan(&parentExists)
		if err != nil {
			return fmt.Errorf("vault: failed to check parent folder: %w", err)
		}
		if parentExists == 0 {
			return fmt.Errorf("%w: parent_id %s", ErrFolderNotFound, *folder.ParentID)
		}

		// Check for circular reference
		if v.wouldCreateCircularReference(folder.ID, *folder.ParentID) {
			return ErrFolderCircular
		}
	}

	// Update folder
	now := time.Now().UTC()
	_, err = v.db.Exec(`
		UPDATE folders
		SET name = ?, parent_id = ?, icon = ?, color = ?, sort_order = ?, updated_at = ?
		WHERE id = ?
	`, folder.Name, folder.ParentID, folder.Icon, folder.Color, folder.SortOrder, now, folder.ID)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return ErrFolderExists
		}
		return fmt.Errorf("vault: failed to update folder: %w", err)
	}

	folder.UpdatedAt = now
	return nil
}

// DeleteFolder deletes a folder by ID.
// If force is false, returns error if folder has children or secrets.
// If force is true, moves children and secrets to unfiled (folder_id = NULL).
func (v *Vault) DeleteFolder(id string, force bool) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.dek == nil {
		return ErrVaultLocked
	}

	// Check folder exists
	var exists int
	err := v.db.QueryRow("SELECT COUNT(*) FROM folders WHERE id = ?", id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("vault: failed to check folder: %w", err)
	}
	if exists == 0 {
		return ErrFolderNotFound
	}

	// Check for children
	var childCount int
	err = v.db.QueryRow("SELECT COUNT(*) FROM folders WHERE parent_id = ?", id).Scan(&childCount)
	if err != nil {
		return fmt.Errorf("vault: failed to count child folders: %w", err)
	}

	// Check for secrets
	var secretCount int
	err = v.db.QueryRow("SELECT COUNT(*) FROM secrets WHERE folder_id = ?", id).Scan(&secretCount)
	if err != nil {
		return fmt.Errorf("vault: failed to count secrets: %w", err)
	}

	if !force {
		if childCount > 0 {
			return fmt.Errorf("%w: %d subfolders", ErrFolderHasChildren, childCount)
		}
		if secretCount > 0 {
			return fmt.Errorf("%w: %d secrets", ErrFolderHasSecrets, secretCount)
		}
	}

	// Begin transaction
	tx, err := v.db.Begin()
	if err != nil {
		return fmt.Errorf("vault: failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// If force, move children to unfiled
	if force {
		// Move child folders to root (parent_id = NULL)
		_, err = tx.Exec("UPDATE folders SET parent_id = NULL WHERE parent_id = ?", id)
		if err != nil {
			return fmt.Errorf("vault: failed to move child folders: %w", err)
		}

		// Move secrets to unfiled (folder_id = NULL)
		_, err = tx.Exec("UPDATE secrets SET folder_id = NULL WHERE folder_id = ?", id)
		if err != nil {
			return fmt.Errorf("vault: failed to move secrets: %w", err)
		}
	}

	// Delete folder
	_, err = tx.Exec("DELETE FROM folders WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("vault: failed to delete folder: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("vault: failed to commit transaction: %w", err)
	}

	return nil
}

// MoveSecretToFolder moves a secret to a folder (or unfiled if folderID is nil).
func (v *Vault) MoveSecretToFolder(secretKey string, folderID *string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.dek == nil {
		return ErrVaultLocked
	}

	// Validate folder exists if specified
	if folderID != nil && *folderID != "" {
		var exists int
		err := v.db.QueryRow("SELECT COUNT(*) FROM folders WHERE id = ?", *folderID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("vault: failed to check folder: %w", err)
		}
		if exists == 0 {
			return ErrFolderNotFound
		}
	}

	// Compute key hash
	keyHash := v.hashKey(secretKey)

	// Update secret
	result, err := v.db.Exec("UPDATE secrets SET folder_id = ? WHERE key_hash = ?", folderID, keyHash)
	if err != nil {
		return fmt.Errorf("vault: failed to move secret: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("vault: failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrSecretNotFound
	}

	return nil
}

// ListSecretsInFolder returns secrets in a specific folder.
// If folderID is nil, returns unfiled secrets (folder_id IS NULL).
// If recursive is true, includes secrets from subfolders.
func (v *Vault) ListSecretsInFolder(folderID *string, recursive bool) ([]*SecretEntry, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.dek == nil {
		return nil, ErrVaultLocked
	}

	var query string
	var args []any

	if folderID == nil || *folderID == "" {
		// Unfiled secrets
		query = `
			SELECT encrypted_key, encrypted_metadata, schema, field_count, folder_id, tags, expires_at, created_at, updated_at
			FROM secrets
			WHERE folder_id IS NULL
			ORDER BY created_at
		`
	} else if recursive {
		// Secrets in folder and all subfolders (using recursive CTE)
		query = `
			WITH RECURSIVE folder_tree AS (
				SELECT id FROM folders WHERE id = ?
				UNION ALL
				SELECT f.id FROM folders f
				INNER JOIN folder_tree ft ON f.parent_id = ft.id
			)
			SELECT encrypted_key, encrypted_metadata, schema, field_count, folder_id, tags, expires_at, created_at, updated_at
			FROM secrets
			WHERE folder_id IN (SELECT id FROM folder_tree)
			ORDER BY created_at
		`
		args = []any{*folderID}
	} else {
		// Secrets directly in folder
		query = `
			SELECT encrypted_key, encrypted_metadata, schema, field_count, folder_id, tags, expires_at, created_at, updated_at
			FROM secrets
			WHERE folder_id = ?
			ORDER BY created_at
		`
		args = []any{*folderID}
	}

	rows, err := v.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to query secrets: %w", err)
	}
	defer rows.Close()

	var secrets []*SecretEntry
	for rows.Next() {
		entry, err := v.scanSecretEntryRowWithMetadata(rows)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vault: error iterating rows: %w", err)
	}

	return secrets, nil
}

// computeFolderPathLocked computes the full path for a folder.
// Must be called with lock held.
func (v *Vault) computeFolderPathLocked(folderID string) (string, error) {
	var parts []string
	currentID := folderID
	visited := make(map[string]bool)

	for i := 0; i < MaxFolderDepth+1; i++ {
		if visited[currentID] {
			return "", ErrFolderCircular
		}
		visited[currentID] = true

		var name string
		var parentID sql.NullString
		err := v.db.QueryRow("SELECT name, parent_id FROM folders WHERE id = ?", currentID).Scan(&name, &parentID)
		if err == sql.ErrNoRows {
			break
		}
		if err != nil {
			return "", fmt.Errorf("vault: failed to get folder: %w", err)
		}

		parts = append([]string{name}, parts...)

		if !parentID.Valid || parentID.String == "" {
			break
		}
		currentID = parentID.String
	}

	return strings.Join(parts, "/"), nil
}

// getFolderDepth returns the depth of a folder in the hierarchy (0 for root).
func (v *Vault) getFolderDepth(folderID string) (int, error) {
	depth := 0
	currentID := folderID
	visited := make(map[string]bool)

	for depth < MaxFolderDepth+1 {
		if visited[currentID] {
			return 0, ErrFolderCircular
		}
		visited[currentID] = true

		var parentID sql.NullString
		err := v.db.QueryRow("SELECT parent_id FROM folders WHERE id = ?", currentID).Scan(&parentID)
		if err == sql.ErrNoRows {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("vault: failed to get folder: %w", err)
		}

		if !parentID.Valid || parentID.String == "" {
			break
		}
		depth++
		currentID = parentID.String
	}

	return depth, nil
}

// wouldCreateCircularReference checks if setting newParentID as parent of folderID
// would create a circular reference.
func (v *Vault) wouldCreateCircularReference(folderID, newParentID string) bool {
	currentID := newParentID
	visited := make(map[string]bool)

	for i := 0; i < MaxFolderDepth+1; i++ {
		if currentID == folderID {
			return true
		}
		if visited[currentID] {
			return true // Already circular
		}
		visited[currentID] = true

		var parentID sql.NullString
		err := v.db.QueryRow("SELECT parent_id FROM folders WHERE id = ?", currentID).Scan(&parentID)
		if err != nil || !parentID.Valid || parentID.String == "" {
			break
		}
		currentID = parentID.String
	}

	return false
}
