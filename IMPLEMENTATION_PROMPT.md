# Cursor Implementation Prompt: go-acs — Full TR-069 ACS (GenieACS Replacement)

Use this prompt when implementing or extending the **go-acs** project. Implement in the order specified at the end.

---

## PROJECT OVERVIEW

**go-acs** is a production-grade TR-069 Auto Configuration Server (ACS) written in Go, designed as a complete GenieACS replacement. It must support **50,000+ concurrent CPE sessions** and fully support **TR-069**, **TR-181** (Device:2 data model), and **xPON/GPON** routers and ONTs.

### Standards & Device Support

- **TR-069** (CPE WAN Management Protocol): CWMP 1.0/1.1/1.2 — RPCs, session handling, connection request.
- **TR-181** (Device:2): Primary data model; parameters under `Device.` (e.g. `Device.DeviceInfo`, `Device.WANDevice`, `Device.X_*` vendor extensions).
- **TR-098** (InternetGatewayDevice:1): Legacy IGD model; parameters under `InternetGatewayDevice.` — support both namespaces in SOAP and parameter storage.
- **xPON / GPON**: Support OLT/ONT parameters and vendor extensions typical of GPON/xPON CPE (e.g. `Device.X_*`, `Device.PON.*`, OUI/product class filtering, provisioning profiles for xPON devices).

---

## TECH STACK

| Component   | Choice |
|------------|--------|
| Language   | Go 1.22+ |
| HTTP       | github.com/gofiber/fiber/v2 |
| Database   | PostgreSQL (pgx/v5), MongoDB (mongo-driver), Redis (go-redis/v9) |
| Messaging  | NATS (nats.go) |
| Config     | github.com/spf13/viper |
| Logging    | go.uber.org/zap |
| Auth       | golang-jwt/jwt/v5 |
| Testing    | testify |
| WebSocket  | gorilla/websocket |

---

## PROJECT STRUCTURE

```
go-acs/
├── cmd/server/main.go
├── internal/
│   ├── acs/           # ACS HTTP server, CWMP handler (7547), connection request (7567), middleware
│   ├── cwmp/          # Session, SOAP, Inform, Get/SetParameterValues, Download, Reboot, etc.
│   ├── device/        # Device model (TR-181/TR-098), repository, parameter store, event store
│   ├── provisioning/  # Rule engine, scripts, presets, evaluator
│   ├── task/          # Task model, queue (NATS/Redis), scheduler, executor
│   ├── api/           # REST router, devices, tasks, firmware, provisioning, websocket, auth
│   ├── firmware/      # Store, repository, HTTP server for CPE downloads
│   ├── notification/  # Publisher, subscriber (WebSocket push)
│   └── config/        # Config struct, loader
├── pkg/xmlutil, pkg/netutil
├── migrations/postgres/
├── scripts/provisioning/
├── docker/
├── config/config.yaml
├── go.mod, Makefile
```

---

## CORE: CWMP SESSION STATE MACHINE (`internal/cwmp/session.go`)

States:

```go
const (
    StateNew           SessionState = iota  // Just connected, awaiting Inform
    StateInformed                           // Inform received, dispatching tasks
    StateWaitingTask                        // Sent a task, awaiting response
    StateIdle                               // No pending tasks, session can end
    StateDone                               // Session complete
    StateFailed                             // Session failed
)
```

Requirements:

1. **Goroutine-safe**: mutex or channels; no data races.
2. **Parameter tree**: Persist full device parameter tree from Inform (TR-181/TR-098 paths).
3. **Task queue**: Queue multiple tasks per device; dispatch **one at a time**; wait for CPE response before next.
4. **Hold requests / empty POST**: Handle CPE “hold” (empty body = “what’s next?”) — respond with next RPC or 204.
5. **Idle timeout**: Configurable (default 30s); close session on timeout.
6. **Audit**: Store session events (e.g. to MongoDB) for audit.

Session flow:

1. CPE: `POST /acs` with Inform body → ACS: `InformResponse`.
2. CPE: `POST /acs` (empty) → ACS: next RPC (GetParameterValues, SetParameterValues, Download, Reboot, etc.).
3. CPE: `POST /acs` (response) → ACS: next RPC or empty (session ends).

---

## CORE: SOAP PARSER (`internal/cwmp/soap.go`)

- Use **encoding/xml** only; no third-party SOAP libraries.
- Parse and build all CWMP messages listed below.
- Envelope: `xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"`, `xmlns:cwmp="urn:dslforum-org:cwmp-1-0"`.
- **Always** set `cwmp:ID` in responses to match the request ID.
- Support **TR-098** (`InternetGatewayDevice.`) and **TR-181** (`Device.`) in parameter paths and storage.

**Incoming (from CPE):** Inform, GetParameterValuesResponse, SetParameterValuesResponse, GetParameterNamesResponse, DownloadResponse, RebootResponse, AddObjectResponse, DeleteObjectResponse, TransferComplete, AutonomousTransferComplete, Fault.

**Outgoing (from ACS):** InformResponse, GetParameterValues, SetParameterValues, GetParameterNames, Download, Upload, Reboot, FactoryReset, AddObject, DeleteObject, ScheduleInform.

Handle `text/xml` and `application/soap+xml`. Session tracking via cookie; optionally verify HTTP Basic Auth from CPE.

---

