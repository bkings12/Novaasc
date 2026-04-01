<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { lastEvent } from '$lib/ws';
  import type { Device } from '$lib/types';

  let devices: Device[] = [];
  let total = 0;
  let search = '';
  let filterOnline = '';
  let loading = true;

  async function load() {
    loading = true;
    const params = new URLSearchParams();
    if (search) params.set('search', search);
    if (filterOnline) params.set('online', filterOnline);
    params.set('limit', '100');
    const result = await api.devices.list('?' + params.toString());
    devices = result.data;
    total = result.total;
    loading = false;
  }

  onMount(load);

  $: if (
    $lastEvent?.type.startsWith('device.') ||
    $lastEvent?.type.startsWith('task.')
  ) {
    load();
  }

  async function wake(serial: string) {
    await api.devices.wake(serial);
  }

  async function reboot(serial: string) {
    if (confirm(`Reboot ${serial}?`)) {
      await api.devices.reboot(serial);
    }
  }
</script>

<div class="p-6">
  <div class="flex items-center justify-between mb-6">
    <h1 class="text-xl font-semibold">
      Devices <span class="text-gray-500 text-base ml-2">{total}</span>
    </h1>
    <div class="flex gap-2">
      <input
        bind:value={search}
        on:input={load}
        placeholder="Search serial, IP…"
        class="bg-gray-800 border border-gray-700 rounded-lg px-3 py-1.5 text-sm text-white w-48 focus:outline-none focus:border-blue-500"
      />
      <select
        bind:value={filterOnline}
        on:change={load}
        class="bg-gray-800 border border-gray-700 rounded-lg px-3 py-1.5 text-sm text-white focus:outline-none focus:border-blue-500"
      >
        <option value="">All</option>
        <option value="true">Online</option>
        <option value="false">Offline</option>
      </select>
    </div>
  </div>

  <div class="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
    <table class="w-full text-sm">
      <thead>
        <tr
          class="border-b border-gray-800 text-gray-400 text-xs uppercase tracking-wider"
        >
          <th class="px-4 py-3 text-left">Status</th>
          <th class="px-4 py-3 text-left">Serial</th>
          <th class="px-4 py-3 text-left">Manufacturer</th>
          <th class="px-4 py-3 text-left">Model</th>
          <th class="px-4 py-3 text-left">IP</th>
          <th class="px-4 py-3 text-left">SW Version</th>
          <th class="px-4 py-3 text-left">Last Inform</th>
          <th class="px-4 py-3 text-left">Actions</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-800">
        {#if loading}
          <tr>
            <td colspan="8" class="px-4 py-8 text-center text-gray-500">
              Loading…
            </td>
          </tr>
        {:else}
          {#each devices as dev}
            <tr class="hover:bg-gray-800/50 transition-colors">
              <td class="px-4 py-3">
                <span class="inline-flex items-center gap-1.5">
                  <span
                    class="w-2 h-2 rounded-full
                    {dev.online ? 'bg-green-400' : 'bg-gray-600'}"
                  >
                  </span>
                  <span
                    class="{dev.online ? 'text-green-400' : 'text-gray-500'} text-xs"
                  >
                    {dev.online ? 'Online' : 'Offline'}
                  </span>
                </span>
              </td>
              <td class="px-4 py-3">
                <a
                  href="/devices/{dev.serial_number}"
                  class="font-mono text-blue-400 hover:text-blue-300"
                >
                  {dev.serial_number}
                </a>
              </td>
              <td class="px-4 py-3 text-gray-300">{dev.manufacturer}</td>
              <td class="px-4 py-3 text-gray-400">{dev.model_name || '—'}</td>
              <td class="px-4 py-3 font-mono text-gray-400 text-xs">
                {dev.ip_address}
              </td>
              <td class="px-4 py-3 text-gray-400 text-xs">
                {dev.software_version || '—'}
              </td>
              <td class="px-4 py-3 text-gray-500 text-xs">
                {new Date(dev.last_inform).toLocaleString()}
              </td>
              <td class="px-4 py-3">
                <div class="flex gap-1">
                  <button
                    on:click={() => wake(dev.serial_number)}
                    class="px-2 py-1 bg-blue-600/20 text-blue-400 hover:bg-blue-600/40 rounded text-xs transition-colors"
                  >
                    Wake
                  </button>
                  <button
                    on:click={() => reboot(dev.serial_number)}
                    class="px-2 py-1 bg-yellow-600/20 text-yellow-400 hover:bg-yellow-600/40 rounded text-xs transition-colors"
                  >
                    Reboot
                  </button>
                  <a
                    href="/devices/{dev.serial_number}"
                    class="px-2 py-1 bg-gray-700 text-gray-300 hover:bg-gray-600 rounded text-xs transition-colors"
                  >
                    View
                  </a>
                </div>
              </td>
            </tr>
          {:else}
            <tr>
              <td colspan="8" class="px-4 py-8 text-center text-gray-500">
                No devices found
              </td>
            </tr>
          {/each}
        {/if}
      </tbody>
    </table>
  </div>
</div>
