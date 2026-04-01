<script lang="ts">
  import { page } from '$app/stores';
  import { api } from '$lib/api';
  import type { BackupDetail } from '$lib/types';

  const serial = $page.params.serial;
  const id = $page.params.id;
  let backup: BackupDetail | null = null;
  let error = '';

  async function load() {
    try {
      backup = await api.devices.getBackup(serial, id);
      error = '';
    } catch (e) {
      error = (e as Error).message;
      backup = null;
    }
  }

  load();
</script>

<div class="p-6 max-w-4xl">
  <a
    href="/devices/{serial}"
    class="text-sm text-gray-400 hover:text-white mb-4 inline-block"
  >
    ← Back to device
  </a>
  {#if error}
    <div class="text-red-400">{error}</div>
  {:else if backup}
    <div class="bg-gray-900 border border-gray-800 rounded-xl p-4 mb-4">
      <h1 class="text-lg font-semibold">Backup {backup.id.slice(0, 8)}…</h1>
      <div class="text-sm text-gray-400 mt-2">
        {backup.parameter_count} parameters · {backup.trigger} · SW
        {backup.software_version || '?'} · {backup.ip_address ?? '—'} ·
        {new Date(backup.created_at).toLocaleString()}
      </div>
    </div>
    <div class="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden max-h-[60vh] overflow-y-auto">
      <table class="w-full text-xs font-mono">
        <tbody class="divide-y divide-gray-800">
          {#each Object.entries(backup.parameters ?? {}) as [k, v]}
            <tr class="hover:bg-gray-800/50">
              <td class="px-4 py-2 text-gray-400 w-1/2">{k}</td>
              <td class="px-4 py-2 text-white break-all">{v}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {:else}
    <div class="text-gray-500">Loading…</div>
  {/if}
</div>
