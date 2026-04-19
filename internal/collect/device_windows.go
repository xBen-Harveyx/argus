//go:build windows

package collect

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"syscall"
	"time"

	"github.com/ben/argus/internal/model"
)

type powerShellDeviceCollector struct{}

func NewDeviceCollector() DeviceCollector {
	return powerShellDeviceCollector{}
}

func NewGatewayResolver() gatewayResolver {
	return gatewayResolver{}
}

type gatewayResolver struct{}

func (g gatewayResolver) DefaultGateway() (string, error) {
	script := `$cfg = Get-NetIPConfiguration | Where-Object { $_.IPv4DefaultGateway -ne $null } | Select-Object -First 1; if ($null -eq $cfg) { return }; $cfg.IPv4DefaultGateway.NextHop`
	output, err := runPowerShell(script)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (powerShellDeviceCollector) Collect(ctx context.Context) (model.DeviceInfo, []string) {
	info := model.DeviceInfo{}
	var warnings []string

	if hostname, err := Hostname(); err == nil {
		info.Hostname = hostname
	} else {
		warnings = append(warnings, fmt.Sprintf("hostname lookup failed: %v", err))
		info.CollectionErrors = append(info.CollectionErrors, err.Error())
	}

	if currentUser, err := user.Current(); err == nil {
		info.LoggedInUser = currentUser.Username
	} else {
		warnings = append(warnings, fmt.Sprintf("user lookup failed: %v", err))
		info.CollectionErrors = append(info.CollectionErrors, err.Error())
	}

	script := `
$os = Get-CimInstance Win32_OperatingSystem
$cfg = Get-NetIPConfiguration | Where-Object { $_.IPv4DefaultGateway -ne $null } | Select-Object -First 1
$adapter = $null
if ($cfg -ne $null) {
  $adapter = Get-NetAdapter -InterfaceIndex $cfg.InterfaceIndex -ErrorAction SilentlyContinue
}
$wifi = $null
try {
  $wifi = netsh wlan show interfaces | Out-String
} catch {}
[pscustomobject]@{
  os_version = $os.Caption
  os_build = $os.BuildNumber
  last_boot = $os.LastBootUpTime
  active_adapter = if ($adapter) { $adapter.Name } else { '' }
  adapter_type = if ($adapter) { $adapter.InterfaceDescription } else { '' }
  ip_address = if ($cfg -and $cfg.IPv4Address) { $cfg.IPv4Address.IPAddress } else { '' }
  prefix_length = if ($cfg -and $cfg.IPv4Address) { $cfg.IPv4Address.PrefixLength } else { 0 }
  default_gateway = if ($cfg -and $cfg.IPv4DefaultGateway) { $cfg.IPv4DefaultGateway.NextHop } else { '' }
  dns_servers = if ($cfg) { @($cfg.DNSServer.ServerAddresses) } else { @() }
  mac_address = if ($adapter) { $adapter.MacAddress } else { '' }
  wifi_dump = $wifi
} | ConvertTo-Json -Depth 4`

	output, err := runPowerShellContext(ctx, script)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("device snapshot collection failed: %v", err))
		info.CollectionErrors = append(info.CollectionErrors, err.Error())
		return info, warnings
	}

	var payload struct {
		OSVersion      string   `json:"os_version"`
		OSBuild        string   `json:"os_build"`
		LastBoot       string   `json:"last_boot"`
		ActiveAdapter  string   `json:"active_adapter"`
		AdapterType    string   `json:"adapter_type"`
		IPAddress      string   `json:"ip_address"`
		PrefixLength   int      `json:"prefix_length"`
		DefaultGateway string   `json:"default_gateway"`
		DNSServers     []string `json:"dns_servers"`
		MACAddress     string   `json:"mac_address"`
		WifiDump       string   `json:"wifi_dump"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		warnings = append(warnings, fmt.Sprintf("device snapshot parse failed: %v", err))
		info.CollectionErrors = append(info.CollectionErrors, err.Error())
		return info, warnings
	}

	info.OSVersion = payload.OSVersion
	info.OSBuild = payload.OSBuild
	info.SystemUptime = uptimeString(payload.LastBoot)
	info.ActiveAdapter = payload.ActiveAdapter
	info.AdapterType = normalizeAdapterType(payload.AdapterType)
	info.IPAddress = payload.IPAddress
	info.Subnet = prefixToMask(payload.PrefixLength)
	info.DefaultGateway = payload.DefaultGateway
	info.DNSServers = payload.DNSServers
	info.MACAddress = payload.MACAddress
	info.WifiSSID, info.WifiSignal = parseNetshWifi(payload.WifiDump)

	return info, warnings
}

func uptimeString(lastBoot string) string {
	if lastBoot == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, lastBoot)
	if err != nil {
		return ""
	}
	d := time.Since(parsed)
	if d < 0 {
		return ""
	}
	return d.Truncate(time.Second).String()
}

func normalizeAdapterType(raw string) string {
	lower := strings.ToLower(raw)
	switch {
	case strings.Contains(lower, "wi-fi"), strings.Contains(lower, "wireless"):
		return "WiFi"
	case strings.Contains(lower, "ethernet"):
		return "Ethernet"
	default:
		return raw
	}
}

func prefixToMask(prefix int) string {
	if prefix <= 0 || prefix > 32 {
		return ""
	}
	mask := uint32(0xffffffff) << (32 - prefix)
	return fmt.Sprintf("%d.%d.%d.%d", byte(mask>>24), byte(mask>>16), byte(mask>>8), byte(mask))
}

func parseNetshWifi(raw string) (string, string) {
	var ssid string
	var signal string
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "SSID"):
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 && !strings.HasPrefix(trimmed, "SSID BSSID") {
				ssid = strings.TrimSpace(parts[1])
			}
		case strings.HasPrefix(trimmed, "Signal"):
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				signal = strings.TrimSpace(parts[1])
			}
		}
	}
	return ssid, signal
}

func runPowerShell(script string) (string, error) {
	return runPowerShellContext(context.Background(), script)
}

func runPowerShellContext(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}
