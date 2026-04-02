package acs

import "strings"

// InformParameterSyncCreatedBy marks auto-enqueued GetParameterValues from Inform handling.
const InformParameterSyncCreatedBy = "inform:xpon-parameter-sync"

// igdInformSyncPaths are TR-098 (InternetGatewayDevice) subtree roots for Huawei/ZTE-style XPON CPEs.
// Trailing dots request all parameters under that object (TR-069 GetParameterValues).
func igdInformSyncPaths() []string {
	return []string{
		"InternetGatewayDevice.DeviceInfo.",
		"InternetGatewayDevice.ManagementServer.",
		"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.",
		"InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.",
		"InternetGatewayDevice.LANDevice.1.Hosts.",
		"InternetGatewayDevice.LANDevice.1.LANEthernetInterfaceConfig.",
		// WANConnectionDevice.* (PPPoE/IP indices vary by firmware)
		"InternetGatewayDevice.WANDevice.1.",
		// Huawei GPON optics (vendor object; typo + corrected spelling — CPE may only support one)
		"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.",
		"InternetGatewayDevice.WANDevice.1.X_GponInterfaceConfig.",
		"InternetGatewayDevice.Services.VoiceService.1.",
	}
}

// tr181InformSyncPaths are Device:2 style roots for TR-181 CPEs (e.g. some routers).
func tr181InformSyncPaths() []string {
	return []string{
		"Device.DeviceInfo.",
		"Device.ManagementServer.",
		"Device.WiFi.",
		"Device.Hosts.Host.",
		"Device.Ethernet.",
		"Device.IP.",
		"Device.Optical.",
	}
}

// ParameterSyncPathsForInform chooses IGD vs TR-181 paths from keys present in the Inform snapshot.
func ParameterSyncPathsForInform(informParams map[string]string) []string {
	for k := range informParams {
		if strings.HasPrefix(k, "InternetGatewayDevice.") {
			return igdInformSyncPaths()
		}
	}
	for k := range informParams {
		if strings.HasPrefix(k, "Device.") {
			return tr181InformSyncPaths()
		}
	}
	return igdInformSyncPaths()
}
