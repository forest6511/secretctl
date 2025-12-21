//go:build !windows

package mcp

import (
	"errors"
	"os"
	"syscall"
)

// openPolicyFile opens the policy file with O_NOFOLLOW to reject symlinks
func openPolicyFile(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrPolicyNotFound
		}
		if os.IsPermission(err) || errors.Is(err, syscall.ELOOP) {
			return nil, ErrPolicySymlink
		}
		return nil, err
	}
	return f, nil
}

// checkFileOwnership verifies the file is owned by the current user
func checkFileOwnership(info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if ok {
		if stat.Uid != uint32(os.Getuid()) {
			return ErrPolicyNotOwnedByUser
		}
	}
	return nil
}
