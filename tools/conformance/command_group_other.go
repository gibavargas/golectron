//go:build !darwin && !linux

package main

import "os/exec"

func prepareCommandForCleanup(cmd *exec.Cmd) {}

func terminateCommandGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
