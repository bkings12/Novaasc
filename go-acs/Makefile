.PHONY: run build test test-cwmp docker compose-up migrate migrate-docker migrate-2 migrate-3 migrate-4 db-up run-docker run-with-docker-dbs lint bootstrap db-shell mongo-up mongo-shell mongo-devices

run:
	go run ./cmd/server

build:
	go build -o bin/go-acs ./cmd/server

test:
	go test ./...

test-cwmp:
	go test ./internal/cwmp/...

docker:
	docker build -t go-acs -f docker/Dockerfile .

compose-up:
	docker compose -f docker/docker-compose.yml up -d

# One-time: create acs user and novaacs DB on existing Postgres only (run as superuser; not for Docker)
bootstrap:
	psql -U postgres -h localhost -f scripts/000_bootstrap.sql

migrate:
	psql postgres://acs:acs@localhost:5432/novaacs -f migrations/001_tenants.sql

# Docker Postgres on host port 5434 (5433 is often taken by other Docker Postgres instances)
db-up:
	docker compose -f docker/docker-compose.yml up postgres -d

# Migrate when using Docker Postgres (host port 5434)
migrate-docker:
	psql postgres://acs:acs@localhost:5434/novaacs?sslmode=disable -f migrations/001_tenants.sql

# Run provisioning rules migration (Docker Postgres on 5434)
migrate-2:
	PGPASSWORD=acs psql -h localhost -p 5434 -U acs -d novaacs -f migrations/002_provisioning_rules.sql

# Run users migration (Docker Postgres on 5434)
migrate-3:
	PGPASSWORD=acs psql -h localhost -p 5434 -U acs -d novaacs -f migrations/003_users.sql

# Run credential profiles migration (Docker Postgres on 5434)
migrate-4:
	PGPASSWORD=acs psql -h localhost -p 5434 -U acs -d novaacs -f migrations/004_credential_profiles.sql

# Run app using Docker Postgres + Mongo (host ports 5434 and 27018 to avoid local DBs)
run-docker: build db-up mongo-up
	@echo "Waiting for Postgres and Mongo..."
	@sleep 4
	$(MAKE) run-with-docker-dbs

# Run app against already-running Docker Postgres (5434) and Mongo (27018). Use after: make db-up mongo-up
run-with-docker-dbs: build
	DATABASE_POSTGRES_DSN='postgres://acs:acs@localhost:5434/novaacs?sslmode=disable' \
	DATABASE_MONGO_URI='mongodb://localhost:27018' \
	DATABASE_MONGO_DB='novaacs' \
	./bin/go-acs

db-shell:
	docker compose -f docker/docker-compose.yml exec postgres psql -U acs -d novaacs

mongo-up:
	docker compose -f docker/docker-compose.yml up mongodb -d

mongo-shell:
	docker compose -f docker/docker-compose.yml exec mongodb mongosh novaacs

# Query devices (when using Docker Mongo on 27018)
mongo-devices:
	docker compose -f docker/docker-compose.yml exec mongodb mongosh novaacs --eval "db.devices.find().pretty()"

lint:
	golangci-lint run ./...
