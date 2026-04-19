## **Argus**
Argus is a Windows-based network diagnostic utility written in Go. It is designed to run silently on endpoints, typically via an RMM platform, and collect time-aligned connectivity data to help diagnose intermittent and persistent network issues.

### CLI

`argus.exe` runs silently by default and supports:

- `--duration` with a default of `30m`
- `--out` with a default of `C:\ProgramData\Argus`
- repeatable `--internet-target`
- `--interval-gateway` and `--interval-internet`
- `--include-events`
- `--run-id`
- `--silent` as a compatibility no-op flag

### Output Files

Each run writes a timestamped directory containing:

- `run.json`
- `summary.json`
- `device_info.json`
- `events.json`
- one ping CSV per target, such as `ping_local_gateway.csv`

Ping CSV rows follow:

```csv
run_id,target_label,target_host,target_ip,seq,date,time,result,rtt_ms,error
```

The JSON files use stable field names that match the v1 design doc:

- `run.json` includes run metadata, file manifest, warnings, and non-fatal errors
- `summary.json` includes per-target packet statistics, outage windows, and top-level classification
- `device_info.json` includes the startup network snapshot
- `events.json` contains normalized Windows event log records for the run window plus a five-minute buffer
