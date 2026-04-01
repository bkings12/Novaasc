# Running NovaACS (go-acs) on a VPS

This guide assumes a fresh **Ubuntu 22.04/24.04 LTS** (or similar) VPS with SSH access. The ACS stack is the **`go-acs/`** service plus **PostgreSQL**, **MongoDB**, **Redis**, and **NATS** (as defined in `go-acs/docker/docker-compose.yml`).

---

## 1. What you are deploying

| Service   | Port (default) | Role |
|-----------|----------------|------|
| CWMP (TR-069) | **7547** | ONUs POST SOAP here (often fronted by nginx on **80**) |
| REST API    | **8080** | Dashboard / API (often behind nginx + HTTPS) |
| Connection request | **7567** | Reserved / future use |

**CPE URL (typical):** `http://<your-acs-host>/novaacs/cwmp/default` if you use the nginx layout from `go-acs/nginx-sites-available-dev.conf`, or `http://<vps-ip>:7547/cwmp/default` for a direct test.

---

## 2. Server prerequisites

```bash
sudo apt update && sudo apt install -y git ca-certificates curl
```

Install **Docker Engine** and the **Compose plugin** (follow [Docker’s official docs](https://docs.docker.com/engine/install/ubuntu/)), then:

```bash
sudo usermod -aG docker "$USER"
# log out and back in so group membership applies
```

Optional but recommended: **nginx** on the host if you want port 80/443 and TLS for the API.

---

## 3. Clone the repository

On the VPS (replace the URL with your real GitHub repo):

```bash
sudo mkdir -p /opt/novaacs
sudo chown "$USER:$USER" /opt/novaacs
cd /opt/novaacs
git clone https://github.com/bkings12/Novaasc.git .
# If the repo keeps go-acs in a subfolder (this project layout):
cd go-acs
```

If your remote is named **`Novaacs`** instead of **`Novaasc`**, use that URL everywhere below.

---

## 4. Run with Docker Compose (simplest on a VPS)

From **`go-acs/`** (where `docker/docker-compose.yml` lives):

```bash
cd /opt/novaacs/go-acs   # adjust if your clone layout differs
docker compose -f docker/docker-compose.yml build
docker compose -f docker/docker-compose.yml up -d
```

Check containers:

```bash
docker compose -f docker/docker-compose.yml ps
docker compose -f docker/docker-compose.yml logs -f go-acs --tail=100
```

The **go-acs** container reads **`config/config.yaml`** baked into the image and **environment variables** (see below). Postgres and Mongo use the internal Docker network; the compose file also publishes some **host** ports (e.g. Postgres on **5434**, Mongo on **27018**) for debugging—**lock these down** with a firewall (see section 8).

---

## 5. Production configuration (secrets and env)

**Do not** commit real secrets. On the VPS, override settings with environment variables (Viper: `database.postgres_dsn` → `DATABASE_POSTGRES_DSN`, etc.).

Edit **`docker/docker-compose.yml`** under the `go-acs` service `environment:` block, or use a **`docker-compose.override.yml`** (git-ignored locally) to set:

- **`DATABASE_POSTGRES_DSN`** — must match the **postgres** service user/password/db (change defaults in production).
- **`DATABASE_MONGO_URI`**, **`DATABASE_MONGO_DB`**
- **`AUTH_ACCESS_SECRET`**, **`AUTH_REFRESH_SECRET`** — long random strings (map to `auth.access_secret` / `auth.refresh_secret` in YAML).

Also change the default tenant **API key** in the database (`tenants.api_key`) from the dev seed value; CWMP paths like `/cwmp/<slug>` avoid needing the `X-ACS-Key` header on devices.

Rebuild/restart after changes:

```bash
docker compose -f docker/docker-compose.yml up -d --build
```

---

## 6. Firewall (UFW example)

```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
# Only if you do NOT use nginx and expose CWMP directly:
# sudo ufw allow 7547/tcp
sudo ufw enable
```

**Important:** If Postgres (**5434**), Mongo (**27018**), Redis, or NATS are published on `0.0.0.0` in compose, ensure they are **not** reachable from the internet (UFW deny, or bind to `127.0.0.1` in compose for production).

---

## 7. Nginx (recommended)

- Serve **CWMP on plain HTTP port 80** for hostnames your ONUs use (many CPEs cannot follow HTTPS redirects).
- Terminate **TLS on 443** for the **REST API** / dashboard if needed.

Adapt `go-acs/nginx-sites-available-dev.conf`: replace upstream IPs with `127.0.0.1:7547` (CWMP) and `127.0.0.1:8080` (API), set `server_name` to your domain, then:

```bash
sudo ln -sf /opt/novaacs/go-acs/nginx-sites-available-dev.conf /etc/nginx/sites-available/novaacs
sudo ln -sf /etc/nginx/sites-available/novaacs /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

---

## 8. Verifying the ACS

On the VPS:

```bash
curl -sS -o /dev/null -w '%{http_code}\n' \
  -X POST 'http://127.0.0.1:7547/cwmp/default' \
  -H 'Content-Type: text/xml; charset=utf-8' \
  --data-binary '<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:cwmp="urn:dslforum-org:cwmp-1-0"><soap:Header><cwmp:ID soap:mustUnderstand="1">1</cwmp:ID></soap:Header><soap:Body><cwmp:Inform><DeviceId><Manufacturer>T</Manufacturer><OUI>000000</OUI><ProductClass>X</ProductClass><SerialNumber>TEST-1</SerialNumber></DeviceId><Event><EventStruct><EventCode>1 BOOT</EventCode></EventStruct></Event><MaxEnvelopes>1</MaxEnvelopes><CurrentTime>2026-01-01T00:00:00Z</CurrentTime><RetryCount>0</RetryCount><ParameterList></ParameterList></cwmp:Inform></soap:Body></soap:Envelope>'
```

Expect **200**. Watch logs for `inform received` / `device upserted`.

---

## 9. Pushing this project to GitHub from your laptop

If the repo is still only on your machine:

```bash
cd /path/to/Novaacs   # your local project root

git init
git branch -M main

# Optional: root README (or keep the one in the repo)
git add .
git commit -m "Initial commit"

git remote add origin https://github.com/bkings12/Novaasc.git
git push -u origin main
```

Create an **empty** repository on GitHub first (same name as in the URL). Fix the typo **`Novaasc`** vs **`Novaacs`** in the repo name and remote URL so they match everywhere.

**Before the first push:** ensure **`.gitignore`** excludes `node_modules/`, `go-acs/bin/`, local env files, and secrets.

---

## 10. Operations cheatsheet

| Action | Command |
|--------|---------|
| Logs | `docker compose -f docker/docker-compose.yml logs -f go-acs` |
| Restart ACS only | `docker compose -f docker/docker-compose.yml restart go-acs` |
| Stop all | `docker compose -f docker/docker-compose.yml down` |
| Update code | `git pull && docker compose -f docker/docker-compose.yml up -d --build` |

---

## 11. Dashboard (optional)

The **`dashboard/`** Svelte app is separate. For production you typically build static assets or run a Node adapter behind nginx; see that package’s README when you are ready to host it on the same VPS.
