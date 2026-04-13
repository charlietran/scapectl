//go:build !windows

package tray

import "os/exec"

func hideConsole(_ *exec.Cmd) {}
