# go-acs

Production TR-069 ACS (Auto Configuration Server) in Go — a GenieACS replacement. Targets 50,000+ concurrent CPE sessions with full TR-069, TR-181 (Device:2), and xPON/GPON router support.

## Quick start

```bash
# From repo root
go mod tidy
go build -o bin/go-acs ./cmd/server
./bin/go-acs                    # uses ./config/config.yaml
./bin/go-acs /path/to/config.yaml
```

Or:

```bash
make run
```

## Implementation prompt

**Use the implementation prompt when extending or building features.** It defines the full architecture, build order, and TR-069/TR-181/xPON details:

- **Project root:** [IMPLEMENTATION_PROMPT.md](../IMPLEMENTATION_PROMPT.md) (one level up from `go-acs/`)
- **Or in this repo:** see the same content referenced from the project root.

Build order: config → logger → SOAP/CWMP → session state machine → ACS HTTP (7547) → Inform → device repo → task queue → RPC handlers → provisioning → REST API → WebSocket → connection request → auth → Docker → tests.

## Tech stack

- **Go 1.22+**, Fiber, pgx/v5, MongoDB driver, Redis, NATS, Viper, Zap, JWT, testify, gorilla/websocket

## Ports

- **7547** — CWMP (TR-069)
- **7567** — Connection request (to be implemented)
- **8080** — REST API (to be implemented)

## Config

Edit `config/config.yaml`. Key options: `acs.cwmp_port`, `acs.session_timeout`, `database.*`, `auth.jwt_secret`.

## Docker

```bash
docker compose -f docker/docker-compose.yml up -d
```

Services: go-acs, mongodb:7, postgres:16, redis:7-alpine, nats:2-alpine.
