//go:build windows

package audit

// checkDiskSpace on Windows returns nil as disk space checking
// is not implemented for Windows. Audit operations proceed without
// disk space verification.
func (l *Logger) checkDiskSpace() error {
	return nil
}
