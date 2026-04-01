<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { api } from '$lib/api';
  import { lastEvent } from '$lib/ws';
  import type { Device, Task } from '$lib/types';

  const serial = $page.params.serial;
  let device: Device | null = null;
  let tasks: Task[] = [];
  let params: Record<string, string> = {};
  let wifiParams: Record<string, string> = {};
  let paramPrefix = 'Device.DeviceInfo.';
  let tab = 'overview';

  let setKey = '';
  let setValue = '';
  let setMsg = '';

  // WiFi tab (TR-181: Device.WiFi.SSID.1.SSID, KeyPassphrase, ModeEnabled)
  let wifiSSID = '';
  let wifiPassword = '';
  let wifiSecurity = 'WPA2-Personal';
  let wifiFetching = false;
  let wifiAutoFetched = false;

  // Pre-fill SSID and security from WiFi parameters only when the relevant keys exist
  $: {
    const ssid = wifiParams['Device.WiFi.SSID.1.SSID'];
    const sec = wifiParams['Device.WiFi.AccessPoint.1.Security.ModeEnabled'];
    if (ssid !== undefined) wifiSSID = ssid;
    if (sec !== undefined) wifiSecurity = sec || 'WPA2-Personal';
  }

  // Refresh WiFi panel when backend broadcasts parameters_updated for this device
  $: {
    const ev = $lastEvent;
    if (ev?.type === 'device.parameters_updated' && (ev.payload as { serial?: string })?.serial === serial) {
      loadWifiParams();
    }
  }

  // Auto-fetch WiFi from device when user opens WiFi tab and we have no data (once per page load)
  $: if (tab === 'wifi' && !wifiFetching && !wifiAutoFetched && Object.keys(wifiParams).length === 0) {
    wifiAutoFetched = true;
    fetchWifiFromDevice();
  }

  onMount(async () => {
    device = await api.devices.get(serial);
    const tasksRes = await api.devices.tasks(serial);
    tasks = tasksRes.data ?? [];
    params = await api.devices.parameters(serial, paramPrefix);
    wifiParams = await api.devices.parameters(serial, 'Device.WiFi.');
  });

  async function loadParams() {
    params = await api.devices.parameters(serial, paramPrefix);
  }

  async function loadWifiParams() {
    wifiParams = await api.devices.parameters(serial, 'Device.WiFi.');
  }

  async function fetchWifiFromDevice() {
    wifiFetching = true;
    setMsg = '';
    try {
      await api.devices.getParameters(serial, [
        'Device.WiFi.SSID.1.SSID',
        'Device.WiFi.Radio.1.Channel',
        'Device.WiFi.Radio.1.OperatingFrequencyBand',
        'Device.WiFi.Radio.1.OperatingStandards',
        'Device.WiFi.AccessPoint.1.Security.ModeEnabled',
        'Device.WiFi.AccessPoint.1.AssociatedDeviceNumberOfEntries'
      ]);
      await api.devices.wake(serial);
      setMsg = 'Fetching WiFi params from device — refreshing in 8s…';
      setTimeout(async () => {
        await loadWifiParams();
        setMsg = '';
        wifiFetching = false;
      }, 8000);
    } catch (e) {
      setMsg = `Error: ${(e as Error).message}`;
      wifiFetching = false;
    }
  }

  async function doReboot() {
    if (confirm(`Reboot ${serial}?`)) {
      await api.devices.reboot(serial);
      setMsg = 'Reboot task queued';
    }
  }

  async function doWake() {
    await api.devices.wake(serial);
    setMsg = 'Wake sent';
  }

  async function doSet() {
    if (!setKey || !setValue) return;
    await api.devices.setParameters(serial, { [setKey]: setValue });
    setMsg = `Set ${setKey} queued`;
    setKey = '';
    setValue = '';
  }

  async function applyWiFi() {
    const values: Record<string, string> = {};
    if (wifiSSID.trim()) {
      values['Device.WiFi.SSID.1.SSID'] = wifiSSID.trim();
    }
    if (wifiSecurity) {
      values['Device.WiFi.AccessPoint.1.Security.ModeEnabled'] = wifiSecurity;
    }
    if (wifiPassword) {
      values['Device.WiFi.AccessPoint.1.Security.KeyPassphrase'] = wifiPassword;
    }
    if (Object.keys(values).length === 0) {
      setMsg = 'Nothing to change';
      return;
    }
    await api.devices.setParameters(serial, values);
    setMsg = `WiFi settings queued (${Object.keys(values).join(', ')})`;
    await api.devices.wake(serial);
    setMsg += ' — Wake sent, changes apply in ~5 seconds';
    // Refresh WiFi params after a short delay so status updates once device reports back
    setTimeout(loadWifiParams, 8000);
  }

  function statusColor(s: string) {
    const map: Record<string, string> = {
      complete: 'text-green-400 bg-green-900/20',
      pending: 'text-yellow-400 bg-yellow-900/20',
      dispatched: 'text-blue-400 bg-blue-900/20',
      failed: 'text-red-400 bg-red-900/20',
      timeout: 'text-orange-400 bg-orange-900/20',
      cancelled: 'text-gray-400 bg-gray-800'
    };
    return map[s] ?? 'text-gray-400';
  }

  // Backup tab
  let backups: { id: string; parameter_count: number; trigger: string; software_version?: string; ip_address?: string; created_at: string }[] = [];
  let backupLoading = false;
  let backupMsg = '';
  let restoring = false;

  async function loadBackups() {
    try {
      const r = await api.devices.listBackups(serial);
      backups = r.data ?? [];
    } catch (_) {
      backups = [];
    }
  }

  async function takeBackup() {
    backupLoading = true;
    backupMsg = '';
    try {
      const b = await api.devices.createBackup(serial, 'manual');
      if ('parameter_count' in b) {
        backupMsg = `Backup created: ${b.parameter_count} parameters`;
      } else {
        backupMsg = (b as { message?: string }).message ?? 'Backup created';
      }
      await loadBackups();
    } catch (e: unknown) {
      backupMsg = (e as Error).message;
    } finally {
      backupLoading = false;
    }
  }

  async function refreshAndBackup() {
    backupLoading = true;
    backupMsg = '';
    try {
      const r = await api.devices.createBackup(serial, 'manual', true);
      if ('task_id' in r) {
        backupMsg =
          'GetParameterValues queued — click "Backup Now" after task completes (~30s)';
        await api.devices.wake(serial);
      } else {
        backupMsg = `Backup created: ${(r as { parameter_count?: number }).parameter_count ?? 0} parameters`;
        await loadBackups();
      }
    } catch (e: unknown) {
      backupMsg = (e as Error).message;
    } finally {
      backupLoading = false;
    }
  }

  async function restoreBackup(backupId: string, label: string) {
    if (
      !confirm(
        `Restore backup "${label}"?\n\nThis will push all stored parameters back to the device.\nThe device may restart or drop connections.`
      )
    )
      return;
    restoring = true;
    backupMsg = '';
    try {
      const job = await api.devices.restoreBackup(serial, backupId);
      backupMsg = job.message;
    } catch (e: unknown) {
      backupMsg = (e as Error).message;
    } finally {
      restoring = false;
    }
  }

  // Load backups when user opens Backup tab (reactive on tab change)
  $: if (tab === 'backup') {
    loadBackups();
  }
