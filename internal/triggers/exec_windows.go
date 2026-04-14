//go:build windows

package triggers

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// shellCmd builds a command that runs script via cmd.exe.
// Uses SysProcAttr.CmdLine to pass the command line verbatim —
// Go's default argument escaping is incompatible with cmd.exe's quoting rules.
func shellCmd(script string) *exec.Cmd {
	cmd := exec.Command("cmd")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine:       `cmd /C ` + script,
		CreationFlags: createNoWindow,
	}
	return cmd
}
