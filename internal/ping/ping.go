package ping

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/ben/argus/internal/model"
)

type Prober interface {
	Probe(ctx context.Context, target model.Target, timeout time.Duration) model.ProbeResult
}

type CSVLogger struct {
	file   *os.File
	writer *csv.Writer
	mu     sync.Mutex
}

func NewCSVLogger(dir string, target model.Target) (*CSVLogger, error) {
	path := filepath.Join(dir, target.FileName)
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create ping log %s: %w", target.FileName, err)
	}

	writer := csv.NewWriter(file)
	header := []string{"run_id", "target_label", "target_host", "target_ip", "seq", "date", "time", "result", "rtt_ms", "error"}
	if err := writer.Write(header); err != nil {
		file.Close()
		return nil, fmt.Errorf("write ping log header %s: %w", target.FileName, err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		file.Close()
		return nil, fmt.Errorf("flush ping log header %s: %w", target.FileName, err)
	}

	return &CSVLogger{file: file, writer: writer}, nil
}

func (l *CSVLogger) Write(result model.ProbeResult) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	record := []string{
		result.RunID,
		result.TargetLabel,
		result.TargetHost,
		result.TargetIP,
		strconv.Itoa(result.Seq),
		result.Timestamp.Format("2006-01-02"),
		result.Timestamp.Format("15:04:05"),
		result.Result,
		"",
		result.Error,
	}
	if result.RTTMs != nil {
		record[8] = strconv.FormatInt(*result.RTTMs, 10)
	}
	if err := l.writer.Write(record); err != nil {
		return err
	}
	l.writer.Flush()
	return l.writer.Error()
}

func (l *CSVLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.writer.Flush()
	if err := l.writer.Error(); err != nil {
		_ = l.file.Close()
		return err
	}
	return l.file.Close()
}

type Worker struct {
	Target  model.Target
	RunID   string
	Timeout time.Duration
	Prober  Prober
	Logger  *CSVLogger
	Now     func() time.Time
}

func (w Worker) Run(ctx context.Context, results chan<- model.ProbeResult, errors chan<- error) {
	ticker := time.NewTicker(w.Target.Interval)
	defer ticker.Stop()
	defer func() {
		if err := w.Logger.Close(); err != nil {
			errors <- fmt.Errorf("close ping log for %s: %w", w.Target.Label, err)
		}
	}()

	seq := 0
	for {
		seq++
		result := w.Prober.Probe(ctx, w.Target, w.Timeout)
		result.RunID = w.RunID
		result.TargetLabel = w.Target.Label
		result.TargetHost = w.Target.Host
		result.TargetIP = w.Target.IP
		result.Seq = seq
		if result.Timestamp.IsZero() {
			result.Timestamp = w.now()
		}
		if err := w.Logger.Write(result); err != nil {
			errors <- fmt.Errorf("write ping result for %s: %w", w.Target.Label, err)
		}
		results <- result

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w Worker) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}
