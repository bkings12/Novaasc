<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import type { TenantSettings } from '$lib/types';
  import { Copy, Eye, EyeOff } from 'lucide-svelte';

  let settings: TenantSettings | null = null;
  let err = '';
  let copied = false;
  let showKey = false;

  const acsHost =
    import.meta.env.VITE_ACS_PUBLIC_HOST ??
    (typeof window !== 'undefined' ? window.location.hostname : 'localhost');

  $: cwmpUrl = settings ? `http://${acsHost}:7547/cwmp/${settings.slug}` : '';

  onMount(async () => {
    try {
      settings = await api.tenant.settings();
    } catch (e) {
      err = (e as Error).message;
    }
  });

  async function copyCwmp() {
    try {
      await navigator.clipboard.writeText(cwmpUrl);
      copied = true;
      setTimeout(() => (copied = false), 2000);
    } catch {
      err = 'Could not copy to clipboard';
    }
  }
</script>

<div class="p-6 max-w-2xl">
  <h1 class="text-xl font-semibold mb-2">Tenant</h1>
  <p class="text-sm text-gray-500 mb-6">ACS connection details for your CPEs.</p>

  {#if err && !settings}
    <div class="bg-red-900/20 border border-red-800 text-red-400 rounded-lg p-4 text-sm">{err}</div>
  {:else if !settings}
    <p class="text-gray-500">Loading…</p>
  {:else}
    <div class="space-y-6">
      <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
        <div class="text-xs text-gray-500 uppercase tracking-wider mb-1">Name</div>
        <div class="text-white">{settings.name}</div>
      </div>
      <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
        <div class="text-xs text-gray-500 uppercase tracking-wider mb-1">Slug</div>
        <div class="font-mono text-white">{settings.slug}</div>
      </div>
      <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
        <div class="text-xs text-gray-500 uppercase tracking-wider mb-2">CWMP URL (configure on CPEs)</div>
        <div class="flex gap-2 items-center flex-wrap">
          <code class="text-sm text-blue-300 break-all flex-1 min-w-0">{cwmpUrl}</code>
          <button
            type="button"
            on:click={copyCwmp}
            class="shrink-0 flex items-center gap-1 px-3 py-1.5 bg-gray-800 hover:bg-gray-700 rounded-lg text-sm text-gray-200"
          >
            <Copy size={14} />
            {copied ? 'Copied' : 'Copy'}
          </button>
        </div>
        <p class="text-xs text-gray-600 mt-2">
          Host comes from <span class="font-mono">VITE_ACS_PUBLIC_HOST</span> or the dashboard hostname; port
          <span class="font-mono">7547</span> must reach your ACS.
        </p>
      </div>
      <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
        <div class="flex items-center justify-between gap-2 mb-2">
          <div class="text-xs text-gray-500 uppercase tracking-wider">API key</div>
          <button
            type="button"
            class="p-1 text-gray-500 hover:text-white"
            on:click={() => (showKey = !showKey)}
            aria-label="Toggle API key visibility"
          >
            {#if showKey}
              <EyeOff size={16} />
            {:else}
              <Eye size={16} />
            {/if}
          </button>
        </div>
        <div class="font-mono text-sm text-white break-all">
          {showKey ? settings.api_key || '—' : settings.api_key ? '•'.repeat(Math.min(32, settings.api_key.length)) : '—'}
        </div>
      </div>
      <div class="grid grid-cols-2 gap-4 text-sm">
        <div class="bg-gray-900/50 border border-gray-800 rounded-lg p-3">
          <div class="text-xs text-gray-500">Plan</div>
          <div class="text-white mt-1">{settings.plan}</div>
        </div>
        <div class="bg-gray-900/50 border border-gray-800 rounded-lg p-3">
          <div class="text-xs text-gray-500">Max devices</div>
          <div class="text-white mt-1">{settings.max_devices}</div>
        </div>
      </div>
    </div>
  {/if}
</div>
