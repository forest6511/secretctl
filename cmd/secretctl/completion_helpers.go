package main

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// isDynamicCompletionEnabled checks if dynamic completion is opt-in enabled.
// Dynamic completion is disabled by default to prevent vault unlock prompts
// during tab completion.
func isDynamicCompletionEnabled() bool {
	return os.Getenv("SECRETCTL_COMPLETION_ENABLED") == "1"
}

// completeSecretKeys provides secret key completion (opt-in only).
// Returns empty list if:
// - Dynamic completion is disabled (default)
// - Vault is not unlocked
func completeSecretKeys(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Only provide dynamic completion if explicitly enabled
	if !isDynamicCompletionEnabled() {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Check if vault is unlocked (don't prompt for password)
	if !isVaultUnlockedForCompletion() {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Get secret keys from vault
	keys, err := getSecretKeysForCompletion(toComplete)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return keys, cobra.ShellCompDirectiveNoFileComp
}

// completeTags provides tag completion (opt-in only).
func completeTags(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !isDynamicCompletionEnabled() {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	if !isVaultUnlockedForCompletion() {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	tags, err := getTagsForCompletion(toComplete)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return tags, cobra.ShellCompDirectiveNoFileComp
}

// isVaultUnlockedForCompletion checks if vault is currently unlocked without prompting.
// This is a conservative check - it returns false unless we can confirm the vault
// is already accessible without requiring user interaction.
//
// Note: Session management may be implemented in Phase 3. For Phase 2c-X4,
// this always returns false (safe default) to prevent password prompts during
// shell completion.
func isVaultUnlockedForCompletion() bool {
	// TODO: Phase 3 - Check for cached session/token
	// For now, return false to prevent password prompts during completion
	return false
}

// getSecretKeysForCompletion returns secret keys matching the given prefix.
// This function should only be called when isVaultUnlockedForCompletion() returns true.
func getSecretKeysForCompletion(prefix string) ([]string, error) {
	// This function is a placeholder for Phase 3 when session management is available
	// Currently isVaultUnlockedForCompletion() always returns false, so this is never called

	if v == nil {
		return nil, nil
	}

	// ListSecrets() returns []string (keys only)
	keys, err := v.ListSecrets()
	if err != nil {
		return nil, err
	}

	// Filter by prefix
	var filtered []string
	lowerPrefix := strings.ToLower(prefix)
	for _, key := range keys {
		if strings.HasPrefix(strings.ToLower(key), lowerPrefix) {
			filtered = append(filtered, key)
		}
	}

	return filtered, nil
}

// getTagsForCompletion returns tags matching the given prefix.
// This function should only be called when isVaultUnlockedForCompletion() returns true.
func getTagsForCompletion(prefix string) ([]string, error) {
	// This function is a placeholder for Phase 3 when session management is available
	// Currently isVaultUnlockedForCompletion() always returns false, so this is never called

	if v == nil {
		return nil, nil
	}

	// ListSecretsWithMetadata() returns []*SecretEntry with tags
	secrets, err := v.ListSecretsWithMetadata()
	if err != nil {
		return nil, err
	}

	// Collect unique tags
	tagSet := make(map[string]struct{})
	lowerPrefix := strings.ToLower(prefix)
	for _, secret := range secrets {
		for _, tag := range secret.Tags {
			if strings.HasPrefix(strings.ToLower(tag), lowerPrefix) {
				tagSet[tag] = struct{}{}
			}
		}
	}

	var tags []string
	for tag := range tagSet {
		tags = append(tags, tag)
	}

	return tags, nil
}

// registerCompletionFunctions registers ValidArgsFunction for commands that support
// dynamic completion.
func registerCompletionFunctions() {
	// Get command - complete secret keys
	getCmd.ValidArgsFunction = completeSecretKeys

	// Delete command - complete secret keys
	deleteCmd.ValidArgsFunction = completeSecretKeys

	// Register flag completion for --tag
	_ = listCmd.RegisterFlagCompletionFunc("tag", completeTags)
}
