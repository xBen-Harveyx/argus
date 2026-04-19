package targets

import (
	"errors"
	"testing"

	"github.com/ben/argus/internal/config"
)

func TestBuildIncludesGatewayAndInternetTargets(t *testing.T) {
	cfg := config.Config{GatewayInterval: config.DefaultGatewayInterval, InternetInterval: config.DefaultInternetInterval}
	got, warnings := Build(cfg, StaticGatewayResolver{Gateway: "192.168.1.1"})

	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if len(got) != 3 {
		t.Fatalf("len(targets) = %d, want 3", len(got))
	}
	if got[0].Label != "local_gateway" {
		t.Fatalf("first target label = %q, want local_gateway", got[0].Label)
	}
	if got[1].Label != "internet_1" || got[2].Label != "internet_2" {
		t.Fatalf("internet labels = %q, %q", got[1].Label, got[2].Label)
	}
}

func TestBuildWithoutGatewayAddsWarning(t *testing.T) {
	cfg := config.Config{GatewayInterval: config.DefaultGatewayInterval, InternetInterval: config.DefaultInternetInterval}
	got, warnings := Build(cfg, StaticGatewayResolver{Err: errors.New("not found")})

	if len(got) != 2 {
		t.Fatalf("len(targets) = %d, want 2", len(got))
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want one warning", warnings)
	}
}
