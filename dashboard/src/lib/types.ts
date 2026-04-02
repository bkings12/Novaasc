export interface Tenant {
  id: string;
  slug: string;
  name: string;
  plan: string;
  max_devices: number;
}

export interface Device {
  id: string;
  tenant_id: string;
  serial_number: string;
  manufacturer: string;
  oui: string;
  product_class: string;
  model_name: string;
  software_version: string;
  hardware_version: string;
  ip_address: string;
  mac_address: string;
  online: boolean;
  last_inform: string;
  last_boot: string;
  first_seen: string;
  last_events: string[];
  connection_request_url: string;
  parameters: Record<string, string>;
  tags: string[];
}

export interface Task {
  id: string;
  tenant_id: string;
  device_serial: string;
  type: string;
  status: 'pending' | 'dispatched' | 'complete' | 'failed' | 'timeout' | 'cancelled';
  priority: number;
  parameter_names?: string[];
  parameter_values?: Record<string, string>;
  result?: {
    success: boolean;
    values?: Record<string, string>;
    fault?: { code: string; message: string };
    completed_at: string;
  };
  created_at: string;
  dispatched_at?: string;
  completed_at?: string;
  created_by: string;
}

export interface ProvisioningRule {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  priority: number;
  active: boolean;
  trigger: string;
  match_manufacturer: string;
  match_oui: string;
  match_product_class: string;
  match_model_name: string;
  match_sw_version: string;
  actions: RuleAction[];
  created_at: string;
  updated_at: string;
}

export interface RuleAction {
  type: string;
  parameter_names?: string[];
  parameter_values?: Record<string, string>;
  priority: number;
}

export interface Stats {
  devices: { online: number; offline: number; total: number };
  tasks: { pending: number; failed: number };
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type: string;
}

export interface WSEvent {
  type: string;
  payload: Record<string, unknown>;
  time: string;
}

export interface BackupSummary {
  id: string;
  device_serial: string;
  label: string;
  trigger: string;
  parameter_count: number;
  software_version: string;
  ip_address: string;
  created_at: string;
  created_by: string;
}

export interface BackupDetail extends BackupSummary {
  parameters: Record<string, string>;
}

export interface BackupCreated extends BackupSummary {
  parameters: Record<string, string>;
}

export interface RestoreJobResponse {
  job_id: string;
  total_chunks: number;
  task_count: number;
  message: string;
}

export interface ApiUser {
  id: string;
  tenant_id: string;
  email: string;
  role: string;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface TenantSettings {
  id: string;
  name: string;
  slug: string;
  plan: string;
  max_devices: number;
  api_key: string;
}
