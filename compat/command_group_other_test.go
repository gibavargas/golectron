//go:build !darwin && !linux

package compat_test

import "os/exec"

func prepareCommandForCleanup(cmd *exec.Cmd) {}

func terminateCommandGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
