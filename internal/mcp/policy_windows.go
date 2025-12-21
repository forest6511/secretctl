//go:build windows

package mcp

import (
	"os"
)

// openPolicyFile opens the policy file on Windows.
// Windows doesn't have O_NOFOLLOW, but symlinks are less common on Windows
// and require special privileges to create.
func openPolicyFile(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrPolicyNotFound
		}
		return nil, err
	}
	return f, nil
}

// checkFileOwnership on Windows is a no-op.
// Windows uses ACLs for file ownership which requires different handling.
// The permission check (0600 equivalent) is the primary security control.
func checkFileOwnership(_ os.FileInfo) error {
	// Windows uses different security model (ACLs)
	// Skip ownership check, rely on file permissions
	return nil
}
