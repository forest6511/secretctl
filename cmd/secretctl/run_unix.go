//go:build !windows

package main

import (
	"os"
	"syscall"
)

// signalsToNotify returns the signals that should be forwarded to child processes
func signalsToNotify() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}
}

// terminateSignal returns the signal to send for graceful termination
func terminateSignal() os.Signal {
	return syscall.SIGTERM
}

// disableCoreDumps sets RLIMIT_CORE to 0 to prevent core dumps
func disableCoreDumps() error {
	var rLimit syscall.Rlimit
	rLimit.Cur = 0
	rLimit.Max = 0
	return syscall.Setrlimit(syscall.RLIMIT_CORE, &rLimit)
}
