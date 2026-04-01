package cwmp

import "bytes"

// DetectMessageType returns the CWMP method name from the SOAP body.
func DetectMessageType(body []byte) string {
	checks := []struct {
		token  []byte
		result string
	}{
		{[]byte("cwmp:Inform"), "Inform"},
		{[]byte(":Inform>"), "Inform"},
		{[]byte("cwmp:TransferComplete"), "TransferComplete"},
		{[]byte("cwmp:GetParameterValuesResponse"), "GetParameterValuesResponse"},
		{[]byte("cwmp:SetParameterValuesResponse"), "SetParameterValuesResponse"},
		{[]byte("cwmp:GetParameterNamesResponse"), "GetParameterNamesResponse"},
		{[]byte("cwmp:RebootResponse"), "RebootResponse"},
		{[]byte("cwmp:FactoryResetResponse"), "FactoryResetResponse"},
		{[]byte("cwmp:DownloadResponse"), "DownloadResponse"},
		{[]byte("cwmp:Fault"), "Fault"},
		{[]byte(":Fault>"), "Fault"},
	}
	for _, ch := range checks {
		if bytes.Contains(body, ch.token) {
			return ch.result
		}
	}
	return "Unknown"
}
