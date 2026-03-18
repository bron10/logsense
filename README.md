# LogSense - AWS EBS/EC2 Log Explorer

A local log exploration system that ingests AWS EC2/EBS logs and provides a Datadog-like experience using Grafana + Loki.

## Architecture

```
Sample Logs (.log files)
     |                    \
     v                     v
[Promtail]           [Go Parser]
(raw ingest)      (structured JSON)
     |                     |
     v                     v
     +-------[Loki]-------+
               |
               v
           [Grafana]
        localhost:3000
```

**Dual ingestion paths:**
- **Promtail** (`job=logsense-raw`) — raw log lines with regex-extracted level labels
- **Go Parser** (`job=logsense-parsed`) — structured JSON with parsed fields, source, and level labels

## Quick Start

```bash
docker-compose up --build
```

Then open [http://localhost:3000](http://localhost:3000) — Grafana loads with no login required.

## Querying Logs

Navigate to **Explore** in Grafana and use these LogQL queries:

| Query | Description |
|-------|-------------|
| `{job="logsense-raw"}` | All raw log lines |
| `{job="logsense-parsed"}` | All structured/parsed logs |
| `{job="logsense-parsed", level="error"}` | Errors only |
| `{job="logsense-parsed", source="ec2-app"}` | EC2 app logs |
| `{job="logsense-parsed", source="ebs-volume"}` | EBS volume logs |
| `{job="logsense-parsed", source="auth-service"}` | Auth service logs |
| `{job="logsense-parsed"} \|= "timeout"` | Search for "timeout" in parsed logs |
| `{job="logsense-parsed"} \| json \| latency > 1000` | Slow requests (>1s) |

## Verification

```bash
# Check Loki is ready
curl http://localhost:3100/ready

# Check parser is running
docker logs logsense-log-parser-1

# Run Go tests
cd parser && go test ./...
```

## Project Structure

```
logsense/
├── docker-compose.yml              # Loki, Promtail, Grafana, Parser
├── config/
│   ├── loki-config.yaml            # Loki: single-tenant, filesystem, TSDB v13
│   ├── promtail-config.yaml        # Promtail: scrapes *.log, extracts level
│   └── grafana/provisioning/
│       └── datasources/
│           └── datasource.yaml     # Auto-provisions Loki datasource
├── sample-logs/
│   ├── ec2-app.log                 # ~80 lines, KV format
│   ├── ebs-volume.log              # ~50 lines, mixed JSON + plaintext
│   └── auth-service.log            # ~40 lines, KV format
└── parser/
    ├── go.mod                      # Zero external dependencies
    ├── Dockerfile                  # Multi-stage Alpine build
    ├── main.go                     # File discovery, goroutine-per-file, graceful shutdown
    └── internal/
        ├── parser/                 # Strategy pattern: JSON → Regex → raw fallback
        ├── loki/                   # Batched HTTP client for Loki push API
        └── tailer/                 # Polling-based file tailer (500ms)
```

## Design Decisions

- **Polling-based tailer** instead of fsnotify — inotify doesn't reliably propagate across Docker bind mounts
- **Zero external Go deps** — stdlib only
- **Batch pushing** — 100 entries or 1s timer, well under Loki's limits
- **Grafana anonymous admin** — zero-friction local dev experience
