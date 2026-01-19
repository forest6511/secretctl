package vault

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestVault creates a temporary vault for testing
func setupTestVaultForFolder(t *testing.T) (*Vault, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "vault-folder-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	vaultPath := filepath.Join(tmpDir, "vault")
	v := New(vaultPath)

	// Initialize vault with password
	if err := v.Init("testpassword123"); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to initialize vault: %v", err)
	}

	// Unlock vault
	if err := v.Unlock("testpassword123"); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to unlock vault: %v", err)
	}

	cleanup := func() {
		v.Lock()
		os.RemoveAll(tmpDir)
	}

	return v, tmpDir, cleanup
}

func TestCreateFolder(t *testing.T) {
	v, _, cleanup := setupTestVaultForFolder(t)
	defer cleanup()

	t.Run("create root folder", func(t *testing.T) {
		folder := &Folder{
			ID:   "test-id-1",
			Name: "Work",
		}
		if err := v.CreateFolder(folder); err != nil {
			t.Fatalf("failed to create folder: %v", err)
		}

		// Verify folder exists
		got, err := v.GetFolder("test-id-1")
		if err != nil {
			t.Fatalf("failed to get folder: %v", err)
		}
		if got.Name != "Work" {
			t.Errorf("expected name 'Work', got '%s'", got.Name)
		}
		if got.ParentID != nil {
			t.Errorf("expected nil parent, got '%v'", got.ParentID)
		}
	})

	t.Run("create nested folder", func(t *testing.T) {
		parentID := "test-id-1"
		folder := &Folder{
			ID:       "test-id-2",
			Name:     "APIs",
			ParentID: &parentID,
		}
		if err := v.CreateFolder(folder); err != nil {
			t.Fatalf("failed to create nested folder: %v", err)
		}

		// Verify folder exists
		got, err := v.GetFolder("test-id-2")
		if err != nil {
			t.Fatalf("failed to get folder: %v", err)
		}
		if got.Name != "APIs" {
			t.Errorf("expected name 'APIs', got '%s'", got.Name)
		}
		if got.ParentID == nil || *got.ParentID != parentID {
			t.Errorf("expected parent '%s', got '%v'", parentID, got.ParentID)
		}
	})

	t.Run("create with icon and color", func(t *testing.T) {
		folder := &Folder{
			ID:    "test-id-3",
			Name:  "Personal",
			Icon:  "🏠",
			Color: "#3B82F6",
		}
		if err := v.CreateFolder(folder); err != nil {
			t.Fatalf("failed to create folder: %v", err)
		}

		got, err := v.GetFolder("test-id-3")
		if err != nil {
			t.Fatalf("failed to get folder: %v", err)
		}
		if got.Icon != "🏠" {
			t.Errorf("expected icon '🏠', got '%s'", got.Icon)
		}
		if got.Color != "#3B82F6" {
			t.Errorf("expected color '#3B82F6', got '%s'", got.Color)
		}
	})
}

func TestCreateFolderValidation(t *testing.T) {
	v, _, cleanup := setupTestVaultForFolder(t)
	defer cleanup()

	t.Run("name with slash rejected", func(t *testing.T) {
		folder := &Folder{
			ID:   "test-slash",
			Name: "Work/APIs",
		}
		err := v.CreateFolder(folder)
		if err == nil {
			t.Fatal("expected error for name with slash")
		}
	})

	t.Run("empty name rejected", func(t *testing.T) {
		folder := &Folder{
			ID:   "test-empty",
			Name: "",
		}
		err := v.CreateFolder(folder)
		if err == nil {
			t.Fatal("expected error for empty name")
		}
	})

	t.Run("name too long rejected", func(t *testing.T) {
		longName := make([]byte, 300)
		for i := range longName {
			longName[i] = 'a'
		}
		folder := &Folder{
			ID:   "test-long",
			Name: string(longName),
		}
		err := v.CreateFolder(folder)
		if err == nil {
			t.Fatal("expected error for name too long")
		}
	})

	t.Run("duplicate name at same level rejected", func(t *testing.T) {
		folder1 := &Folder{ID: "dup-1", Name: "Duplicate"}
		if err := v.CreateFolder(folder1); err != nil {
			t.Fatalf("failed to create first folder: %v", err)
		}

		folder2 := &Folder{ID: "dup-2", Name: "Duplicate"}
		err := v.CreateFolder(folder2)
		if err == nil {
			t.Fatal("expected error for duplicate name")
		}
	})

	t.Run("same name at different levels allowed", func(t *testing.T) {
		parentID := "dup-1"
		folder := &Folder{
			ID:       "dup-child",
			Name:     "Duplicate",
			ParentID: &parentID,
		}
		if err := v.CreateFolder(folder); err != nil {
			t.Fatalf("same name at different level should be allowed: %v", err)
		}
	})
}

