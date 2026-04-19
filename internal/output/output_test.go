package output

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCreatesTimestampedDirectory(t *testing.T) {
	root := t.TempDir()
	start := time.Date(2026, 4, 19, 14, 30, 0, 0, time.UTC)

	writer, err := New(root, start)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	want := filepath.Join(root, "2026-04-19_143000")
	if writer.RunDir() != want {
		t.Fatalf("RunDir() = %q, want %q", writer.RunDir(), want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("os.Stat(%q) error = %v", want, err)
	}
}
