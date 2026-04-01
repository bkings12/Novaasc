CREATE TABLE IF NOT EXISTS users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(50)  NOT NULL DEFAULT 'user',
    active        BOOLEAN      NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);

CREATE INDEX IF NOT EXISTS idx_users_tenant_email
    ON users(tenant_id, email);

-- Seed: admin user for default tenant
-- Password: admin123  (bcrypt cost 12, change immediately)
INSERT INTO users (tenant_id, email, password_hash, role)
VALUES (
    (SELECT id FROM tenants WHERE slug='default'),
    'admin@novaacs.local',
    '$2a$12$/XuyDXTEf6Fk3p8Jo3.h/.NtPOLtnMpkKLLw0gG33ymGw0PmohkEi',
    'admin'
)
ON CONFLICT (tenant_id, email) DO UPDATE SET password_hash = EXCLUDED.password_hash;
