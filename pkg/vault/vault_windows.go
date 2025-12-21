//go:build windows

package vault

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// CheckDiskSpace returns disk space information for the vault directory
func (v *Vault) CheckDiskSpace() (*DiskSpaceInfo, error) {
	path := v.path
	// Use parent directory if vault path doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = filepath.Dir(path)
	}

	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to convert path: %w", err)
	}

	err = windows.GetDiskFreeSpaceEx(pathPtr, &freeBytesAvailable, &totalBytes, &totalFreeBytes)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to get disk stats: %w", err)
	}

	usedPct := 0
	if totalBytes > 0 {
		usedPct = int(100 * (totalBytes - totalFreeBytes) / totalBytes)
	}

	return &DiskSpaceInfo{
		Total:     totalBytes,
		Free:      totalFreeBytes,
		Available: freeBytesAvailable,
		UsedPct:   usedPct,
	}, nil
}
