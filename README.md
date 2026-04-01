# NovaACS

TR-069 / xPON ACS stack. Core server lives in **`go-acs/`** (Go). Optional UI in **`dashboard/`**.

## Documentation

- **[VPS deployment](docs/VPS-DEPLOYMENT.md)** — Docker Compose on a server, firewall, nginx, GitHub push.
- **go-acs:** [go-acs/README.md](go-acs/README.md) — local build, ports, config.
- **Architecture / implementation:** [IMPLEMENTATION_PROMPT.md](IMPLEMENTATION_PROMPT.md)

## Quick links

| Component | Path |
|-----------|------|
| ACS + API | `go-acs/` |
| Example nginx | `go-acs/nginx-sites-available-dev.conf` |
| Compose stack | `go-acs/docker/docker-compose.yml` |
