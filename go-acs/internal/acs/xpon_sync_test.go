package acs

import "testing"

func TestParameterSyncPathsForInform_IGD(t *testing.T) {
	p := ParameterSyncPathsForInform(map[string]string{
		"InternetGatewayDevice.DeviceInfo.SoftwareVersion": "1",
	})
	if len(p) < 3 {
		t.Fatalf("expected IGD paths, got %d entries", len(p))
	}
	if p[0] != "InternetGatewayDevice.DeviceInfo." {
		t.Fatalf("first path: %q", p[0])
	}
}

func TestParameterSyncPathsForInform_TR181(t *testing.T) {
	p := ParameterSyncPathsForInform(map[string]string{
		"Device.DeviceInfo.SoftwareVersion": "1",
	})
	if len(p) < 2 {
		t.Fatalf("expected TR-181 paths, got %d entries", len(p))
	}
	if p[0] != "Device.DeviceInfo." {
		t.Fatalf("first path: %q", p[0])
	}
}
