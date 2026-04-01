CREATE TABLE IF NOT EXISTS provisioning_rules (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name         VARCHAR(255) NOT NULL,
    description  TEXT         NOT NULL DEFAULT '',
    priority     INT          NOT NULL DEFAULT 0,
    active       BOOLEAN      NOT NULL DEFAULT true,

    -- When to fire: "0 BOOTSTRAP" | "1 BOOT" | "2 PERIODIC" | "ANY"
    trigger      VARCHAR(50)  NOT NULL DEFAULT 'ANY',

    -- Match criteria (all fields are optional, empty = match any)
    match_manufacturer  VARCHAR(255) NOT NULL DEFAULT '',
    match_oui           VARCHAR(50)  NOT NULL DEFAULT '',
    match_product_class VARCHAR(255) NOT NULL DEFAULT '',
    match_model_name    VARCHAR(255) NOT NULL DEFAULT '',
    match_sw_version    VARCHAR(255) NOT NULL DEFAULT '',

    -- Actions: JSON array of task definitions
    actions      JSONB        NOT NULL DEFAULT '[]',

    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_prov_rules_tenant_active
    ON provisioning_rules(tenant_id, active, priority DESC);

-- Seed: on every BOOTSTRAP from any MikroTik
INSERT INTO provisioning_rules
    (tenant_id, name, description, priority, trigger,
     match_manufacturer, actions)
VALUES (
    (SELECT id FROM tenants WHERE slug='default'),
    'MikroTik Bootstrap',
    'Full parameter discovery + set inform interval on first contact',
    100,
    '0 BOOTSTRAP',
    'MikroTik',
    '[
        {
            "type": "GetParameterValues",
            "parameter_names": ["Device."]
        },
        {
            "type": "SetParameterValues",
            "parameter_values": {
                "Device.ManagementServer.PeriodicInformInterval": "300",
                "Device.ManagementServer.PeriodicInformEnable": "true"
            }
        }
    ]'::jsonb
)
ON CONFLICT (tenant_id, name) DO NOTHING;

-- Seed: on every BOOT from any device
INSERT INTO provisioning_rules
    (tenant_id, name, description, priority, trigger, actions)
VALUES (
    (SELECT id FROM tenants WHERE slug='default'),
    'Any Device Boot - Collect Info',
    'Collect key parameters after every reboot',
    50,
    '1 BOOT',
    '[
        {
            "type": "GetParameterValues",
            "parameter_names": [
                "Device.DeviceInfo.SoftwareVersion",
                "Device.DeviceInfo.Uptime",
                "Device.DeviceInfo.MemoryStatus.Free",
                "Device.DeviceInfo.ProcessStatus.CPUUsage"
            ]
        }
    ]'::jsonb
)
ON CONFLICT (tenant_id, name) DO NOTHING;
