import { getToken, logout, tryRefresh } from './auth';
import type {
  Device,
  Task,
  ProvisioningRule,
  Stats,
  BackupSummary,
  BackupDetail,
  BackupCreated,
  RestoreJobResponse,
  ApiUser,
  TenantSettings
} from './types';

const BASE =
  import.meta.env.VITE_API_URL ??
  (typeof window !== 'undefined' ? '' : 'http://localhost:8080');

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  isRetry = false
): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${getToken()}`
    },
    body: body ? JSON.stringify(body) : undefined
  });

  if (res.status === 401 && !isRetry) {
    const ok = await tryRefresh();
    if (ok) return request<T>(method, path, body, true);
    logout();
    throw new Error('Unauthorized');
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error((err as { error?: string }).error ?? res.statusText);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

const get = <T>(path: string) => request<T>('GET', path);
const post = <T>(path: string, body?: unknown) => request<T>('POST', path, body);
const put = <T>(path: string, body?: unknown) => request<T>('PUT', path, body);
const del = <T>(path: string) => request<T>('DELETE', path);

export const api = {
  stats: () => get<Stats>('/api/v1/stats'),

  tenant: {
    settings: () => get<TenantSettings>('/api/v1/tenant/settings')
  },

  users: {
    list: () => get<{ data: ApiUser[]; count: number }>('/api/v1/users'),
    create: (body: { email: string; password: string; role: string }) =>
      post<ApiUser>('/api/v1/users', body),
    delete: (id: string) => del(`/api/v1/users/${id}`)
  },

  devices: {
    list: (params = '') => get<{ data: Device[]; total: number }>(`/api/v1/devices${params}`),
    get: (serial: string) => get<Device>(`/api/v1/devices/${serial}`),
    parameters: (serial: string, prefix = '') =>
      get<Record<string, string>>(`/api/v1/devices/${serial}/parameters?prefix=${encodeURIComponent(prefix)}`),
    tasks: (serial: string) =>
      get<{ data: Task[] }>(`/api/v1/devices/${serial}/tasks?limit=20`),
    delete: (serial: string) => del(`/api/v1/devices/${serial}`),
    reboot: (serial: string) => post(`/api/v1/devices/${serial}/reboot`),
    factoryReset: (serial: string) => post(`/api/v1/devices/${serial}/factory-reset`),
    wake: (serial: string, body?: { username?: string; password?: string }) =>
      post(`/api/v1/devices/${serial}/wake`, body ?? undefined),
    getParameters: (serial: string, names: string[]) =>
      post(`/api/v1/devices/${serial}/get-parameters`, { names }),
    setParameters: (serial: string, values: Record<string, string>) =>
      post(`/api/v1/devices/${serial}/set-parameters`, { values }),
    download: (serial: string, args: unknown) =>
      post(`/api/v1/devices/${serial}/download`, args),
    listBackups: (serial: string, limit = 20) =>
      get<{ data: BackupSummary[]; count: number }>(
        `/api/v1/devices/${serial}/backups?limit=${limit}`
      ),
    createBackup: (serial: string, label: string, refresh = false) =>
      post<BackupCreated | { message: string; task_id: string }>(
        `/api/v1/devices/${serial}/backups`,
        { label, refresh }
      ),
    getBackup: (serial: string, id: string) =>
      get<BackupDetail>(`/api/v1/devices/${serial}/backups/${id}`),
    restoreBackup: (serial: string, backupId: string) =>
      post<RestoreJobResponse>(`/api/v1/devices/${serial}/backups/${backupId}/restore`),
    deleteBackup: (serial: string, id: string) =>
      del(`/api/v1/devices/${serial}/backups/${id}`)
  },

  tasks: {
    list: (params = '') => get<{ data: Task[] }>(`/api/v1/tasks${params}`),
    get: (id: string) => get<Task>(`/api/v1/tasks/${id}`),
    cancel: (id: string) => del(`/api/v1/tasks/${id}`)
  },

  provisioning: {
    list: () => get<{ data: ProvisioningRule[] }>('/api/v1/provisioning/rules'),
    get: (id: string) => get<ProvisioningRule>(`/api/v1/provisioning/rules/${id}`),
    create: (rule: Partial<ProvisioningRule>) =>
      post<ProvisioningRule>('/api/v1/provisioning/rules', rule),
    update: (id: string, rule: Partial<ProvisioningRule>) =>
      put<ProvisioningRule>(`/api/v1/provisioning/rules/${id}`, rule),
    delete: (id: string) => del(`/api/v1/provisioning/rules/${id}`)
  }
};
