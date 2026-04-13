//go:build !windows

package triggers

import "os/exec"

func hideConsole(_ *exec.Cmd) {}
