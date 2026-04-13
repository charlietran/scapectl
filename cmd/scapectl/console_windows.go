//go:build windows

package main

import (
	"log"
	"os"
	"syscall"
)

var (
	modkernel32       = syscall.NewLazyDLL("kernel32.dll")
	procAttachConsole = modkernel32.NewProc("AttachConsole")
)

func init() {
	// When built with -H windowsgui the process has no console.
	// Reattach to the parent's console (e.g. cmd.exe or PowerShell)
	// so CLI subcommands can print to the terminal.
	const attachParentProcess = ^uint32(0) // ATTACH_PARENT_PROCESS
	r, _, _ := procAttachConsole.Call(uintptr(attachParentProcess))
	if r == 0 {
		return // no parent console (double-clicked) — nothing to attach
	}

	// Reopen stdout/stderr to the newly attached console.
	con, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	os.Stdout = con
	os.Stderr = con
	log.SetOutput(con)
}
