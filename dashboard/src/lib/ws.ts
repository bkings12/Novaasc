import { writable } from 'svelte/store';
import { getToken } from './auth';
import type { WSEvent } from './types';

function getWsBase(): string {
  if (import.meta.env.VITE_WS_URL) {
    const u = import.meta.env.VITE_WS_URL;
    return u.startsWith('http') ? u.replace(/^http/, 'ws') : u;
  }
  if (typeof window !== 'undefined') {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${protocol}//${window.location.host}`;
  }
  return 'ws://localhost:8080';
}

export const lastEvent = writable<WSEvent | null>(null);
export const wsConnected = writable(false);
export const liveInforms = writable<WSEvent[]>([]);

let socket: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout>;

export function connectWS(): void {
  const token = getToken();
  if (!token) return;

  const url = `${getWsBase()}/api/v1/ws?token=${encodeURIComponent(token)}`;
  socket = new WebSocket(url);

  socket.onopen = () => {
    wsConnected.set(true);
    clearTimeout(reconnectTimer);
  };

  socket.onmessage = (e) => {
    const event: WSEvent = JSON.parse(e.data as string);
    if (event.type === 'connected') return;

    lastEvent.set(event);

    if (event.type === 'device.inform') {
      liveInforms.update((prev) => [event, ...prev].slice(0, 50));
    }
  };

  socket.onclose = () => {
    wsConnected.set(false);
    reconnectTimer = setTimeout(connectWS, 5000);
  };

  socket.onerror = () => socket?.close();
}

export function disconnectWS(): void {
  clearTimeout(reconnectTimer);
  socket?.close();
  socket = null;
}
