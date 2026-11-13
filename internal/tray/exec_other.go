//go:build !windows

package tray

import (
	"os/exec"
	"syscall"
)

func hideConsole(_ *exec.Cmd) {}

func startRestart(cmd *exec.Cmd) error {
	// launchd cleans up the old process group when the app exits.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd.Start()
}
