//go:build !windows

package vault

import (
	"fmt"
	"path/filepath"
	"syscall"
)

// CheckDiskSpace returns disk space information for the vault directory
func (v *Vault) CheckDiskSpace() (*DiskSpaceInfo, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(v.path, &stat); err != nil {
		// If vault directory doesn't exist yet, check parent
		parentDir := filepath.Dir(v.path)
		if err := syscall.Statfs(parentDir, &stat); err != nil {
			return nil, fmt.Errorf("vault: failed to get disk stats: %w", err)
		}
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)

	usedPct := 0
	if total > 0 {
		usedPct = int(100 * (total - free) / total)
	}

	return &DiskSpaceInfo{
		Total:     total,
		Free:      free,
		Available: available,
		UsedPct:   usedPct,
	}, nil
}