</script>

<div class="p-6">
  {#if !device}
    <div class="text-gray-500">Loading…</div>
  {:else}
    <div class="flex items-start justify-between mb-6">
      <div>
        <div class="flex items-center gap-3">
          <span
            class="w-3 h-3 rounded-full
            {device.online ? 'bg-green-400' : 'bg-gray-600'}"
          >
          </span>
          <h1 class="text-xl font-semibold font-mono">
            {device.serial_number}
          </h1>
        </div>
        <p class="text-gray-400 text-sm mt-1">
          {device.manufacturer} · {device.model_name || device.product_class} ·
          {device.ip_address}
        </p>
      </div>
      <div class="flex gap-2">
        <button
          on:click={doWake}
          class="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm transition-colors"
        >
          Wake
        </button>
        <button
          on:click={doReboot}
          class="px-3 py-1.5 bg-yellow-600 hover:bg-yellow-700 text-white rounded-lg text-sm transition-colors"
        >
          Reboot
        </button>
      </div>
    </div>

    {#if setMsg}
      <div
        class="bg-green-900/20 border border-green-700 text-green-400 rounded-lg p-3 mb-4 text-sm"
      >
        {setMsg}
      </div>
    {/if}

    <div class="flex gap-1 mb-4 border-b border-gray-800">
      {#each ['overview', 'wifi', 'parameters', 'tasks', 'backup', 'set'] as t}
        <button
          on:click={() => (tab = t)}
          class="px-4 py-2 text-sm transition-colors
            {tab === t
              ? 'text-white border-b-2 border-blue-500'
              : 'text-gray-400 hover:text-white'}"
        >
          {t.charAt(0).toUpperCase() + t.slice(1)}
        </button>
      {/each}
    </div>

    {#if tab === 'overview'}
      <div class="grid grid-cols-2 gap-4">
        {#each [
          ['Serial', device.serial_number],
          ['OUI', device.oui],
          ['SW Version', device.software_version],
          ['HW Version', device.hardware_version],
          ['IP Address', device.ip_address],
          ['MAC', device.mac_address],
          ['Last Inform', new Date(device.last_inform).toLocaleString()],
          ['First Seen', new Date(device.first_seen).toLocaleString()]
        ] as [label, value]}
          <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
            <div class="text-xs text-gray-500 uppercase tracking-wider">
              {label}
            </div>
            <div class="text-sm text-white mt-1 font-mono break-all">
              {value || '—'}
            </div>
          </div>
        {/each}
      </div>

    {:else if tab === 'wifi'}
      <div class="max-w-lg space-y-4">
        <!-- Current WiFi Status (from wifiParams loaded via parameters API) -->
        <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
          <div class="flex items-center justify-between mb-3">
            <h3 class="text-xs text-gray-500 uppercase tracking-wider">
              Current WiFi Status
            </h3>
            <div class="flex gap-3">
              <button
                type="button"
                on:click={fetchWifiFromDevice}
                disabled={wifiFetching}
                class="text-xs text-yellow-400 hover:text-yellow-300 disabled:opacity-40"
              >
                {wifiFetching ? 'Fetching…' : 'Fetch from device'}
              </button>
              <button
                type="button"
                on:click={loadWifiParams}
                disabled={wifiFetching}
                class="text-xs text-blue-400 hover:text-blue-300 disabled:opacity-40"
              >
                Refresh
              </button>
            </div>
          </div>
          {#if Object.keys(wifiParams).length === 0}
            <p class="text-xs text-yellow-500/80">
              No WiFi data cached yet — click <span class="font-medium">Fetch from device</span> to pull live values.
            </p>
          {/if}
          <div class="space-y-2 text-sm">
            <div class="flex justify-between">
              <span class="text-gray-400">SSID</span>
              <span class="text-white font-mono">
                {wifiParams['Device.WiFi.SSID.1.SSID'] || '—'}
              </span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-400">Security</span>
              <span class="text-white">
                {wifiParams['Device.WiFi.AccessPoint.1.Security.ModeEnabled'] || '—'}
              </span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-400">Channel</span>
              <span class="text-white">
                {wifiParams['Device.WiFi.Radio.1.Channel'] || '—'}
              </span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-400">Band</span>
              <span class="text-white">
                {wifiParams['Device.WiFi.Radio.1.OperatingFrequencyBand'] || '—'}
              </span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-400">Standard</span>
              <span class="text-white">
                {wifiParams['Device.WiFi.Radio.1.OperatingStandards'] || '—'}
              </span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-400">Connected Clients</span>
              <span class="text-white">
                {wifiParams['Device.WiFi.AccessPoint.1.AssociatedDeviceNumberOfEntries'] || '0'}
              </span>
            </div>
          </div>
        </div>

        <!-- Change WiFi Settings -->
        <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
          <h3 class="text-xs text-gray-500 uppercase tracking-wider mb-3">
            Change WiFi Settings
          </h3>
          <div class="space-y-3">
            <div>
              <label for="wifi-ssid" class="text-xs text-gray-400">SSID (Network Name)</label>
              <input
                id="wifi-ssid"
                type="text"
                autocomplete="off"
                bind:value={wifiSSID}
                placeholder="e.g. MyNetwork"
                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white mt-1 focus:outline-none focus:border-blue-500"
              />
            </div>
            <div>
              <label for="wifi-security" class="text-xs text-gray-400">Security Mode</label>
              <select
                id="wifi-security"
                bind:value={wifiSecurity}
                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white mt-1 focus:outline-none focus:border-blue-500"
              >
                <option value="None">None (Open)</option>
                <option value="WPA2-Personal">WPA2-Personal</option>
                <option value="WPA-WPA2-Personal">WPA + WPA2 Personal</option>
              </select>
            </div>
            <div>
              <label for="wifi-password" class="text-xs text-gray-400">
                Password
                <span class="text-gray-600 ml-1">(leave blank to keep current)</span>
              </label>
              <input
                id="wifi-password"
                type="password"
                autocomplete="new-password"
                bind:value={wifiPassword}
                placeholder="leave blank to keep current"
                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white mt-1 focus:outline-none focus:border-blue-500"
              />
            </div>
            <button
              on:click={applyWiFi}
              class="w-full bg-blue-600 hover:bg-blue-700 text-white rounded-lg py-2 text-sm font-medium transition-colors"
            >
              Queue WiFi Changes
            </button>
            <p class="text-xs text-gray-600">
              Changes apply on next device contact. Use Wake to apply immediately.
              The device may disconnect briefly when WiFi settings change.
            </p>
          </div>
        </div>

        <!-- Connected clients -->
        {#if wifiParams['Device.WiFi.AccessPoint.1.AssociatedDevice.1.MACAddress']}
          <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
            <h3 class="text-xs text-gray-500 uppercase tracking-wider mb-3">
              Connected Clients
            </h3>
            <div class="text-sm space-y-2">
              <div class="flex justify-between">
                <span class="text-gray-400">MAC</span>
                <span class="font-mono text-white">
                  {wifiParams['Device.WiFi.AccessPoint.1.AssociatedDevice.1.MACAddress']}
                </span>
              </div>
              <div class="flex justify-between">
                <span class="text-gray-400">Signal</span>
                <span class="text-white">
                  {wifiParams['Device.WiFi.AccessPoint.1.AssociatedDevice.1.SignalStrength']} dBm
                </span>
              </div>
              <div class="flex justify-between">
                <span class="text-gray-400">TX Rate</span>
                <span class="text-white text-xs">
                  {wifiParams['Device.WiFi.AccessPoint.1.AssociatedDevice.1.X_MIKROTIK_Stats.TxRate'] || '—'}
                </span>
              </div>
              <div class="flex justify-between">
                <span class="text-gray-400">RX Rate</span>
                <span class="text-white text-xs">
                  {wifiParams['Device.WiFi.AccessPoint.1.AssociatedDevice.1.X_MIKROTIK_Stats.RxRate'] || '—'}
                </span>
              </div>
            </div>
          </div>
        {/if}
      </div>

    {:else if tab === 'backup'}
      <div class="space-y-4 max-w-2xl">
        <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
          <h3 class="text-xs text-gray-500 uppercase tracking-wider mb-3">
            Create Backup
          </h3>
          <div class="flex gap-2">
            <button
              on:click={takeBackup}
              disabled={backupLoading}
              class="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm transition-colors"
            >
              {backupLoading ? 'Working…' : 'Backup Now'}
            </button>
            <button
              on:click={refreshAndBackup}
              disabled={backupLoading}
              class="px-4 py-2 bg-gray-700 hover:bg-gray-600 disabled:opacity-50 text-white rounded-lg text-sm transition-colors"
            >
              Refresh Params + Backup
            </button>
          </div>
          <p class="text-xs text-gray-600 mt-2">
            "Backup Now" snapshots parameters already in the database. "Refresh +
            Backup" fetches latest values from device first.
          </p>
          {#if backupMsg}
            <div
              class="mt-3 text-sm text-green-400 bg-green-900/20 border border-green-800 rounded-lg p-3"
            >
              {backupMsg}
            </div>
          {/if}
        </div>

        <div
          class="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden"
        >
          <div
            class="px-4 py-3 border-b border-gray-800 flex items-center justify-between"
          >
            <span class="text-sm font-medium">Backup History</span>
            <button
              on:click={loadBackups}
              class="text-xs text-gray-500 hover:text-white"
            >
              Refresh
            </button>
          </div>

          {#if backups.length === 0}
            <div class="px-4 py-8 text-center text-gray-500 text-sm">
              No backups yet. Click "Backup Now" to create one.
            </div>
          {:else}
            <div class="divide-y divide-gray-800">
              {#each backups as b}
                <div class="px-4 py-3 flex items-center gap-4">
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2">
                      <span class="text-sm text-white">
                        {new Date(b.created_at).toLocaleString()}
                      </span>
                      <span
                        class="text-xs px-1.5 py-0.5 rounded bg-gray-800 text-gray-400"
                      >
                        {b.trigger}
                      </span>
                    </div>
                    <div class="text-xs text-gray-500 mt-0.5">
                      {b.parameter_count} parameters · SW
                      {b.software_version || '?'} · {b.ip_address ?? '—'}
                    </div>
                  </div>
                  <div class="flex gap-2 shrink-0">
                    <a
                      href="/devices/{serial}/backups/{b.id}"
                      class="px-2 py-1 bg-gray-700 hover:bg-gray-600 text-gray-300 rounded text-xs"
                    >
                      View
                    </a>
                    <button
                      on:click={() =>
                        restoreBackup(b.id, new Date(b.created_at).toLocaleString())}
                      disabled={restoring}
                      class="px-2 py-1 bg-yellow-600/20 hover:bg-yellow-600/40 text-yellow-400 rounded text-xs transition-colors disabled:opacity-50"
                    >
                      Restore
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>

    {:else if tab === 'parameters'}
      <div class="mb-3 flex gap-2">
        <input
          bind:value={paramPrefix}
          class="bg-gray-800 border border-gray-700 rounded-lg px-3 py-1.5 text-sm text-white flex-1 focus:outline-none focus:border-blue-500"
          placeholder="Device.DeviceInfo."
        />
        <button
          on:click={loadParams}
          class="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm"
        >
          Filter
        </button>
      </div>
      <div
        class="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden max-h-[60vh] overflow-y-auto"
      >
        <table class="w-full text-xs font-mono">
          <tbody class="divide-y divide-gray-800">
            {#each Object.entries(params) as [k, v]}
              <tr class="hover:bg-gray-800/50">
                <td class="px-4 py-2 text-gray-400 w-1/2">{k}</td>
                <td class="px-4 py-2 text-white break-all">{v}</td>
              </tr>
            {:else}
              <tr>
                <td colspan="2" class="px-4 py-8 text-center text-gray-500">
                  No parameters found for prefix "{paramPrefix}"
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

    {:else if tab === 'tasks'}
      <div class="space-y-2">
        {#each tasks as t}
          <div
            class="bg-gray-900 border border-gray-800 rounded-xl p-4 flex items-center gap-4"
          >
            <span class="font-mono text-xs text-gray-500 w-28 shrink-0">
              {t.type}
            </span>
            <span
              class="px-2 py-0.5 rounded text-xs {statusColor(t.status)}"
            >
              {t.status}
            </span>
            <span class="text-xs text-gray-500 ml-auto">
              {new Date(t.created_at).toLocaleString()}
            </span>
          </div>
        {:else}
          <div class="text-center text-gray-500 py-8 text-sm">
            No tasks yet
          </div>
        {/each}
      </div>

    {:else if tab === 'set'}
      <div class="max-w-lg">
        <div class="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <h2 class="text-sm font-medium mb-4">Set Parameter Value</h2>
          <div class="space-y-3">
            <input
              bind:value={setKey}
              placeholder="Device.ManagementServer.PeriodicInformInterval"
              class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white font-mono focus:outline-none focus:border-blue-500"
            />
            <input
              bind:value={setValue}
              placeholder="300"
              class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-blue-500"
            />
            <button
              on:click={doSet}
              class="w-full bg-blue-600 hover:bg-blue-700 text-white rounded-lg py-2 text-sm transition-colors"
            >
              Queue SetParameterValues
            </button>
          </div>
        </div>
      </div>
    {/if}
  {/if}
</div>
