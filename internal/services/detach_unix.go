//go:build !windows

package services

import (
	"os/exec"
	"syscall"
)

// detach starts cmd in a session of its own: signals to this process's
// group (the terminal closing, Ctrl+C) do not reach it.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
