// Package main provides the secretctl CLI application.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/forest6511/secretctl/pkg/vault"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// Folder command flags
var (
	folderParent    string
	folderIcon      string
	folderColor     string
	folderJSON      bool
	folderRecursive bool
)

// folderCmd is the parent command for folder operations.
var folderCmd = &cobra.Command{
	Use:   "folder",
	Short: "Folder operations",
	Long: `Manage folders for organizing secrets.

Folders provide hierarchical organization for secrets. Use path syntax
(e.g., "Work/APIs") to specify nested folders.`,
}

// folderCreateCmd creates a new folder.
var folderCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new folder",
	Long: `Create a new folder for organizing secrets.

Examples:
  secretctl folder create "Work"
  secretctl folder create "APIs" --parent="Work"
  secretctl folder create "Production" --parent="Work/APIs" --icon="🚀"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		name := args[0]

		// Resolve parent folder if specified
		var parentID *string
		if folderParent != "" {
			parent, err := v.GetFolderByPath(folderParent)
			if err != nil {
				if errors.Is(err, vault.ErrFolderNotFound) || errors.Is(err, vault.ErrFolderPathNotFound) {
					return fmt.Errorf("parent folder not found: %s", folderParent)
				}
				return fmt.Errorf("failed to find parent folder: %w", err)
			}
			parentID = &parent.ID
		}

		folder := &vault.Folder{
			ID:       uuid.New().String(),
			Name:     name,
			ParentID: parentID,
			Icon:     folderIcon,
			Color:    folderColor,
		}

		if err := v.CreateFolder(folder); err != nil {
			if errors.Is(err, vault.ErrFolderExists) {
				return fmt.Errorf("folder already exists: %s", name)
			}
			if errors.Is(err, vault.ErrFolderNameInvalid) || errors.Is(err, vault.ErrFolderNameSlash) ||
				errors.Is(err, vault.ErrFolderNameTooLong) || errors.Is(err, vault.ErrFolderNameTooShort) {
				return fmt.Errorf("invalid folder name: %s", err)
			}
			return fmt.Errorf("failed to create folder: %w", err)
		}

		if folderJSON {
			output, _ := json.MarshalIndent(folder, "", "  ")
			fmt.Println(string(output))
		} else {
			path := name
			if folderParent != "" {
				path = folderParent + "/" + name
			}
			fmt.Printf("Created folder: %s (ID: %s)\n", path, folder.ID)
		}

		return nil
	},
}

// folderListCmd lists all folders.
var folderListCmd = &cobra.Command{
	Use:   "list [parent-path]",
	Short: "List folders",
	Long: `List folders in the vault.

Without arguments, lists all root folders.
With a path argument, lists children of that folder.

Examples:
  secretctl folder list
  secretctl folder list "Work"`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		var parentID *string
		if len(args) > 0 {
			parent, err := v.GetFolderByPath(args[0])
			if err != nil {
				if errors.Is(err, vault.ErrFolderNotFound) || errors.Is(err, vault.ErrFolderPathNotFound) {
					return fmt.Errorf("folder not found: %s", args[0])
				}
				return fmt.Errorf("failed to find folder: %w", err)
			}
			parentID = &parent.ID
		}

		folders, err := v.ListFolders(parentID)
		if err != nil {
			return fmt.Errorf("failed to list folders: %w", err)
		}

		if len(folders) == 0 {
			if !folderJSON {
				fmt.Println("No folders found.")
			} else {
				fmt.Println("[]")
			}
			return nil
		}

		if folderJSON {
			output, _ := json.MarshalIndent(folders, "", "  ")
			fmt.Println(string(output))
		} else {
			for _, f := range folders {
				icon := ""
				if f.Icon != "" {
					icon = f.Icon + " "
				}
				stats := ""
				if f.SecretCount > 0 {
					stats = fmt.Sprintf(" (%d secrets)", f.SecretCount)
				}
				fmt.Printf("%s%s%s\n", icon, f.Path, stats)
			}
		}

		return nil
	},
}

// folderDeleteCmd deletes a folder.
var folderDeleteCmd = &cobra.Command{
	Use:   "delete <path>",
	Short: "Delete a folder",
	Long: `Delete a folder from the vault.

The folder must be empty (no secrets or subfolders) unless --force is specified.
With --force, secrets are moved to unfiled and subfolders are deleted recursively.