func TestGetFolderByPath(t *testing.T) {
	v, _, cleanup := setupTestVaultForFolder(t)
	defer cleanup()

	// Create folder hierarchy: Work/APIs/Production
	work := &Folder{ID: "work-id", Name: "Work"}
	if err := v.CreateFolder(work); err != nil {
		t.Fatalf("failed to create Work: %v", err)
	}

	workID := "work-id"
	apis := &Folder{ID: "apis-id", Name: "APIs", ParentID: &workID}
	if err := v.CreateFolder(apis); err != nil {
		t.Fatalf("failed to create APIs: %v", err)
	}

	apisID := "apis-id"
	prod := &Folder{ID: "prod-id", Name: "Production", ParentID: &apisID}
	if err := v.CreateFolder(prod); err != nil {
		t.Fatalf("failed to create Production: %v", err)
	}

	t.Run("get root folder", func(t *testing.T) {
		folder, err := v.GetFolderByPath("Work")
		if err != nil {
			t.Fatalf("failed to get folder: %v", err)
		}
		if folder.ID != "work-id" {
			t.Errorf("expected 'work-id', got '%s'", folder.ID)
		}
	})

	t.Run("get nested folder", func(t *testing.T) {
		folder, err := v.GetFolderByPath("Work/APIs")
		if err != nil {
			t.Fatalf("failed to get folder: %v", err)
		}
		if folder.ID != "apis-id" {
			t.Errorf("expected 'apis-id', got '%s'", folder.ID)
		}
	})

	t.Run("get deeply nested folder", func(t *testing.T) {
		folder, err := v.GetFolderByPath("Work/APIs/Production")
		if err != nil {
			t.Fatalf("failed to get folder: %v", err)
		}
		if folder.ID != "prod-id" {
			t.Errorf("expected 'prod-id', got '%s'", folder.ID)
		}
	})

	t.Run("non-existent path returns error", func(t *testing.T) {
		_, err := v.GetFolderByPath("NonExistent")
		if err == nil {
			t.Fatal("expected error for non-existent path")
		}
	})

	t.Run("partial path returns error", func(t *testing.T) {
		_, err := v.GetFolderByPath("Work/NonExistent")
		if err == nil {
			t.Fatal("expected error for partial path")
		}
	})
}

func TestListFolders(t *testing.T) {
	v, _, cleanup := setupTestVaultForFolder(t)
	defer cleanup()

	// Create folder hierarchy
	work := &Folder{ID: "work-id", Name: "Work"}
	personal := &Folder{ID: "personal-id", Name: "Personal"}
	v.CreateFolder(work)
	v.CreateFolder(personal)

	workID := "work-id"
	apis := &Folder{ID: "apis-id", Name: "APIs", ParentID: &workID}
	v.CreateFolder(apis)

	t.Run("list root folders", func(t *testing.T) {
		// Pass empty string pointer for root folders only
		emptyStr := ""
		folders, err := v.ListFolders(&emptyStr)
		if err != nil {
			t.Fatalf("failed to list folders: %v", err)
		}
		if len(folders) != 2 {
			t.Errorf("expected 2 root folders, got %d", len(folders))
		}
	})

	t.Run("list children of folder", func(t *testing.T) {
		folders, err := v.ListFolders(&workID)
		if err != nil {
			t.Fatalf("failed to list folders: %v", err)
		}
		if len(folders) != 1 {
			t.Errorf("expected 1 child folder, got %d", len(folders))
		}
		if folders[0].Name != "APIs" {
			t.Errorf("expected 'APIs', got '%s'", folders[0].Name)
		}
	})

	t.Run("list includes path", func(t *testing.T) {
		folders, err := v.ListFolders(&workID)
		if err != nil {
			t.Fatalf("failed to list folders: %v", err)
		}
		if folders[0].Path != "Work/APIs" {
			t.Errorf("expected path 'Work/APIs', got '%s'", folders[0].Path)
		}
	})
}