## CORE: DEVICE MODEL (`internal/device/model.go`)

- `Device`: ID, SerialNumber, OUI, ProductClass, Manufacturer, SoftwareVersion, HardwareVersion, IPAddress, ConnectionURL, LastInform, FirstSeen, Status, Tags, Parameters (map[string]Parameter), Events, CreatedAt, UpdatedAt.
- `Parameter`: Value (interface{}), Type (xsd:string, xsd:boolean, xsd:int, etc.), Writable, UpdatedAt.
- `DeviceEvent`: EventCode, CommandKey, Timestamp.
- **Device ID**: Unique = `OUI + "-" + ProductClass + "-" + SerialNumber`.
- Parameters stored as **flat map** with **dot-notation** keys (e.g. `Device.DeviceInfo.SerialNumber`, `InternetGatewayDevice.DeviceInfo.SerialNumber`). Index by device_id + path for fast lookups.
- For **xPON/GPON**: Store and index vendor paths (e.g. `Device.X_*`, `Device.PON.*`) the same way; use Tags or ProductClass for OLT/ONT filtering in provisioning.

---

## CORE: TASK MODEL & QUEUE (`internal/task/`)

- **Task**: ID, DeviceID, Type, Status, Params, Result, Error, CommandKey, Retries, MaxRetries, CreatedBy, CreatedAt, UpdatedAt, ExecutedAt.
- **TaskType**: getParameterValues, setParameterValues, getParameterNames, download, reboot, factoryReset, addObject, deleteObject, refresh.
- **TaskStatus**: pending, running, completed, failed, cancelled.
- **One active task per device**; correlate via CommandKey (e.g. UUID). On session_timeout with no response, mark task failed.

---

## PROVISIONING ENGINE (`internal/provisioning/`)

- **ProvisioningProfile**: ID, Name, Priority, Conditions (AND), Actions, Active.
- **ProfileCondition**: Field (e.g. manufacturer, productClass, tag, parameter path), Operator (eq, neq, contains, regex, exists), Value.
- **ProfileAction**: Type (setParameterValues, download, reboot, etc.), Params (JSON).
- After **every Inform**: load active profiles by priority, evaluate conditions (device attributes + parameters, including TR-181/xPON paths), queue actions as tasks. Skip if profile already applied (e.g. hash of profile + device state). Support **xPON/GPON** presets (e.g. by OUI, ProductClass, or tag "gpon"/"xpon").

---

## REST API (`internal/api/`)

- **Devices**: GET/POST/DELETE, GET parameters (all or by path), GET/POST/DELETE tasks, POST actions (reboot, refresh, factory-reset, set-parameters, download).
- **Firmware**: GET/POST/DELETE.
- **Profiles**: GET/POST/PUT/DELETE.
- **Tasks**: GET global list; GET stats.
- **WebSocket** `/api/v1/ws`: events device_inform, device_online, device_offline, task_completed, task_failed (with device id and task id).
- **Auth**: POST login/refresh (JWT).

---

## CONFIG (`config/config.yaml`)

- **acs**: cwmp_port (7547), connection_request_port (7567), api_port (8080), base_url, firmware_base_url, session_timeout (30s), max_concurrent_sessions (10000), tls.
- **database**: mongodb uri/database, postgres dsn, redis addr/password/db.
- **nats**: url.
- **auth**: jwt_secret, token_expiry.
- **logging**: level, format (json/console).

---

## IMPLEMENTATION NOTES

1. **SOAP**: HTTP 200 even for SOAP Fault (TR-069). Echo cwmp:ID. Cookie for session. Support Basic Auth check.
2. **Concurrency**: sync.Map (or equivalent) for sessions; one goroutine per session; context with timeout; rate limit per device IP.
3. **Parameter tree**: TR-181 `Device.*` and TR-098 `InternetGatewayDevice.*`; flat dot-notation in DB; support `Device.X_*` for xPON.
4. **Faults**: 8000 method not supported, 8005 invalid parameter name, 9001 request denied; log device + session.
5. **Firmware**: ACS sends Download RPC with URL to internal firmware server; CPE downloads via HTTP; use TransferComplete for status.
6. **WebSocket**: gorilla/websocket, ping/pong ~30s; clients filter by device ID or “all”.

---

## DOCKER & MAKE

- **docker-compose**: go-acs, mongodb:7, postgres:16, redis:7-alpine, nats:2-alpine.
- **Makefile**: run, build, test, test-cwmp, docker, compose-up, migrate, lint.

---

## BUILD ORDER (implement in this order)

1. go.mod + project structure  
2. Config loader (viper)  
3. Logger (zap)  
4. SOAP parser/builder (encoding/xml, all CWMP message types)  
5. CWMP session state machine  
6. ACS HTTP handler (Fiber, port 7547)  
7. Inform handler  
8. MongoDB device repository  
9. Task queue (Redis)  
10. GetParameterValues / SetParameterValues handlers  
11. Download / Reboot handlers  
12. Provisioning engine  
13. REST API (Fiber, port 8080)  
14. WebSocket live updates  
15. Connection request server (port 7567)  
16. Auth (JWT)  
17. Docker / docker-compose  
18. Tests: CWMP session, SOAP parsing, provisioning engine  

**Start with steps 1–6:** project layout, config, logger, SOAP structs, session state machine, ACS HTTP server. All other features plug into the session and SOAP layer.
