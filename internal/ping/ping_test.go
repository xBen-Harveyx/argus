package ping

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ben/argus/internal/model"
)

type staticProber struct {
	results []model.ProbeResult
	index   int
}

func (s *staticProber) Probe(_ context.Context, _ model.Target, _ time.Duration) model.ProbeResult {
	result := s.results[s.index]
	if s.index < len(s.results)-1 {
		s.index++
	}
	return result
}

func TestCSVLoggerWritesExpectedShape(t *testing.T) {
	dir := t.TempDir()
	target := model.Target{FileName: "ping_local_gateway.csv", Label: "local_gateway"}

	logger, err := NewCSVLogger(dir, target)
	if err != nil {
		t.Fatalf("NewCSVLogger() error = %v", err)
	}

	rtt := int64(42)
	err = logger.Write(model.ProbeResult{
		RunID:       "run-1",
		TargetLabel: "local_gateway",
		TargetHost:  "192.168.1.1",
		TargetIP:    "192.168.1.1",
		Seq:         1,
		Timestamp:   time.Date(2026, 4, 19, 14, 30, 0, 0, time.UTC),
		Result:      "success",
		RTTMs:       &rtt,
	})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, target.FileName))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 2 {
		t.Fatalf("line count = %d, want 2", len(lines))
	}
	if !strings.Contains(lines[1], "14:30:00") {
		t.Fatalf("row = %q, want HH:MM:SS time", lines[1])
	}
}
