package config

import "testing"

func TestParseDefaults(t *testing.T) {
	cfg, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if cfg.Duration != DefaultDuration {
		t.Fatalf("Duration = %v, want %v", cfg.Duration, DefaultDuration)
	}
	if cfg.OutDir != DefaultOutDir {
		t.Fatalf("OutDir = %q, want %q", cfg.OutDir, DefaultOutDir)
	}
	if !cfg.IncludeEvents {
		t.Fatal("IncludeEvents = false, want true")
	}
	if !cfg.Silent {
		t.Fatal("Silent = false, want true")
	}
}

func TestParseRepeatableTargets(t *testing.T) {
	cfg, err := Parse([]string{"--internet-target", "9.9.9.9", "--internet-target", "example.com"})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(cfg.InternetTargets) != 2 {
		t.Fatalf("len(InternetTargets) = %d, want 2", len(cfg.InternetTargets))
	}
}

func TestParseRejectsInvalidDuration(t *testing.T) {
	if _, err := Parse([]string{"--duration", "0s"}); err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
}
