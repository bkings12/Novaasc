<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import type { Task } from '$lib/types';

  let tasks: Task[] = [];
  let loading = true;

  onMount(async () => {
    const res = await api.tasks.list('?limit=50');
    tasks = res.data ?? [];
    loading = false;
  });
</script>

<div class="p-6">
  <h1 class="text-xl font-semibold mb-6">Tasks</h1>
  {#if loading}
    <p class="text-gray-500">Loading…</p>
  {:else}
    <div class="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-800 text-gray-400 text-xs uppercase">
            <th class="px-4 py-3 text-left">Type</th>
            <th class="px-4 py-3 text-left">Device</th>
            <th class="px-4 py-3 text-left">Status</th>
            <th class="px-4 py-3 text-left">Created</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-800">
          {#each tasks as t}
            <tr class="hover:bg-gray-800/50">
              <td class="px-4 py-3 font-mono text-gray-300">{t.type}</td>
              <td class="px-4 py-3 font-mono text-blue-400">{t.device_serial}</td>
              <td class="px-4 py-3 text-gray-400">{t.status}</td>
              <td class="px-4 py-3 text-gray-500 text-xs">
                {new Date(t.created_at).toLocaleString()}
              </td>
            </tr>
          {:else}
            <tr>
              <td colspan="4" class="px-4 py-8 text-center text-gray-500">
                No tasks
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
