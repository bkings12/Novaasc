import { get } from 'svelte/store';
import { browser } from '$app/environment';
import { accessToken } from '$lib/auth';
import { redirect } from '@sveltejs/kit';

export function load({ url }: { url: URL }) {
  const isLogin = url.pathname === '/login';
  // Only run auth redirect in the browser (server has no access to localStorage)
  if (browser) {
    const token = get(accessToken);
    if (!token && !isLogin) throw redirect(302, '/login');
    if (token && isLogin) throw redirect(302, '/');
  }
}
