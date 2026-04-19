package collect

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/ben/argus/internal/model"
)

type DeviceCollector interface {
	Collect(ctx context.Context) (model.DeviceInfo, []string)
}

type EventCollector interface {
	Collect(ctx context.Context, start, end time.Time) ([]model.EventRecord, []string)
}

func Hostname() (string, error) {
	name, err := os.Hostname()
	if err != nil {
		return "UNKNOWN", err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "UNKNOWN", nil
	}
	return name, nil
}
