package targets

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ben/argus/internal/config"
	"github.com/ben/argus/internal/model"
)

const (
	defaultInternetOne = "1.1.1.1"
	defaultInternetTwo = "8.8.8.8"
)

type GatewayResolver interface {
	DefaultGateway() (string, error)
}

func Build(cfg config.Config, resolver GatewayResolver) ([]model.Target, []string) {
	var result []model.Target
	var warnings []string

	if gateway, err := resolver.DefaultGateway(); err != nil {
		warnings = append(warnings, fmt.Sprintf("default gateway discovery failed: %v", err))
	} else if strings.TrimSpace(gateway) == "" {
		warnings = append(warnings, "default gateway discovery returned no gateway")
	} else {
		result = append(result, model.Target{
			Label:      "local_gateway",
			Host:       gateway,
			IP:         gateway,
			Interval:   cfg.GatewayInterval,
			FileName:   "ping_local_gateway.csv",
			IsGateway:  true,
			IsInternet: false,
		})
	}

	internetHosts := []string{defaultInternetOne, defaultInternetTwo}
	internetHosts = append(internetHosts, cfg.InternetTargets...)

	for idx, host := range internetHosts {
		ip, warning := resolveTarget(host)
		if warning != "" {
			warnings = append(warnings, warning)
		}
		label := fmt.Sprintf("internet_%d", idx+1)
		result = append(result, model.Target{
			Label:      label,
			Host:       host,
			IP:         ip,
			Interval:   cfg.InternetInterval,
			FileName:   fmt.Sprintf("ping_%s.csv", label),
			IsGateway:  false,
			IsInternet: true,
		})
	}

	return result, warnings
}

func resolveTarget(host string) (string, string) {
	host = strings.TrimSpace(host)
	if ip := net.ParseIP(host); ip != nil {
		if v4 := ip.To4(); v4 != nil {
			return v4.String(), ""
		}
		return ip.String(), ""
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return "", fmt.Sprintf("failed to resolve target %q: %v", host, err)
	}
	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			return v4.String(), ""
		}
	}
	if len(ips) > 0 {
		return ips[0].String(), ""
	}
	return "", fmt.Sprintf("failed to resolve target %q: no ip addresses returned", host)
}

type StaticGatewayResolver struct {
	Gateway string
	Err     error
}

func (s StaticGatewayResolver) DefaultGateway() (string, error) {
	return s.Gateway, s.Err
}

func RoundDurationToInterval(duration, interval time.Duration) int {
	if interval <= 0 {
		return 0
	}
	return int(duration / interval)
}