Examples:
  secretctl folder delete "Work/APIs"
  secretctl folder delete "Work" --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		folderPath := args[0]

		folder, err := v.GetFolderByPath(folderPath)
		if err != nil {
			if errors.Is(err, vault.ErrFolderNotFound) || errors.Is(err, vault.ErrFolderPathNotFound) {
				return fmt.Errorf("folder not found: %s", folderPath)
			}
			return fmt.Errorf("failed to find folder: %w", err)
		}

		if err := v.DeleteFolder(folder.ID, folderRecursive); err != nil {
			if errors.Is(err, vault.ErrFolderHasChildren) || errors.Is(err, vault.ErrFolderHasSecrets) {
				return fmt.Errorf("folder is not empty (use --force to delete with contents)")
			}
			return fmt.Errorf("failed to delete folder: %w", err)
		}

		fmt.Printf("Deleted folder: %s\n", folderPath)
		return nil
	},
}

// folderRenameCmd renames a folder.
var folderRenameCmd = &cobra.Command{
	Use:   "rename <path> <new-name>",
	Short: "Rename a folder",
	Long: `Rename a folder in the vault.

Examples:
  secretctl folder rename "Work" "Professional"
  secretctl folder rename "Work/APIs" "Services"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		folderPath := args[0]
		newName := args[1]

		folder, err := v.GetFolderByPath(folderPath)
		if err != nil {
			if errors.Is(err, vault.ErrFolderNotFound) || errors.Is(err, vault.ErrFolderPathNotFound) {
				return fmt.Errorf("folder not found: %s", folderPath)
			}
			return fmt.Errorf("failed to find folder: %w", err)
		}

		// Update the folder name
		folder.Name = newName
		if err := v.UpdateFolder(folder); err != nil {
			if errors.Is(err, vault.ErrFolderExists) {
				return fmt.Errorf("a folder with name %q already exists in the same location", newName)
			}
			if errors.Is(err, vault.ErrFolderNameInvalid) || errors.Is(err, vault.ErrFolderNameSlash) ||
				errors.Is(err, vault.ErrFolderNameTooLong) || errors.Is(err, vault.ErrFolderNameTooShort) {
				return fmt.Errorf("invalid folder name: %s", err)
			}
			return fmt.Errorf("failed to rename folder: %w", err)
		}

		fmt.Printf("Renamed folder: %s -> %s\n", folderPath, newName)
		return nil
	},
}

// folderMoveCmd moves a folder to a new parent.
var folderMoveCmd = &cobra.Command{
	Use:   "move <path> <new-parent-path>",
	Short: "Move a folder to a new location",
	Long: `Move a folder to a new parent folder.

Use empty string "" or "/" to move to root level.

Examples:
  secretctl folder move "Work/APIs" "Personal"
  secretctl folder move "Work/APIs" ""`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		folderPath := args[0]
		newParentPath := args[1]

		folder, err := v.GetFolderByPath(folderPath)
		if err != nil {
			if errors.Is(err, vault.ErrFolderNotFound) || errors.Is(err, vault.ErrFolderPathNotFound) {
				return fmt.Errorf("folder not found: %s", folderPath)
			}
			return fmt.Errorf("failed to find folder: %w", err)
		}

		// Resolve new parent
		var newParentID *string
		if newParentPath != "" && newParentPath != "/" {
			newParent, err := v.GetFolderByPath(newParentPath)
			if err != nil {
				if errors.Is(err, vault.ErrFolderNotFound) || errors.Is(err, vault.ErrFolderPathNotFound) {
					return fmt.Errorf("target folder not found: %s", newParentPath)
				}
				return fmt.Errorf("failed to find target folder: %w", err)
			}
			newParentID = &newParent.ID
		}

		// Update the folder parent
		folder.ParentID = newParentID
		if err := v.UpdateFolder(folder); err != nil {
			if errors.Is(err, vault.ErrFolderExists) {
				return fmt.Errorf("a folder with name %q already exists in the target location", folder.Name)
			}
			if errors.Is(err, vault.ErrFolderCircular) {
				return errors.New("cannot move folder into its own subtree")
			}
			return fmt.Errorf("failed to move folder: %w", err)
		}

		if newParentPath == "" || newParentPath == "/" {
			fmt.Printf("Moved folder: %s -> / (root)\n", folderPath)
		} else {
			fmt.Printf("Moved folder: %s -> %s/%s\n", folderPath, newParentPath, folder.Name)
		}
		return nil
	},
}

// folderInfoCmd shows detailed information about a folder.
var folderInfoCmd = &cobra.Command{
	Use:   "info <path>",
	Short: "Show folder information",
	Long: `Show detailed information about a folder.

Examples:
  secretctl folder info "Work"
  secretctl folder info "Work/APIs" --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureUnlocked(); err != nil {
			return err
		}
		defer v.Lock()

		folderPath := args[0]

		folder, err := v.GetFolderByPath(folderPath)
		if err != nil {
			if errors.Is(err, vault.ErrFolderNotFound) || errors.Is(err, vault.ErrFolderPathNotFound) {
				return fmt.Errorf("folder not found: %s", folderPath)
			}
			return fmt.Errorf("failed to find folder: %w", err)
		}

		// Get folder stats by listing this folder's parent and finding it
		var parentID *string
		if folder.ParentID != nil {
			parentID = folder.ParentID
		}
		allFolders, err := v.ListFolders(parentID)
		if err != nil {
			return fmt.Errorf("failed to get folder stats: %w", err)
		}

		// Find the matching folder in the list to get stats
		var stats *vault.FolderWithStats
		for _, f := range allFolders {
			if f.ID == folder.ID {
				stats = f
				break
			}
		}

		if stats == nil {
			// Fallback: create stats without counts
			folder.Path = folderPath
			stats = &vault.FolderWithStats{
				Folder: *folder,
			}
		}

		if folderJSON {
			output, _ := json.MarshalIndent(stats, "", "  ")
			fmt.Println(string(output))
		} else {
			fmt.Printf("Name:         %s\n", stats.Name)
			fmt.Printf("Path:         %s\n", stats.Path)
			fmt.Printf("ID:           %s\n", stats.ID)
			if stats.Icon != "" {
				fmt.Printf("Icon:         %s\n", stats.Icon)
			}
			if stats.Color != "" {
				fmt.Printf("Color:        %s\n", stats.Color)
			}
			fmt.Printf("Secrets:      %d\n", stats.SecretCount)
			fmt.Printf("Subfolders:   %d\n", stats.SubfolderCount)
			fmt.Printf("Created:      %s\n", stats.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated:      %s\n", stats.UpdatedAt.Format("2006-01-02 15:04:05"))
		}

		return nil
	},
}

