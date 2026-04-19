package run

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ben/argus/internal/config"
	"github.com/ben/argus/internal/model"
	"github.com/ben/argus/internal/targets"
)

type fakeDeviceCollector struct{}
type fakeEventCollector struct{}
type fakeProber struct{}
type fakeGatewayResolver struct{}

func (fakeDeviceCollector) Collect(context.Context) (model.DeviceInfo, []string) {
	return model.DeviceInfo{Hostname: "WS123", DefaultGateway: "192.168.1.1"}, nil
}

func (fakeEventCollector) Collect(context.Context, time.Time, time.Time) ([]model.EventRecord, []string) {
	return []model.EventRecord{{
		Timestamp: "2026-04-19T14:30:00Z",
		Provider:  "Microsoft-Windows-TCPIP",
		EventID:   1,
		Level:     "information",
		Channel:   "System",
		Message:   "ok",
	}}, nil
}

func (fakeGatewayResolver) DefaultGateway() (string, error) {
	return "192.168.1.1", nil
}

func (fakeProber) Probe(_ context.Context, target model.Target, _ time.Duration) model.ProbeResult {
	rtt := int64(5)
	return model.ProbeResult{
		Timestamp: time.Date(2026, 4, 19, 14, 30, 0, 0, time.UTC),
		Result:    "success",
		RTTMs:     &rtt,
		TargetIP:  target.IP,
	}
}

func TestExecuteWritesExpectedBundle(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
		Duration:         10 * time.Millisecond,
		OutDir:           root,
		GatewayInterval:  10 * time.Millisecond,
		InternetInterval: 10 * time.Millisecond,
		IncludeEvents:    true,
		RunID:            "20260419-143000-WS123",
		Silent:           true,
	}
	now := time.Date(2026, 4, 19, 14, 30, 0, 0, time.UTC)

	err := execute(context.Background(), cfg, runtimeDeps{
		gatewayResolver: fakeGatewayResolver{},
		deviceCollector: fakeDeviceCollector{},
		eventCollector:  fakeEventCollector{},
		prober:          fakeProber{},
		now:             func() time.Time { return now },
	})
	if err != nil && !errors.Is(err, ErrPartialFailure) {
		t.Fatalf("execute() error = %v", err)
	}

	runDir := filepath.Join(root, "2026-04-19_143000")
	for _, name := range []string{
		"run.json",
		"summary.json",
		"device_info.json",
		"events.json",
		"ping_local_gateway.csv",
		"ping_internet_1.csv",
		"ping_internet_2.csv",
	} {
		if _, err := os.Stat(filepath.Join(runDir, name)); err != nil {
			t.Fatalf("os.Stat(%q) error = %v", name, err)
		}
	}
}

func TestExecuteRejectsNonWindowsPlatform(t *testing.T) {
	cfg := config.Config{Duration: time.Second}
	err := cfg.ValidatePlatform("linux")
	if err == nil {
		t.Fatal("ValidatePlatform() error = nil, want error")
	}
}

var _ targets.GatewayResolver = fakeGatewayResolver{}
