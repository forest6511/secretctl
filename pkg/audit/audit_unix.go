//go:build !windows

package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// checkDiskSpace verifies sufficient disk space for audit log writes
func (l *Logger) checkDiskSpace() error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(l.path, &stat); err != nil {
		// If audit directory doesn't exist yet, check parent
		parentDir := filepath.Dir(l.path)
		if err := syscall.Statfs(parentDir, &stat); err != nil {
			// Log warning but don't block audit operation
			fmt.Fprintf(os.Stderr, "warning: failed to check disk space for audit: %v\n", err)
			return nil
		}
	}

	available := stat.Bavail * uint64(stat.Bsize)
	if available < MinAuditDiskSpace {
		return fmt.Errorf("audit: insufficient disk space: only %d bytes available, need at least %d",
			available, MinAuditDiskSpace)
	}

	return nil
}
