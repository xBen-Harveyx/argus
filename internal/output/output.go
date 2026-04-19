package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ben/argus/internal/model"
)

type Writer struct {
	rootDir string
	runDir  string
}

func New(root string, start time.Time) (*Writer, error) {
	runDir := filepath.Join(root, start.Format("2006-01-02_150405"))
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, fmt.Errorf("create run directory: %w", err)
	}
	return &Writer{rootDir: root, runDir: runDir}, nil
}

func (w *Writer) RunDir() string {
	return w.runDir
}

func (w *Writer) WriteJSON(name string, value any) error {
	path := filepath.Join(w.runDir, name)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", name, err)
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

func ManifestFor(pingLogs []string) model.FileManifest {
	return model.FileManifest{
		DeviceInfo: "device_info.json",
		Events:     "events.json",
		Summary:    "summary.json",
		PingLogs:   pingLogs,
	}
}