// Helper function to parse folder path from flags
func resolveFolderFromFlags() (*string, error) {
	// Check if --folder-id is specified (takes precedence)
	if setFolderID != "" {
		return &setFolderID, nil
	}

	// Check if --folder is specified
	if setFolder != "" {
		folder, err := v.GetFolderByPath(setFolder)
		if err != nil {
			if errors.Is(err, vault.ErrFolderNotFound) || errors.Is(err, vault.ErrFolderPathNotFound) {
				return nil, fmt.Errorf("folder not found: %s", setFolder)
			}
			return nil, fmt.Errorf("failed to find folder: %w", err)
		}
		return &folder.ID, nil
	}

	return nil, nil
}

// formatFolderPath formats a folder path for display
func formatFolderPath(folderID *string) string {
	if folderID == nil {
		return "(unfiled)"
	}

	folder, err := v.GetFolder(*folderID)
	if err != nil {
		return fmt.Sprintf("(folder: %s)", (*folderID)[:8])
	}

	// Build path
	var parts []string
	current := folder
	for current != nil {
		parts = append([]string{current.Name}, parts...)
		if current.ParentID == nil {
			break
		}
		parent, err := v.GetFolder(*current.ParentID)
		if err != nil {
			break
		}
		current = parent
	}

	return strings.Join(parts, "/")
}

func init() {
	// Add folder subcommands
	folderCmd.AddCommand(folderCreateCmd)
	folderCmd.AddCommand(folderListCmd)
	folderCmd.AddCommand(folderDeleteCmd)
	folderCmd.AddCommand(folderRenameCmd)
	folderCmd.AddCommand(folderMoveCmd)
	folderCmd.AddCommand(folderInfoCmd)

	// Create command flags
	folderCreateCmd.Flags().StringVar(&folderParent, "parent", "", "Parent folder path")
	folderCreateCmd.Flags().StringVar(&folderIcon, "icon", "", "Folder icon (emoji)")
	folderCreateCmd.Flags().StringVar(&folderColor, "color", "", "Folder color (hex code)")
	folderCreateCmd.Flags().BoolVar(&folderJSON, "json", false, "Output as JSON")

	// List command flags
	folderListCmd.Flags().BoolVar(&folderJSON, "json", false, "Output as JSON")

	// Delete command flags
	folderDeleteCmd.Flags().BoolVarP(&folderRecursive, "force", "f", false, "Force delete folder and all contents")

	// Info command flags
	folderInfoCmd.Flags().BoolVar(&folderJSON, "json", false, "Output as JSON")
}
