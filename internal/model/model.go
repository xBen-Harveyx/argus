package model

import "time"

const ToolName = "Argus"

type Target struct {
	Label      string
	Host       string
	IP         string
	Interval   time.Duration
	FileName   string
	IsGateway  bool
	IsInternet bool
}

type ProbeResult struct {
	RunID       string
	TargetLabel string
	TargetHost  string
	TargetIP    string
	Seq         int
	Timestamp   time.Time
	Result      string
	RTTMs       *int64
	Error       string
}

type OutageWindow struct {
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	MissedProbes int   `json:"missed_probes"`
}

type TargetSummary struct {
	TargetLabel          string         `json:"target_label"`
	TargetHost           string         `json:"target_host"`
	TargetIP             string         `json:"target_ip"`
	TargetKind           string         `json:"target_kind"`
	PacketsSent          int            `json:"packets_sent"`
	PacketsReceived      int            `json:"packets_received"`
	PacketsLost          int            `json:"packets_lost"`
	PacketLossPercent    float64        `json:"packet_loss_percent"`
	MinRTTMs             *int64         `json:"min_rtt_ms,omitempty"`
	AvgRTTMs             *float64       `json:"avg_rtt_ms,omitempty"`
	MaxRTTMs             *int64         `json:"max_rtt_ms,omitempty"`
	LongestFailureStreak int            `json:"longest_failure_streak"`
	OutageWindows        []OutageWindow `json:"outage_windows"`
	Warnings             []string       `json:"warnings,omitempty"`
}

type Summary struct {
	Classification string          `json:"classification"`
	Targets        []TargetSummary `json:"targets"`
	Warnings       []string        `json:"warnings,omitempty"`
}

type DeviceInfo struct {
	Hostname         string   `json:"hostname"`
	LoggedInUser     string   `json:"logged_in_user"`
	OSVersion        string   `json:"os_version"`
	OSBuild          string   `json:"os_build"`
	SystemUptime     string   `json:"system_uptime"`
	ActiveAdapter    string   `json:"active_adapter"`
	AdapterType      string   `json:"adapter_type"`
	IPAddress        string   `json:"ip_address"`
	Subnet           string   `json:"subnet"`
	DefaultGateway   string   `json:"default_gateway"`
	DNSServers       []string `json:"dns_servers"`
	MACAddress       string   `json:"mac_address"`
	WifiSSID         string   `json:"wifi_ssid,omitempty"`
	WifiSignal       string   `json:"wifi_signal,omitempty"`
	CollectionErrors []string `json:"collection_errors,omitempty"`
}

type EventRecord struct {
	Timestamp string `json:"timestamp"`
	Provider  string `json:"provider"`
	EventID   int    `json:"event_id"`
	Level     string `json:"level"`
	Channel   string `json:"channel"`
	Message   string `json:"message"`
}

type FileManifest struct {
	DeviceInfo string   `json:"device_info"`
	Events     string   `json:"events"`
	Summary    string   `json:"summary"`
	PingLogs   []string `json:"ping_logs"`
}

type RunMetadata struct {
	RunID           string       `json:"run_id"`
	Tool            string       `json:"tool"`
	Version         string       `json:"version"`
	StartTime       string       `json:"start_time"`
	EndTime         string       `json:"end_time"`
	DurationMinutes int          `json:"duration_minutes"`
	Files           FileManifest `json:"files"`
	Warnings        []string     `json:"warnings,omitempty"`
	Errors          []string     `json:"errors,omitempty"`
}
