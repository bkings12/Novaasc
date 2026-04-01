-- One-time only: create acs user and novaacs DB on your *existing* Postgres (not for Docker).
-- Run as superuser: psql -U postgres -h localhost -f scripts/000_bootstrap.sql
-- Then: make migrate   and   ./bin/go-acs
-- When using Docker Postgres (make db-up / make run-docker), do NOT run this file.

CREATE ROLE acs WITH LOGIN PASSWORD 'acs';

CREATE DATABASE novaacs OWNER acs;
