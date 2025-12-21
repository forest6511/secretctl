//go:build windows

package main

import "os"

// signalsToNotify returns the signals that should be forwarded to child processes
// On Windows, only os.Interrupt is available (Ctrl+C)
func signalsToNotify() []os.Signal {
	return []os.Signal{os.Interrupt}
}

// terminateSignal returns the signal to send for graceful termination
// On Windows, os.Kill is used as there's no SIGTERM equivalent
func terminateSignal() os.Signal {
	return os.Kill
}

// disableCoreDumps is a no-op on Windows.
// Windows uses different mechanisms for crash dumps (WER).
func disableCoreDumps() error {
	// Windows Error Reporting handles crash dumps differently
	// and doesn't use RLIMIT_CORE. This is a security best-effort.
	return nil
}
