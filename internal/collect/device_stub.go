//go:build !windows

package collect

import (
	"context"
	"time"

	"github.com/ben/argus/internal/model"
)

type unsupportedDeviceCollector struct{}
type unsupportedGatewayResolver struct{}
type unsupportedEventCollector struct{}
type unsupportedProber struct{}

func NewDeviceCollector() DeviceCollector { return unsupportedDeviceCollector{} }
func NewGatewayResolver() unsupportedGatewayResolver { return unsupportedGatewayResolver{} }
func NewEventCollector() EventCollector   { return unsupportedEventCollector{} }
func NewProber() unsupportedProber        { return unsupportedProber{} }

func (unsupportedDeviceCollector) Collect(context.Context) (model.DeviceInfo, []string) {
	return model.DeviceInfo{}, []string{"device collection is supported only on windows"}
}

func (unsupportedGatewayResolver) DefaultGateway() (string, error) {
	return "", nil
}

func (unsupportedEventCollector) Collect(context.Context, time.Time, time.Time) ([]model.EventRecord, []string) {
	return nil, []string{"event collection is supported only on windows"}
}

func (unsupportedProber) Probe(context.Context, model.Target, time.Duration) model.ProbeResult {
	return model.ProbeResult{Result: "error", Error: "icmp probing is supported only on windows"}
}
