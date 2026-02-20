# go-disaster-alerts

A Go backend service that aggregates real-time disaster data from GDACS, providing both a REST API and gRPC streaming for live alerts.

## Architecture

```
             ┌─────────────┐
             │    GDACS    │
             │  (RSS/XML)  │
             └──────┬──────┘
                    │ poll (5m interval)
                    ▼
         ┌─────────────────────┐
         │  Ingestion Manager  │
         │  (retry + backoff)  │
         └──────────┬──────────┘
                    │
         ┌──────────┴──────────┐
         │                     │
         ▼                     ▼
    ┌───────────┐       ┌─────────────┐
    │  SQLite   │       │ Broadcaster │
    │   (DB)    │       │   (gRPC)    │
    └─────┬─────┘       └──────┬──────┘
          │                    │
          ▼                    ▼
    ┌───────────┐       ┌─────────────┐
    │ REST API  │       │ gRPC Stream │
    │ (GeoJSON) │       │ (real-time) │
    └─────┬─────┘       └──────┬──────┘
          │                    │
          ▼                    ▼
    ┌───────────┐       ┌─────────────┐
    │  Web App  │       │ Discord Bot │
    │   (map)   │       │  (alerts)   │
    └───────────┘       └─────────────┘
```

## Features

- Polls GDACS for earthquakes, floods, cyclones, tsunamis, volcanoes, wildfires, and droughts
- REST API returning GeoJSON for map integration
- gRPC streaming for real-time disaster notifications
- SQLite storage with deduplication
- Retry with exponential backoff for API resilience
- Rate limiting and CORS middleware

## Tech Stack

- Go 1.25.6
- Gin (HTTP router)
- gRPC + Protocol Buffers
- SQLite (via modernc.org/sqlite)

## Setup

```bash
# Clone
git clone https://github.com/mr1hm/go-disaster-alerts.git
cd go-disaster-alerts

# Install dependencies
go mod download

# Run
go run ./cmd/disaster-alert
```

## Configuration

Create a `.env` file (for local development):

```env
# Server
SERVER_HOST=localhost
SERVER_PORT=8080
GRPC_PORT=50051

# Database
DB_PATH=./data/disasters.db

# Sources
GDACS_ENABLED=true
GDACS_URL=https://www.gdacs.org/xml/rss.xml
GDACS_POLL_INTERVAL=5m

# Logging
LOG_LEVEL=info
```

## Docker Deployment

```bash
# Build and run
docker-compose up -d --build

# View logs
docker logs -f disaster-alerts

# Stop
docker-compose down
```

The `docker-compose.yml` includes all necessary environment variables - no `.env` file needed for Docker.

## REST API

### GET /api/disasters

Returns disasters as GeoJSON.

Query params:
- `type` - earthquake, flood, cyclone, tsunami, volcano, wildfire, drought
- `min_magnitude` - minimum magnitude (e.g., 5.0)
- `alert_level` - exact match: green, orange, red
- `min_alert_level` - minimum level (e.g., `orange` returns orange AND red)
- `since` - date filter (YYYY-MM-DD)
- `limit` - max results (default 20, max 500)

```bash
# Get all earthquakes with magnitude >= 5.0
curl "http://localhost:8080/api/disasters?type=earthquake&min_magnitude=5.0"

# Get all orange and red alerts
curl "http://localhost:8080/api/disasters?min_alert_level=orange"
```

### GET /health

Health check endpoint.

### POST /api/debug/test-disaster

Broadcasts a test disaster to gRPC subscribers (not persisted to DB).

```bash
curl -X POST http://localhost:8080/api/debug/test-disaster
```

## gRPC Service

Proto file: `proto/disasters/v1/disasters.proto`

### RPCs

- `GetDisaster(id)` - Get single disaster by ID
- `ListDisasters(limit, type, min_magnitude, alert_level, min_alert_level, discord_sent, since, min_affected_population_count)` - Query disasters
- `StreamDisasters(type, min_magnitude, alert_level, min_alert_level)` - Server-side stream of new disasters
- `AcknowledgeDisasters(ids)` - Mark disasters as successfully posted to Discord (prevents duplicates on bot restart)

### Streaming Example

```bash
grpcurl -plaintext -proto proto/disasters/v1/disasters.proto \
  localhost:50051 disasters.v1.DisasterService/StreamDisasters
```

## Disaster Fields

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique ID (e.g., `gdacs_12345`) |
| source | string | `GDACS` |
| type | DisasterType | Category of disaster |
| title | string | Event title/summary |
| magnitude | double | Richter scale (earthquakes) |
| alert_level | AlertLevel | Severity level |
| latitude | double | Event latitude |
| longitude | double | Event longitude |
| timestamp | int64 | Unix timestamp of event |
| country | string | Country where disaster occurred |
| affected_population | string | Text description (e.g., "1 thousand (in MMI>=VII)") |
| report_url | string | Link to detailed GDACS report |
| affected_population_count | int64 | Numeric population value for filtering |

## Enums

### DisasterType
- UNSPECIFIED (0)
- EARTHQUAKE (1)
- FLOOD (2)
- CYCLONE (3)
- TSUNAMI (4)
- VOLCANO (5)
- WILDFIRE (6)
- DROUGHT (7)

### AlertLevel
- UNKNOWN (0) - Default/unset
- GREEN (1) - Minor impact
- ORANGE (2) - Moderate impact
- RED (3) - Severe, may need international aid

## Development

```bash
# Run tests
go test -race ./...

# Regenerate proto
protoc --go_out=. --go_opt=module=github.com/mr1hm/go-disaster-alerts \
       --go-grpc_out=. --go-grpc_opt=module=github.com/mr1hm/go-disaster-alerts \
       proto/disasters/v1/disasters.proto
```

## License

MIT
