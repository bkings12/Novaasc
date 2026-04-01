CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS tenants (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        VARCHAR(100) UNIQUE NOT NULL,
    name        VARCHAR(255) NOT NULL,
    plan        VARCHAR(50)  NOT NULL DEFAULT 'free',
    max_devices INT          NOT NULL DEFAULT 100,
    api_key     VARCHAR(255) UNIQUE NOT NULL,
    active      BOOLEAN      NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenants_slug    ON tenants(slug);
CREATE INDEX IF NOT EXISTS idx_tenants_api_key ON tenants(api_key);

-- Seed a default tenant for development
INSERT INTO tenants (slug, name, plan, max_devices, api_key)
VALUES ('default', 'Default Tenant', 'enterprise', 999999, 'dev-api-key-change-in-prod')
ON CONFLICT (slug) DO NOTHING;
