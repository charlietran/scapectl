package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiagnosticLogRetention(t *testing.T) {
	for _, tc := range []struct {
		name    string
		initial string
		entries []string
		current string
		backup  string
	}{
		{"append existing", "old\n", []string{"new\n"}, "old\nnew\n", ""},
		{"rotate at boundary", "old\n", []string{"new\n", "next\n"}, "next\n", "old\nnew\n"},
		{"replace backup", "old\n", []string{"second\n", "third\n"}, "third\n", "second\n"},
		{"oversized entry", "old\n", []string{"0123456789"}, "23456789", "old\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "scapectl.log")
			if err := os.WriteFile(path, []byte(tc.initial), 0600); err != nil {
				t.Fatal(err)
			}
			writer := &diagnosticLog{path: path, limit: 8}
			defer func() {
				if writer.file != nil {
					writer.file.Close()
				}
			}()
			for _, entry := range tc.entries {
				if n, err := writer.Write([]byte(entry)); err != nil || n != len(entry) {
					t.Fatalf("Write: n=%d err=%v", n, err)
				}
			}
			for _, file := range []struct{ path, want string }{{path, tc.current}, {path + ".1", tc.backup}} {
				got, err := os.ReadFile(file.path)
				if file.want == "" && os.IsNotExist(err) {
					continue
				}
				if err != nil || string(got) != file.want {
					t.Fatalf("%s: got %q, want %q, err=%v", file.path, got, file.want, err)
				}
			}
		})
	}
}

func TestDiagnosticsPersistWhenConsoleIsClosed(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CACHE_HOME", dir)
	t.Setenv("LocalAppData", dir)
	console, err := os.CreateTemp(dir, "console")
	if err != nil {
		t.Fatal(err)
	}
	console.Close()
	previous := log.Writer()
	log.SetOutput(console)
	defer log.SetOutput(previous)
	closeLog, err := startDiagnostics()
	if err != nil {
		t.Fatal(err)
	}
	log.Print("headset diagnostic with no console")
	closeLog()
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(cache, "scapectl", "scapectl.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "headset diagnostic with no console") {
		t.Fatalf("closed console suppressed persistent diagnostics: %s", data)
	}
}
