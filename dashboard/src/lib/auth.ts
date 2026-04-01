import { writable, get } from 'svelte/store';
import type { TokenPair } from './types';

const API =
  import.meta.env.VITE_API_URL ??
  (typeof window !== 'undefined' ? '' : 'http://localhost:8080');

function persisted<T>(key: string, initial: T) {
  const stored =
    typeof localStorage !== 'undefined' ? localStorage.getItem(key) : null;
  const store = writable<T>(stored ? JSON.parse(stored) : initial);
  store.subscribe((v) => {
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(key, JSON.stringify(v));
    }
  });
  return store;
}

export const accessToken = persisted<string>('access_token', '');
export const refreshToken = persisted<string>('refresh_token', '');
export const currentUser = persisted<{ email?: string; tenant_id?: string } | null>('user', null);

export async function login(
  tenant: string,
  email: string,
  password: string
): Promise<void> {
  const res = await fetch(`${API}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ tenant, email, password })
  });
  if (!res.ok) throw new Error('Invalid credentials');

  const pair: TokenPair = await res.json();
  accessToken.set(pair.access_token);
  refreshToken.set(pair.refresh_token);

  const meRes = await fetch(`${API}/api/v1/auth/me`, {
    headers: { Authorization: `Bearer ${pair.access_token}` }
  });
  if (meRes.ok) {
    currentUser.set(await meRes.json());
  }
}

export function logout(): void {
  accessToken.set('');
  refreshToken.set('');
  currentUser.set(null);
  if (typeof window !== 'undefined') {
    window.location.href = '/login';
  }
}

export function getToken(): string {
  return get(accessToken);
}
