package config

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"
)

const (
	DefaultDuration         = 30 * time.Minute
	DefaultOutDir           = `C:\ProgramData\Argus`
	DefaultGatewayInterval  = time.Second
	DefaultInternetInterval = 3 * time.Second
	DefaultIncludeEvents    = true
)

type Config struct {
	Duration         time.Duration
	OutDir           string
	InternetTargets  []string
	GatewayInterval  time.Duration
	InternetInterval time.Duration
	IncludeEvents    bool
	RunID            string
	Silent           bool
}

type stringListFlag struct {
	values []string
}

func (s *stringListFlag) String() string {
	return strings.Join(s.values, ",")
}

func (s *stringListFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("internet target cannot be empty")
	}
	s.values = append(s.values, value)
	return nil
}

func Parse(args []string) (Config, error) {
	cfg := Config{
		Duration:         DefaultDuration,
		OutDir:           DefaultOutDir,
		GatewayInterval:  DefaultGatewayInterval,
		InternetInterval: DefaultInternetInterval,
		IncludeEvents:    DefaultIncludeEvents,
		Silent:           true,
	}

	fs := flag.NewFlagSet("argus", flag.ContinueOnError)
	fs.SetOutput(new(strings.Builder))

	var targets stringListFlag
	fs.Var(&targets, "internet-target", "Additional internet target")
	fs.DurationVar(&cfg.Duration, "duration", cfg.Duration, "Duration such as 5m, 15m, or 30m")
	fs.StringVar(&cfg.OutDir, "out", cfg.OutDir, "Output directory")
	fs.DurationVar(&cfg.GatewayInterval, "interval-gateway", cfg.GatewayInterval, "Gateway ping interval")
	fs.DurationVar(&cfg.InternetInterval, "interval-internet", cfg.InternetInterval, "Internet ping interval")
	fs.BoolVar(&cfg.IncludeEvents, "include-events", cfg.IncludeEvents, "Enable event collection")
	fs.StringVar(&cfg.RunID, "run-id", "", "Override generated run ID")
	fs.BoolVar(&cfg.Silent, "silent", true, "Compatibility flag; Argus runs silently by default")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	cfg.InternetTargets = targets.values

	if cfg.Duration <= 0 {
		return Config{}, errors.New("duration must be greater than zero")
	}
	if cfg.GatewayInterval <= 0 {
		return Config{}, errors.New("interval-gateway must be greater than zero")
	}
	if cfg.InternetInterval <= 0 {
		return Config{}, errors.New("interval-internet must be greater than zero")
	}
	if strings.TrimSpace(cfg.OutDir) == "" {
		return Config{}, errors.New("out must not be empty")
	}

	cfg.OutDir = strings.TrimSpace(cfg.OutDir)
	cfg.RunID = strings.TrimSpace(cfg.RunID)

	return cfg, nil
}

func (c Config) DurationMinutes() int {
	return int(c.Duration / time.Minute)
}

func (c Config) ValidatePlatform(goos string) error {
	if goos != "windows" {
		return fmt.Errorf("argus v1 is supported only on windows, got %s", goos)
	}
	return nil
}
