package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

const diagnosticLogLimit = 5 * 1024 * 1024

// Keep diagnostics when macOS launches the app with stderr attached to /dev/null.
func startDiagnostics() (func(), error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(cache, "scapectl", "scapectl.log")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	writer := &diagnosticLog{path: path, limit: diagnosticLogLimit}
	if err := writer.open(); err != nil {
		return nil, err
	}
	previous := log.Writer()
	flags := log.Flags()
	log.SetOutput(io.MultiWriter(writer, previous))
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC)
	log.Printf("[startup] version=%s pid=%d log=%q timestamps=UTC", version, os.Getpid(), path)
	return func() {
		log.Printf("[shutdown] pid=%d", os.Getpid())
		log.SetOutput(previous)
		log.SetFlags(flags)
		writer.mu.Lock()
		defer writer.mu.Unlock()
		if writer.file != nil {
			writer.file.Close()
			writer.file = nil
		}
	}, nil
}

type diagnosticLog struct {
	mu    sync.Mutex
	path  string
	file  *os.File
	size  int64
	limit int64
}

func (w *diagnosticLog) open() error {
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	w.file, w.size = file, info.Size()
	return nil
}

func (w *diagnosticLog) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	defer func() {
		if err != nil {
			fmt.Fprintf(os.Stderr, "scapectl: diagnostic log: %v\n", err)
		}
	}()
	if w.file == nil {
		if err = w.open(); err != nil {
			return 0, err
		}
	}
	// A single oversized entry keeps its newest bytes within the same disk bound.
	originalSize := len(p)
	if int64(len(p)) > w.limit {
		p = p[len(p)-int(w.limit):]
	}
	if w.size+int64(len(p)) > w.limit {
		err = w.file.Close()
		w.file = nil
		if err != nil {
			return 0, err
		}
		if err = os.Rename(w.path, w.path+".1"); err != nil {
			return 0, err
		}
		if err = w.open(); err != nil {
			return 0, err
		}
	}
	n, err = w.file.Write(p)
	w.size += int64(n)
	if err == nil {
		n = originalSize
	}
	return n, err
}