func TestUpdateFolder(t *testing.T) {
	v, _, cleanup := setupTestVaultForFolder(t)
	defer cleanup()

	// Create folder
	folder := &Folder{ID: "update-id", Name: "Original"}
	v.CreateFolder(folder)

	t.Run("rename folder", func(t *testing.T) {
		folder.Name = "Renamed"
		if err := v.UpdateFolder(folder); err != nil {
			t.Fatalf("failed to update folder: %v", err)
		}

		got, _ := v.GetFolder("update-id")
		if got.Name != "Renamed" {
			t.Errorf("expected 'Renamed', got '%s'", got.Name)
		}
	})

	t.Run("update icon and color", func(t *testing.T) {
		folder.Icon = "📁"
		folder.Color = "#FF0000"
		if err := v.UpdateFolder(folder); err != nil {
			t.Fatalf("failed to update folder: %v", err)
		}

		got, _ := v.GetFolder("update-id")
		if got.Icon != "📁" {
			t.Errorf("expected icon '📁', got '%s'", got.Icon)
		}
		if got.Color != "#FF0000" {
			t.Errorf("expected color '#FF0000', got '%s'", got.Color)
		}
	})
}

func TestDeleteFolder(t *testing.T) {
	v, _, cleanup := setupTestVaultForFolder(t)
	defer cleanup()

	t.Run("delete empty folder", func(t *testing.T) {
		folder := &Folder{ID: "delete-empty", Name: "ToDelete"}
		v.CreateFolder(folder)

		if err := v.DeleteFolder("delete-empty", false); err != nil {
			t.Fatalf("failed to delete folder: %v", err)
		}

		_, err := v.GetFolder("delete-empty")
		if err == nil {
			t.Fatal("folder should be deleted")
		}
	})

	t.Run("delete folder with secrets fails without force", func(t *testing.T) {
		folder := &Folder{ID: "delete-secrets", Name: "WithSecrets"}
		v.CreateFolder(folder)

		// Add a secret to the folder
		folderID := "delete-secrets"
		entry := &SecretEntry{Value: []byte("test"), FolderID: &folderID}
		v.SetSecret("test-key", entry)

		err := v.DeleteFolder("delete-secrets", false)
		if err == nil {
			t.Fatal("expected error when deleting folder with secrets")
		}
	})

	t.Run("delete folder with secrets succeeds with force", func(t *testing.T) {
		folder := &Folder{ID: "delete-force", Name: "WithSecretsForce"}
		v.CreateFolder(folder)

		folderID := "delete-force"
		entry := &SecretEntry{Value: []byte("test2"), FolderID: &folderID}
		v.SetSecret("test-key-2", entry)

		if err := v.DeleteFolder("delete-force", true); err != nil {
			t.Fatalf("failed to delete folder with force: %v", err)
		}

		// Secret should be unfiled
		got, _ := v.GetSecret("test-key-2")
		if got.FolderID != nil {
			t.Errorf("secret should be unfiled after folder deletion")
		}
	})

	t.Run("delete folder with children fails without force", func(t *testing.T) {
		parent := &Folder{ID: "delete-parent", Name: "Parent"}
		v.CreateFolder(parent)

		parentID := "delete-parent"
		child := &Folder{ID: "delete-child", Name: "Child", ParentID: &parentID}
		v.CreateFolder(child)

		err := v.DeleteFolder("delete-parent", false)
		if err == nil {
			t.Fatal("expected error when deleting folder with children")
		}
	})
}

