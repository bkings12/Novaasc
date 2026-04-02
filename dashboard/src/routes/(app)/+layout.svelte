<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { page } from '$app/stores';
  import { logout, currentUser } from '$lib/auth';
  import { connectWS, disconnectWS, wsConnected } from '$lib/ws';
  import {
    LayoutDashboard,
    Router,
    ListTodo,
    Settings,
    Wifi,
    WifiOff,
    LogOut,
    Building2,
    Users
  } from 'lucide-svelte';

  onMount(connectWS);
  onDestroy(disconnectWS);

  const navMain = [
    { href: '/', label: 'Dashboard', icon: LayoutDashboard },
    { href: '/devices', label: 'Devices', icon: Router },
    { href: '/tasks', label: 'Tasks', icon: ListTodo },
    { href: '/provisioning', label: 'Provisioning', icon: Settings }
  ];

  const navAdmin = [
    { href: '/settings/tenant', label: 'Tenant', icon: Building2 },
    { href: '/settings/users', label: 'Users', icon: Users }
  ];
</script>

<div class="flex h-screen bg-gray-950 text-white">
  <aside
    class="w-56 bg-gray-900 border-r border-gray-800 flex flex-col"
  >
    <div class="p-4 border-b border-gray-800">
      <span class="font-bold text-lg">NovaACS</span>
      <div class="text-xs text-gray-500 mt-0.5">
        {$currentUser?.tenant ?? $currentUser?.tenant_id ?? ''}
      </div>
    </div>

    <nav class="flex-1 p-3 space-y-1 overflow-y-auto">
      {#each navMain as item}
        {@const active =
          $page.url.pathname === item.href ||
          ($page.url.pathname.startsWith(item.href + '/') && item.href !== '/')}
        <a
          href={item.href}
          class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors
            {active
              ? 'bg-blue-600 text-white'
              : 'text-gray-400 hover:bg-gray-800 hover:text-white'}"
        >
          <svelte:component this={item.icon} size={16} />
          {item.label}
        </a>
      {/each}
      {#if $currentUser?.role === 'admin'}
        <div class="pt-3 mt-2 border-t border-gray-800 text-[10px] uppercase tracking-wider text-gray-600 px-3">
          Settings
        </div>
        {#each navAdmin as item}
          {@const active = $page.url.pathname === item.href || $page.url.pathname.startsWith(item.href + '/')}
          <a
            href={item.href}
            class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors
              {active
                ? 'bg-blue-600 text-white'
                : 'text-gray-400 hover:bg-gray-800 hover:text-white'}"
          >
            <svelte:component this={item.icon} size={16} />
            {item.label}
          </a>
        {/each}
      {/if}
    </nav>

    <div class="p-3 border-t border-gray-800 space-y-2">
      <div
        class="flex items-center gap-2 px-3 py-1.5 text-xs
          {$wsConnected ? 'text-green-400' : 'text-gray-500'}"
      >
        <svelte:component this={$wsConnected ? Wifi : WifiOff} size={12} />
        {$wsConnected ? 'Live' : 'Disconnected'}
      </div>
      <button
        on:click={logout}
        class="flex items-center gap-3 w-full px-3 py-2 rounded-lg text-sm text-gray-400 hover:bg-gray-800 hover:text-white transition-colors"
      >
        <LogOut size={16} />
        Sign out
      </button>
    </div>
  </aside>

  <main class="flex-1 overflow-auto">
    <slot />
  </main>
</div>
