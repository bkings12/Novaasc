/** TR-069 data model hint from stored parameters */
export type ParamNamespace = 'tr181' | 'igd' | 'unknown';

export function detectNamespace(
  params: Record<string, string> | null | undefined
): ParamNamespace {
  if (!params) return 'unknown';
  for (const k of Object.keys(params)) {
    if (k.startsWith('InternetGatewayDevice.')) return 'igd';
  }
  for (const k of Object.keys(params)) {
    if (k.startsWith('Device.')) return 'tr181';
  }
  return 'unknown';
}

/** Pick TR-181 vs IGD path based on namespace. */
export function getParamValue(
  params: Record<string, string>,
  ns: ParamNamespace,
  tr181Path: string,
  igdPath: string
): string {
  if (ns === 'igd') return params[igdPath] ?? '';
  if (ns === 'tr181') return params[tr181Path] ?? '';
  return params[igdPath] ?? params[tr181Path] ?? '';
}

const TR_WIFI = {
  ssid: 'Device.WiFi.SSID.1.SSID',
  passphrase: 'Device.WiFi.AccessPoint.1.Security.KeyPassphrase',
  security: 'Device.WiFi.AccessPoint.1.Security.ModeEnabled',
  channel: 'Device.WiFi.Radio.1.Channel',
  band: 'Device.WiFi.Radio.1.OperatingFrequencyBand',
  standards: 'Device.WiFi.Radio.1.OperatingStandards',
  assocCount: 'Device.WiFi.AccessPoint.1.AssociatedDeviceNumberOfEntries'
};

const IGD_WIFI = {
  ssid24: 'InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID',
  pass24: 'InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase',
  ch24: 'InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Channel',
  ssid5: 'InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID',
  pass5: 'InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase',
  ch5: 'InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.Channel'
};

export function getWiFiDisplay(
  params: Record<string, string>,
  ns: ParamNamespace
): {
  ssid24: string;
  ssid5: string;
  pass24: string;
  pass5: string;
  ch24: string;
  ch5: string;
  security: string;
  channel: string;
  band: string;
  standards: string;
  assocCount: string;
} {
  if (ns === 'igd') {
    return {
      ssid24: params[IGD_WIFI.ssid24] ?? '',
      ssid5: params[IGD_WIFI.ssid5] ?? '',
      pass24: params[IGD_WIFI.pass24] ?? '',
      pass5: params[IGD_WIFI.pass5] ?? '',
      ch24: params[IGD_WIFI.ch24] ?? '',
      ch5: params[IGD_WIFI.ch5] ?? '',
      security: '',
      channel: params[IGD_WIFI.ch24] ?? '',
      band: '',
      standards: '',
      assocCount: ''
    };
  }
  return {
    ssid24: params[TR_WIFI.ssid] ?? '',
    ssid5: '',
    pass24: params[TR_WIFI.passphrase] ?? '',
    pass5: '',
    ch24: params[TR_WIFI.channel] ?? '',
    ch5: '',
    security: params[TR_WIFI.security] ?? '',
    channel: params[TR_WIFI.channel] ?? '',
    band: params[TR_WIFI.band] ?? '',
    standards: params[TR_WIFI.standards] ?? '',
    assocCount: params[TR_WIFI.assocCount] ?? ''
  };
}

export function igdWifiSetPayload(values: {
  ssid24?: string;
  ssid5?: string;
  pass24?: string;
  pass5?: string;
}): Record<string, string> {
  const out: Record<string, string> = {};
  if (values.ssid24?.trim())
    out[IGD_WIFI.ssid24] = values.ssid24.trim();
  if (values.ssid5?.trim()) out[IGD_WIFI.ssid5] = values.ssid5.trim();
  if (values.pass24) out[IGD_WIFI.pass24] = values.pass24;
  if (values.pass5) out[IGD_WIFI.pass5] = values.pass5;
  return out;
}

export function tr181WifiSetPayload(values: {
  ssid?: string;
  passphrase?: string;
  security?: string;
}): Record<string, string> {
  const out: Record<string, string> = {};
  if (values.ssid?.trim()) out[TR_WIFI.ssid] = values.ssid.trim();
  if (values.passphrase) out[TR_WIFI.passphrase] = values.passphrase;
  if (values.security) out[TR_WIFI.security] = values.security;
  return out;
}

export function igdWifiGetParameterNames(): string[] {
  return Object.values(IGD_WIFI);
}

export interface LanHostRow {
  index: string;
  macAddress: string;
  ipAddress: string;
  hostName: string;
  interfaceType: string;
  active: string;
}

