<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { api } from '$lib/api';
  import { lastEvent } from '$lib/ws';
  import type { Device, Task } from '$lib/types';
  import {
    detectNamespace,
    getWiFiDisplay,
    igdWifiGetParameterNames,
    igdWifiSetPayload,
    tr181WifiSetPayload,
    parseIgdLanHosts,
    deviceSummaryHasVoiceService,
    getIgdWanPpp,
    igdWanSetPayload,
    getIgdOptics,
    parseRxDbm,
    rxPowerBadgeClass,
    formatUptimeSeconds,
    getIgdVoip,
    type ParamNamespace
  } from '$lib/deviceParams';
  import { Eye, EyeOff } from 'lucide-svelte';

  const serial = $page.params.serial as string;
  let device: Device | null = null;
  let tasks: Task[] = [];
  /** Merged TR-069 parameters (device document + optional prefix fetches). */
  let mergedParams: Record<string, string> = {};
  let params: Record<string, string> = {};
  let wifiParams: Record<string, string> = {};
  let paramPrefix = 'Device.DeviceInfo.';
  let tab = 'overview';

  let setKey = '';
  let setValue = '';
  let setMsg = '';

  $: ns = detectNamespace(mergedParams) as ParamNamespace;
  $: showVoipTab = ns === 'igd' && deviceSummaryHasVoiceService(mergedParams);
  $: igdExtraTabs = ns === 'igd'
    ? showVoipTab
      ? (['wan', 'optics', 'lan', 'voip'] as const)
      : (['wan', 'optics', 'lan'] as const)
    : ([] as string[]);
  $: mainTabs = ['overview', 'wifi', ...igdExtraTabs, 'parameters', 'tasks', 'backup', 'set'] as string[];
  $: wifiView = getWiFiDisplay(ns === 'igd' ? mergedParams : wifiParams, ns);
  $: lanHosts = ns === 'igd' ? parseIgdLanHosts(mergedParams) : [];
  $: wanPpp = ns === 'igd' ? getIgdWanPpp(mergedParams) : null;
  $: optics = ns === 'igd' ? getIgdOptics(mergedParams) : null;
  $: voipInfo = ns === 'igd' ? getIgdVoip(mergedParams) : null;
  $: rxNum = optics ? parseRxDbm(optics.rxDbm) : null;

  let wifiSSID = '';
  let wifiSSID5 = '';
  let wifiPassword = '';
  let wifiPassword5 = '';
  let wifiSecurity = 'WPA2-Personal';
  let wifiFetching = false;
  let wifiAutoFetched = false;

  let wanUser = '';
  let wanPass = '';
  let showPppPassword = false;
  let wanFormInit = false;

  $: {
    if (ns === 'igd') {
      if (wifiView.ssid24 !== undefined && wifiView.ssid24 !== '') wifiSSID = wifiView.ssid24;
      if (wifiView.ssid5 !== undefined && wifiView.ssid5 !== '') wifiSSID5 = wifiView.ssid5;
    } else {
      if (wifiView.ssid24 !== undefined && wifiView.ssid24 !== '') wifiSSID = wifiView.ssid24;
      if (wifiView.security) wifiSecurity = wifiView.security || 'WPA2-Personal';
    }
  }

  $: if (tab === 'wan' && wanPpp && !wanFormInit) {
    wanUser = wanPpp.username;
    wanFormInit = true;
  }

  $: {
    const ev = $lastEvent;
    if (ev?.type === 'device.parameters_updated' && (ev.payload as { serial?: string })?.serial === serial) {
      reloadFromDevice();
    }
  }

  $: if (tab === 'wifi' && !wifiFetching && !wifiAutoFetched) {
    const src = ns === 'igd' ? mergedParams : wifiParams;
    const hasWifi =
      ns === 'igd'
        ? Object.keys(src).some((k) => k.includes('WLANConfiguration'))
        : Object.keys(src).some((k) => k.startsWith('Device.WiFi.'));
    if (!hasWifi) {
      wifiAutoFetched = true;
      fetchWifiFromDevice();
    }
  }

  async function reloadFromDevice() {
    if (!device) return;
    device = await api.devices.get(serial);
    mergedParams = { ...(device.parameters ?? {}) };
    const branch = detectNamespace(mergedParams);
    if (branch === 'igd') {
      try {
        const h = await api.devices.parameters(
          serial,
          'InternetGatewayDevice.LANDevice.1.Hosts.'
        );
        mergedParams = { ...mergedParams, ...h };
      } catch (_) {
        /* ignore */
      }
    }
    wanFormInit = false;
    await loadWifiParams();
    await loadParams();
  }

  function igdWanLooksConnected(status: string): boolean {
    const s = (status || '').toLowerCase();
    return s.includes('connected') || s === 'up' || s.includes('online');
  }

  onMount(async () => {
    device = await api.devices.get(serial);
    const tasksRes = await api.devices.tasks(serial);
    tasks = tasksRes.data ?? [];
    mergedParams = { ...(device.parameters ?? {}) };
    const detected = detectNamespace(mergedParams);
    if (detected === 'igd') {
      paramPrefix = 'InternetGatewayDevice.DeviceInfo.';
      try {
        const h = await api.devices.parameters(serial, 'InternetGatewayDevice.LANDevice.1.Hosts.');
        mergedParams = { ...mergedParams, ...h };
      } catch (_) {
        /* ignore */
      }
    } else {
      paramPrefix = 'Device.DeviceInfo.';
    }
    params = await api.devices.parameters(serial, paramPrefix);
    wifiParams = await api.devices.parameters(serial, 'Device.WiFi.');
  });

  async function loadParams() {
    params = await api.devices.parameters(serial, paramPrefix);
  }

  async function loadWifiParams() {
    if (ns === 'igd') {
      const w = await api.devices.parameters(
        serial,
        'InternetGatewayDevice.LANDevice.1.WLANConfiguration.'
      );
      mergedParams = { ...mergedParams, ...w };
    } else {
      wifiParams = await api.devices.parameters(serial, 'Device.WiFi.');
    }
  }

  async function fetchWifiFromDevice() {
    wifiFetching = true;
    setMsg = '';
    try {
      const names =
        ns === 'igd'
          ? igdWifiGetParameterNames()
          : [
              'Device.WiFi.SSID.1.SSID',
              'Device.WiFi.Radio.1.Channel',
              'Device.WiFi.Radio.1.OperatingFrequencyBand',
              'Device.WiFi.Radio.1.OperatingStandards',
              'Device.WiFi.AccessPoint.1.Security.ModeEnabled',
              'Device.WiFi.AccessPoint.1.AssociatedDeviceNumberOfEntries'
            ];
      await api.devices.getParameters(serial, names);
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
    let values: Record<string, string> = {};
    if (ns === 'igd') {
      values = igdWifiSetPayload({
        ssid24: wifiSSID,
        ssid5: wifiSSID5,
        pass24: wifiPassword || undefined,
        pass5: wifiPassword5 || undefined
      });
    } else {
      values = tr181WifiSetPayload({
        ssid: wifiSSID,
        security: wifiSecurity,
        passphrase: wifiPassword || undefined
      });
    }
    if (Object.keys(values).length === 0) {
      setMsg = 'Nothing to change';
      return;
    }
    await api.devices.setParameters(serial, values);
    setMsg = `WiFi settings queued (${Object.keys(values).length} params)`;
    await api.devices.wake(serial);
    setMsg += ' — Wake sent, changes apply on next contact';
    wifiPassword = '';
    wifiPassword5 = '';
    setTimeout(loadWifiParams, 8000);
  }

  async function applyWanPpp() {
    const payload = igdWanSetPayload(wanUser, wanPass);
    if (Object.keys(payload).length === 0) {
      setMsg = 'Enter username or password to update';
      return;
    }
    await api.devices.setParameters(serial, payload);
    setMsg = 'PPPoE credentials queued';
    await api.devices.wake(serial);
    wanPass = '';
    setTimeout(reloadFromDevice, 8000);
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

    <div class="flex gap-1 mb-4 border-b border-gray-800 flex-wrap">
      {#each mainTabs as t}
        <button
          type="button"
          on:click={() => (tab = t)}
          class="px-4 py-2 text-sm transition-colors
            {tab === t
              ? 'text-white border-b-2 border-blue-500'
              : 'text-gray-400 hover:text-white'}"
        >
          {t === 'lan'
            ? 'LAN clients'
            : t === 'wan'
              ? 'WAN / PPPoE'
              : t.charAt(0).toUpperCase() + t.slice(1)}
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
        <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
          <div class="flex items-center justify-between mb-3">
            <h3 class="text-xs text-gray-500 uppercase tracking-wider">
              Current WiFi ({ns === 'igd' ? 'IGD / XPON' : 'TR-181'})
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
          {#if ns === 'igd'}
            <div class="space-y-2 text-sm">
              <div class="flex justify-between">
                <span class="text-gray-400">2.4 GHz SSID</span>
                <span class="text-white font-mono">{wifiView.ssid24 || '—'}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-gray-400">2.4 GHz Channel</span>
                <span class="text-white">{wifiView.ch24 || '—'}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-gray-400">5 GHz SSID</span>
                <span class="text-white font-mono">{wifiView.ssid5 || '—'}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-gray-400">5 GHz Channel</span>
                <span class="text-white">{wifiView.ch5 || '—'}</span>
              </div>
            </div>
          {:else}
            {#if Object.keys(wifiParams).length === 0}
              <p class="text-xs text-yellow-500/80">
                No WiFi data cached — click <strong>Fetch from device</strong>.
              </p>
            {/if}
            <div class="space-y-2 text-sm">
              <div class="flex justify-between">
                <span class="text-gray-400">SSID</span>
                <span class="text-white font-mono">{wifiView.ssid24 || '—'}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-gray-400">Security</span>
                <span class="text-white">{wifiView.security || '—'}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-gray-400">Channel</span>
                <span class="text-white">{wifiView.channel || '—'}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-gray-400">Band</span>
                <span class="text-white">{wifiView.band || '—'}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-gray-400">Standard</span>
                <span class="text-white">{wifiView.standards || '—'}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-gray-400">WiFi clients</span>
                <span class="text-white">{wifiView.assocCount || '0'}</span>
              </div>
            </div>
          {/if}
        </div>

        <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
          <h3 class="text-xs text-gray-500 uppercase tracking-wider mb-3">Change WiFi</h3>
          {#if ns === 'igd'}
            <div class="space-y-3">
              <div>
                <label class="text-xs text-gray-400" for="w24">2.4 GHz SSID</label>
                <input
                  id="w24"
                  bind:value={wifiSSID}
                  class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white mt-1"
                />
              </div>
              <div>
                <label class="text-xs text-gray-400" for="p24">2.4 GHz password (optional)</label>
                <input
                  id="p24"
                  type="password"
                  bind:value={wifiPassword}
                  class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white mt-1"
                />
              </div>
              <div>
                <label class="text-xs text-gray-400" for="w5">5 GHz SSID</label>
                <input
                  id="w5"
                  bind:value={wifiSSID5}
                  class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white mt-1"
                />
              </div>
              <div>
                <label class="text-xs text-gray-400" for="p5">5 GHz password (optional)</label>
                <input
                  id="p5"
                  type="password"
                  bind:value={wifiPassword5}
                  class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white mt-1"
                />
              </div>
            </div>
          {:else}
            <div class="space-y-3">
              <div>
                <label class="text-xs text-gray-400" for="wifi-ssid">SSID</label>
                <input
                  id="wifi-ssid"
                  bind:value={wifiSSID}
                  class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white mt-1"
                />
              </div>
              <div>
                <label class="text-xs text-gray-400" for="wifi-security">Security</label>
                <select
                  id="wifi-security"
                  bind:value={wifiSecurity}
                  class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white mt-1"
                >
                  <option value="None">None</option>
                  <option value="WPA2-Personal">WPA2-Personal</option>
                  <option value="WPA-WPA2-Personal">WPA + WPA2</option>
                </select>
              </div>
              <div>
                <label class="text-xs text-gray-400" for="wifi-password">Password (optional)</label>
                <input
                  id="wifi-password"
                  type="password"
                  bind:value={wifiPassword}
                  class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white mt-1"
                />
              </div>
            </div>
          {/if}
          <button
            type="button"
            on:click={applyWiFi}
            class="w-full mt-4 bg-blue-600 hover:bg-blue-700 text-white rounded-lg py-2 text-sm font-medium"
          >
            Queue WiFi changes
          </button>
        </div>

        {#if ns !== 'igd' && wifiParams['Device.WiFi.AccessPoint.1.AssociatedDevice.1.MACAddress']}
          <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
            <h3 class="text-xs text-gray-500 uppercase tracking-wider mb-3">Associated client</h3>
            <div class="text-sm space-y-2 font-mono text-white">
              {wifiParams['Device.WiFi.AccessPoint.1.AssociatedDevice.1.MACAddress']}
            </div>
          </div>
        {/if}
      </div>

    {:else if tab === 'wan' && ns === 'igd'}
      <div class="max-w-xl space-y-4">
        <div class="bg-gray-900 border border-gray-800 rounded-xl p-4 space-y-3">
          <div class="flex items-center gap-2">
            <span class="text-xs text-gray-500 uppercase">Status</span>
            <span
              class="px-2 py-0.5 rounded text-xs font-medium
              {igdWanLooksConnected(wanPpp?.connectionStatus ?? '')
                ? 'bg-green-900/40 text-green-400'
                : 'bg-red-900/40 text-red-400'}"
            >
              {wanPpp?.connectionStatus || 'Unknown'}
            </span>
          </div>
          <div class="text-sm">
            <span class="text-gray-400">External IP</span>
            <div class="font-mono text-white mt-1">{wanPpp?.externalIP || '—'}</div>
          </div>
          <div class="text-sm">
            <span class="text-gray-400">Uptime</span>
            <div class="text-white mt-1">{formatUptimeSeconds(wanPpp?.uptimeSec ?? '')}</div>
          </div>
        </div>
        <div class="bg-gray-900 border border-gray-800 rounded-xl p-4 space-y-3">
          <h3 class="text-xs text-gray-500 uppercase">PPPoE credentials</h3>
          <div>
            <label class="text-xs text-gray-400" for="wan-u">Username</label>
            <input
              id="wan-u"
              bind:value={wanUser}
              class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white mt-1"
            />
          </div>
          <div>
            <label class="text-xs text-gray-400 flex items-center gap-2" for="wan-p">
              Password
              <button
                type="button"
                class="p-0.5 text-gray-500 hover:text-white"
                on:click={() => (showPppPassword = !showPppPassword)}
                aria-label="Toggle password visibility"
              >
                {#if showPppPassword}
                  <EyeOff size={14} />
                {:else}
                  <Eye size={14} />
                {/if}
              </button>
            </label>
            <input
              id="wan-p"
              type={showPppPassword ? 'text' : 'password'}
              bind:value={wanPass}
              placeholder="•••••••• (stored value hidden until you set new)"
              class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white mt-1"
            />
            <p class="text-xs text-gray-600 mt-1">
              Current (read-only): <span class="font-mono">{showPppPassword ? wanPpp?.password || '—' : '••••••••'}</span>
            </p>
          </div>
          <button
            type="button"
            on:click={applyWanPpp}
            class="w-full bg-blue-600 hover:bg-blue-700 text-white rounded-lg py-2 text-sm"
          >
            Queue username / password
          </button>
        </div>
      </div>

    {:else if tab === 'optics' && ns === 'igd'}
      <div class="grid grid-cols-2 gap-4 max-w-2xl">
        <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
          <div class="text-xs text-gray-500 uppercase">RX power</div>
          <div class="text-lg font-mono mt-1 {rxPowerBadgeClass(rxNum)} inline-block px-2 py-1 rounded">
            {optics?.rxDbm || '—'} <span class="text-sm">dBm</span>
          </div>
        </div>
        <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
          <div class="text-xs text-gray-500 uppercase">TX power</div>
          <div class="text-lg font-mono text-white mt-1">{optics?.txDbm || '—'} dBm</div>
        </div>
        <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
          <div class="text-xs text-gray-500 uppercase">Voltage</div>
          <div class="text-lg font-mono text-white mt-1">{optics?.voltage || '—'} V</div>
        </div>
        <div class="bg-gray-900 border border-gray-800 rounded-xl p-4">
          <div class="text-xs text-gray-500 uppercase">Temperature</div>
          <div class="text-lg font-mono text-white mt-1">{optics?.temperature || '—'} °C</div>
        </div>
      </div>

    {:else if tab === 'lan' && ns === 'igd'}
      <div class="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden max-w-5xl">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-gray-800 text-gray-400 text-xs uppercase">
              <th class="px-4 py-3 text-left">Host</th>
              <th class="px-4 py-3 text-left">IP</th>
              <th class="px-4 py-3 text-left">MAC</th>
              <th class="px-4 py-3 text-left">Interface</th>
              <th class="px-4 py-3 text-left">Active</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-800">
            {#each lanHosts as h}
              <tr>
                <td class="px-4 py-2 text-white">{h.hostName || '—'}</td>
                <td class="px-4 py-2 font-mono text-gray-300">{h.ipAddress}</td>
                <td class="px-4 py-2 font-mono text-gray-400 text-xs">{h.macAddress}</td>
                <td class="px-4 py-2 text-gray-400">{h.interfaceType || '—'}</td>
                <td class="px-4 py-2">{h.active || '—'}</td>
              </tr>
            {:else}
              <tr>
                <td colspan="5" class="px-4 py-8 text-center text-gray-500">
                  No LAN hosts in parameters — wait for sync or Fetch from WiFi tab to refresh Hosts tree.
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

    {:else if tab === 'voip' && ns === 'igd' && showVoipTab}
      <div class="max-w-lg bg-gray-900 border border-gray-800 rounded-xl p-4 space-y-3">
        <div>
          <div class="text-xs text-gray-500 uppercase">SIP auth username</div>
          <div class="text-white font-mono mt-1">{voipInfo?.sipAuthUserName || '—'}</div>
        </div>
        <div>
          <div class="text-xs text-gray-500 uppercase">Line status</div>
          <div class="text-white mt-1">{voipInfo?.lineStatus || '—'}</div>
        </div>
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
