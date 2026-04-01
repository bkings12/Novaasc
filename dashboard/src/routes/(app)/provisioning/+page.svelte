<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import type { ProvisioningRule } from '$lib/types';

  let rules: ProvisioningRule[] = [];
  let loading = true;

  onMount(async () => {
    const res = await api.provisioning.list();
    rules = res.data ?? [];
    loading = false;
  });
</script>

<div class="p-6">
  <h1 class="text-xl font-semibold mb-6">Provisioning Rules</h1>
  {#if loading}
    <p class="text-gray-500">Loading…</p>
  {:else}
    <div class="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-800 text-gray-400 text-xs uppercase">
            <th class="px-4 py-3 text-left">Name</th>
            <th class="px-4 py-3 text-left">Trigger</th>
            <th class="px-4 py-3 text-left">Active</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-800">
          {#each rules as r}
            <tr class="hover:bg-gray-800/50">
              <td class="px-4 py-3 text-gray-300">{r.name}</td>
              <td class="px-4 py-3 text-gray-400">{r.trigger}</td>
              <td class="px-4 py-3">{r.active ? 'Yes' : 'No'}</td>
            </tr>
          {:else}
            <tr>
              <td colspan="3" class="px-4 py-8 text-center text-gray-500">
                No rules
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