func TestMoveSecretToFolder(t *testing.T) {
	v, _, cleanup := setupTestVaultForFolder(t)
	defer cleanup()

	// Create folder
	folder := &Folder{ID: "move-folder", Name: "Target"}
	v.CreateFolder(folder)

	// Create secret
	entry := &SecretEntry{Value: []byte("secret")}
	v.SetSecret("move-test", entry)

	t.Run("move secret to folder", func(t *testing.T) {
		folderID := "move-folder"
		if err := v.MoveSecretToFolder("move-test", &folderID); err != nil {
			t.Fatalf("failed to move secret: %v", err)
		}

		got, _ := v.GetSecret("move-test")
		if got.FolderID == nil || *got.FolderID != "move-folder" {
			t.Errorf("secret should be in folder 'move-folder'")
		}
	})

	t.Run("unfile secret", func(t *testing.T) {
		if err := v.MoveSecretToFolder("move-test", nil); err != nil {
			t.Fatalf("failed to unfile secret: %v", err)
		}

		got, _ := v.GetSecret("move-test")
		if got.FolderID != nil {
			t.Errorf("secret should be unfiled")
		}
	})
}

func TestListSecretsInFolder(t *testing.T) {
	v, _, cleanup := setupTestVaultForFolder(t)
	defer cleanup()

	// Create folder hierarchy
	work := &Folder{ID: "list-work", Name: "Work"}
	v.CreateFolder(work)

	workID := "list-work"
	apis := &Folder{ID: "list-apis", Name: "APIs", ParentID: &workID}
	v.CreateFolder(apis)

	// Add secrets to different folders
	apisID := "list-apis"
	entry1 := &SecretEntry{Value: []byte("s1"), FolderID: &workID}
	v.SetSecret("work-secret", entry1)

	entry2 := &SecretEntry{Value: []byte("s2"), FolderID: &apisID}
	v.SetSecret("api-secret", entry2)

	entry3 := &SecretEntry{Value: []byte("s3")}
	v.SetSecret("unfiled-secret", entry3)

	t.Run("list secrets in folder", func(t *testing.T) {
		entries, err := v.ListSecretsInFolder(&workID, false)
		if err != nil {
			t.Fatalf("failed to list secrets: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("expected 1 secret, got %d", len(entries))
		}
		if entries[0].Key != "work-secret" {
			t.Errorf("expected 'work-secret', got '%s'", entries[0].Key)
		}
	})

	t.Run("list secrets recursively", func(t *testing.T) {
		entries, err := v.ListSecretsInFolder(&workID, true)
		if err != nil {
			t.Fatalf("failed to list secrets: %v", err)
		}
		if len(entries) != 2 {
			t.Errorf("expected 2 secrets recursively, got %d", len(entries))
		}
	})

	t.Run("list unfiled secrets", func(t *testing.T) {
		entries, err := v.ListSecretsInFolder(nil, false)
		if err != nil {
			t.Fatalf("failed to list unfiled secrets: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("expected 1 unfiled secret, got %d", len(entries))
		}
		if entries[0].Key != "unfiled-secret" {
			t.Errorf("expected 'unfiled-secret', got '%s'", entries[0].Key)
		}
	})
}

func TestFolderDepthLimit(t *testing.T) {
	v, _, cleanup := setupTestVaultForFolder(t)
	defer cleanup()

	// Create folder hierarchy up to max depth (10)
	var lastID string
	for i := 0; i < MaxFolderDepth; i++ {
		folder := &Folder{
			ID:   string(rune('a' + i)),
			Name: string(rune('A' + i)),
		}
		if lastID != "" {
			folder.ParentID = &lastID
		}
		if err := v.CreateFolder(folder); err != nil {
			t.Fatalf("failed to create folder at depth %d: %v", i, err)
		}
		lastID = folder.ID
	}

	// Try to create one more level (should fail)
	folder := &Folder{
		ID:       "too-deep",
		Name:     "TooDeep",
		ParentID: &lastID,
	}
	err := v.CreateFolder(folder)
	if err == nil {
		t.Fatal("expected error when exceeding max folder depth")
	}
}