/** Parse TR-098 Host.{i}.* from flat parameters map */
export function parseIgdLanHosts(params: Record<string, string>): LanHostRow[] {
  const re =
    /^InternetGatewayDevice\.LANDevice\.1\.Hosts\.Host\.(\d+)\.(MACAddress|IPAddress|HostName|InterfaceType|Active)$/;
  const map = new Map<
    string,
    Partial<LanHostRow> & { index: string }
  >();
  for (const [k, v] of Object.entries(params)) {
    const m = k.match(re);
    if (!m) continue;
    const idx = m[1];
    const field = m[2];
    if (!map.has(idx)) map.set(idx, { index: idx });
    const row = map.get(idx)!;
    if (field === 'MACAddress') row.macAddress = v;
    else if (field === 'IPAddress') row.ipAddress = v;
    else if (field === 'HostName') row.hostName = v;
    else if (field === 'InterfaceType') row.interfaceType = v;
    else if (field === 'Active') row.active = v;
  }
  return Array.from(map.values())
    .filter((r) => (r.macAddress || r.ipAddress) && r.index)
    .map((r) => ({
      index: r.index,
      macAddress: r.macAddress ?? '',
      ipAddress: r.ipAddress ?? '',
      hostName: r.hostName ?? '',
      interfaceType: r.interfaceType ?? '',
      active: r.active ?? ''
    }))
    .sort((a, b) => Number(a.index) - Number(b.index));
}

export function deviceSummaryHasVoiceService(
  params: Record<string, string>
): boolean {
  const s =
    params['InternetGatewayDevice.DeviceSummary'] ??
    params['Device.DeviceSummary'] ??
    '';
  return /VoiceService/i.test(s);
}

const IGD_PPP = {
  user: 'InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Username',
  pass: 'InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Password',
  status:
    'InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.ConnectionStatus',
  extIpPPP:
    'InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.ExternalIPAddress',
  uptime:
    'InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Uptime',
  extIpWAN:
    'InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.ExternalIPAddress'
};

export function getIgdWanPpp(params: Record<string, string>) {
  return {
    username: params[IGD_PPP.user] ?? '',
    password: params[IGD_PPP.pass] ?? '',
    connectionStatus: params[IGD_PPP.status] ?? '',
    externalIP:
      params[IGD_PPP.extIpPPP] ||
      params[IGD_PPP.extIpWAN] ||
      '',
    uptimeSec: params[IGD_PPP.uptime] ?? ''
  };
}

export function igdWanSetPayload(username: string, password: string): Record<string, string> {
  const o: Record<string, string> = {};
  if (username.trim()) o[IGD_PPP.user] = username.trim();
  if (password) o[IGD_PPP.pass] = password;
  return o;
}

const IGD_GPON_TYPO =
  'InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.';
const IGD_GPON_OK =
  'InternetGatewayDevice.WANDevice.1.X_GponInterfaceConfig.';

export function getIgdOptics(params: Record<string, string>) {
  const pick = (suffix: string) =>
    params[IGD_GPON_TYPO + suffix] ?? params[IGD_GPON_OK + suffix] ?? '';
  return {
    rxDbm: pick('RXPower'),
    txDbm: pick('TXPower'),
    voltage: pick('Voltage'),
    temperature: pick('Temperature')
  };
}

export function parseRxDbm(s: string): number | null {
  const n = parseFloat(String(s).replace(/[^\d.-]/g, ''));
  return Number.isFinite(n) ? n : null;
}

export function rxPowerBadgeClass(dbm: number | null): string {
  if (dbm === null) return 'bg-gray-700 text-gray-300';
  // Typical GPON RX: about -8 dBm (strong) to -27 dBm (weak but OK)
  if (dbm >= -27 && dbm <= -8)
    return 'bg-green-900/40 text-green-400 border border-green-700';
  if (dbm < -35 || dbm > -5)
    return 'bg-red-900/40 text-red-400 border border-red-700';
  return 'bg-yellow-900/40 text-yellow-400 border border-yellow-700';
}

export function formatUptimeSeconds(raw: string): string {
  const sec = parseInt(raw, 10);
  if (!Number.isFinite(sec) || sec < 0) return raw || '—';
  const d = Math.floor(sec / 86400);
  const h = Math.floor((sec % 86400) / 3600);
  const m = Math.floor((sec % 3600) / 60);
  const parts: string[] = [];
  if (d) parts.push(`${d}d`);
  if (h || d) parts.push(`${h}h`);
  parts.push(`${m}m`);
  return parts.join(' ');
}

const VOIP = {
  sipUser:
    'InternetGatewayDevice.Services.VoiceService.1.VoiceProfile.1.Line.1.SIP.AuthUserName',
  lineStatus:
    'InternetGatewayDevice.Services.VoiceService.1.VoiceProfile.1.Line.1.Status'
};

export function getIgdVoip(params: Record<string, string>) {
  return {
    sipAuthUserName: params[VOIP.sipUser] ?? '',
    lineStatus: params[VOIP.lineStatus] ?? ''
  };
}
