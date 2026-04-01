<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { liveInforms } from '$lib/ws';
  import type { Stats } from '$lib/types';

  let stats: Stats | null = null;

  onMount(async () => {
    stats = await api.stats();
  });
</script>

<div class="p-6">
  <h1 class="text-xl font-semibold mb-6">Dashboard</h1>

  {#if stats}
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
      <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
        <div class="text-2xl font-bold text-green-400">
          {stats.devices.online}
        </div>
        <div class="text-sm text-gray-400 mt-1">Online</div>
      </div>
      <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
        <div class="text-2xl font-bold text-gray-400">
          {stats.devices.offline}
        </div>
        <div class="text-sm text-gray-400 mt-1">Offline</div>
      </div>
      <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
        <div class="text-2xl font-bold text-yellow-400">
          {stats.tasks.pending}
        </div>
        <div class="text-sm text-gray-400 mt-1">Pending Tasks</div>
      </div>
      <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
        <div class="text-2xl font-bold text-red-400">
          {stats.tasks.failed}
        </div>
        <div class="text-sm text-gray-400 mt-1">Failed Tasks</div>
      </div>
    </div>
  {/if}

  <div class="bg-gray-900 border border-gray-800 rounded-xl">
    <div class="px-4 py-3 border-b border-gray-800 text-sm font-medium flex items-center justify-between">
      <span>Live Informs</span>
      <span class="text-xs font-normal text-gray-500">Events appear when devices contact the ACS (periodic or after Wake)</span>
    </div>
    <div class="divide-y divide-gray-800 max-h-96 overflow-y-auto">
      {#each $liveInforms as ev}
        <div class="px-4 py-3 text-sm flex items-center gap-4">
          <span class="text-green-400 font-mono text-xs w-28 shrink-0">
            {(ev.payload as { serial?: string }).serial ?? '—'}
          </span>
          <span class="text-gray-300">
            {(ev.payload as { manufacturer?: string }).manufacturer ?? '—'}
          </span>
          <span class="text-gray-500 text-xs">
            {(ev.payload as { ip?: string }).ip ?? '—'}
          </span>
          <span class="text-gray-600 text-xs ml-auto">
            {new Date(ev.time).toLocaleTimeString()}
          </span>
        </div>
      {:else}
        <div class="px-4 py-8 text-center text-gray-500 text-sm space-y-1">
          <p>No recent informs in this session.</p>
          <p class="text-gray-600 text-xs">Check that the sidebar shows <strong class="text-green-400">Live</strong>. Use <strong>Devices → Wake</strong> to trigger a connection, or wait for the device’s next periodic Inform.</p>
        </div>
      {/each}
    </div>
  </div>
</div>
