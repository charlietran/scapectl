//go:build !windows

package triggers

import "os/exec"

func shellCmd(script string) *exec.Cmd {
	return exec.Command("sh", "-c", script)
}
