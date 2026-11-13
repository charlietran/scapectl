//go:build !windows

package tray

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestRestartSurvivesProcessGroupCleanup(t *testing.T) {
	if os.Getenv("SCAPECTL_TEST_RESTART_HELPER") == "1" {
		child := exec.Command("/bin/sh", "-c", `printf 'ready\n'; read -r line && printf '%s\n' "$line"`)
		child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := startRestart(child); err != nil {
			t.Fatal(err)
		}
		if err := child.Wait(); err != nil {
			t.Fatal(err)
		}
		return
	}

	parent := exec.Command(os.Args[0], "-test.run=^TestRestartSurvivesProcessGroupCleanup$")
	parent.Env = append(os.Environ(), "SCAPECTL_TEST_RESTART_HELPER=1")
	parent.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	input, err := parent.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	output, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	defer writer.Close()
	parent.Stdout, parent.Stderr = writer, os.Stderr
	if err := parent.Start(); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	t.Cleanup(func() {
		input.Close()
		parent.Process.Kill()
		parent.Wait()
	})

	if err := output.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(output)
	if line, err := reader.ReadString('\n'); err != nil || line != "ready\n" {
		t.Fatalf("replacement did not start: %q, %v", line, err)
	}

	// Match launchd's cleanup of a login job without AbandonProcessGroup.
	if err := syscall.Kill(-parent.Process.Pid, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprintln(input, "survived"); err != nil {
		t.Fatalf("replacement stopped with the old process group: %v", err)
	}
	if line, err := reader.ReadString('\n'); err != nil || line != "survived\n" {
		t.Fatalf("replacement did not survive cleanup: %q, %v", line, err)
	}
}
