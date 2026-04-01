-- Add default connection request credentials to tenants table
ALTER TABLE tenants
  ADD COLUMN IF NOT EXISTS default_cr_username VARCHAR(255) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS default_cr_password VARCHAR(255) NOT NULL DEFAULT '';

-- Per-OUI credential profiles
CREATE TABLE IF NOT EXISTS credential_profiles (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name          VARCHAR(255) NOT NULL,
    oui           VARCHAR(50)  NOT NULL DEFAULT '',
    manufacturer  VARCHAR(255) NOT NULL DEFAULT '',
    model_name    VARCHAR(255) NOT NULL DEFAULT '',
    cr_username   VARCHAR(255) NOT NULL DEFAULT '',
    cr_password   VARCHAR(255) NOT NULL DEFAULT '',
    cwmp_username VARCHAR(255) NOT NULL DEFAULT '',
    cwmp_password VARCHAR(255) NOT NULL DEFAULT '',
    active        BOOLEAN      NOT NULL DEFAULT true,
    notes         TEXT         NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_cred_profiles_tenant_name
    ON credential_profiles(tenant_id, name);
CREATE INDEX IF NOT EXISTS idx_cred_profiles_tenant_oui
    ON credential_profiles(tenant_id, oui) WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_cred_profiles_tenant_manufacturer
    ON credential_profiles(tenant_id, manufacturer) WHERE active = true;

-- Seed: common xPON vendor defaults
INSERT INTO credential_profiles
    (tenant_id, name, manufacturer, oui, cr_username, cr_password, notes)
VALUES
    ((SELECT id FROM tenants WHERE slug = 'default' LIMIT 1), 'Huawei ONU Default', 'Huawei', '00E0FC', 'admin', 'admin', 'Huawei HG8xxx factory default'),
    ((SELECT id FROM tenants WHERE slug = 'default' LIMIT 1), 'ZTE ONU Default', 'ZTE', '001E73', 'admin', '1234', 'ZTE F6xx factory default'),
    ((SELECT id FROM tenants WHERE slug = 'default' LIMIT 1), 'Nokia/Alcatel ONU Default', 'Nokia', 'D4CA6D', 'admin', 'admin', 'Nokia G-series factory default'),
    ((SELECT id FROM tenants WHERE slug = 'default' LIMIT 1), 'FiberHome ONU Default', 'FiberHome', 'ACBF09', 'admin', 'admin', 'FiberHome AN5506 factory default'),
    ((SELECT id FROM tenants WHERE slug = 'default' LIMIT 1), 'Calix ONU Default', 'Calix', '0026CE', 'admin', 'admin', 'Calix 700-series factory default')
ON CONFLICT (tenant_id, name) DO NOTHING;
