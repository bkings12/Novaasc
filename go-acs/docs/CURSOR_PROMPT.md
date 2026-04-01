# Cursor prompt: implement go-acs TR-069 ACS (TR-181, xPON/GPON)

Use this when asking Cursor to implement or extend **go-acs**. Implement in the order listed at the end.

---

**Project:** go-acs — production TR-069 ACS in Go (GenieACS replacement), 50k+ concurrent CPE sessions. Full support for **TR-069**, **TR-181** (Device:2), **TR-098** (InternetGatewayDevice), and **xPON/GPON** routers and ONTs.

**Standards:** TR-069 (CWMP 1.0/1.1/1.2), TR-181 (`Device.*`), TR-098 (`InternetGatewayDevice.*`). xPON/GPON: support `Device.X_*`, `Device.PON.*`, OUI/product class and provisioning for OLT/ONT.

**Stack:** Go 1.22+, Fiber, PostgreSQL (pgx/v5), MongoDB, Redis, NATS, Viper, Zap, JWT, gorilla/websocket, testify.

**Layout:** `cmd/server`, `internal/acs`, `internal/cwmp`, `internal/device`, `internal/provisioning`, `internal/task`, `internal/api`, `internal/firmware`, `internal/notification`, `internal/config`, `pkg/xmlutil`, `pkg/netutil`, `config`, `migrations/postgres`, `docker`.

**Session (internal/cwmp/session.go):** States: New → Informed → WaitingTask → Idle → Done/Failed. Goroutine-safe, store full parameter tree from Inform (TR-181/TR-098), one task at a time, handle empty POST (“what’s next?”), configurable idle timeout, audit to MongoDB.

**SOAP (internal/cwmp/soap.go):** stdlib `encoding/xml` only. Parse/build: Inform, InformResponse, GetParameterValues/Response, SetParameterValues/Response, GetParameterNames/Response, Download/Response, Reboot/Response, AddObject/DeleteObject/Response, TransferComplete, AutonomousTransferComplete, Fault. Envelope: `soap` and `cwmp` namespaces; always echo `cwmp:ID` in responses.

**Device (internal/device/model.go):** Device ID = OUI-ProductClass-SerialNumber. Parameters flat map, dot-notation (Device.*, InternetGatewayDevice.*, Device.X_* for xPON). Parameter: Value, Type, Writable, UpdatedAt.

**Tasks:** One active task per device; correlate by CommandKey; on session timeout mark task failed. Types: getParameterValues, setParameterValues, getParameterNames, download, reboot, factoryReset, addObject, deleteObject, refresh.

**Provisioning:** After each Inform, evaluate active profiles (conditions on device + params), queue actions as tasks. Support xPON/GPON presets (OUI, ProductClass, tags). Skip if profile already applied (hash).

**REST API:** Devices (CRUD, parameters, tasks, actions), firmware, profiles, global tasks, stats. WebSocket `/api/v1/ws`: device_inform, device_online, device_offline, task_completed, task_failed. Auth: JWT login/refresh.

**Config:** config.yaml — acs (ports 7547/7567/8080, session_timeout, max_sessions, tls), database (mongodb, postgres, redis), nats, auth (jwt_secret, token_expiry), logging.

**Build order:** (1) go.mod + structure (2) config (3) logger (4) SOAP + CWMP messages (5) session state machine (6) ACS HTTP 7547 (7) Inform (8) MongoDB device repo (9) task queue (10) Get/SetParameterValues (11) Download/Reboot (12) provisioning (13) REST API (14) WebSocket (15) connection request 7567 (16) auth (17) Docker (18) tests. Start with 1–6; everything else plugs into session + SOAP.

**Notes:** HTTP 200 even on SOAP Fault. Session cookie. Rate limit per IP. Fault codes 8000/8005/9001. Firmware: Download RPC with URL to internal server; CPE pulls via HTTP; use TransferComplete.
