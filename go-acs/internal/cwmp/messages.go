package cwmp

import (
	"encoding/xml"
	"time"
)

// ---- Inform (from CPE) ----

// Inform holds DeviceId and Event/ParameterList from CPE.
type Inform struct {
	XMLName       xml.Name      `xml:"urn:dslforum-org:cwmp-1-0 Inform"`
	DeviceID      DeviceID      `xml:"DeviceId"`
	Event         EventList     `xml:"Event"`
	MaxEnvelopes  int           `xml:"MaxEnvelopes"`
	CurrentTime   time.Time     `xml:"CurrentTime"`
	RetryCount    int           `xml:"RetryCount"`
	ParameterList ParameterList `xml:"ParameterList"`
}

// DeviceID from Inform (TR-069).
type DeviceID struct {
	Manufacturer string `xml:"Manufacturer"`
	OUI          string `xml:"OUI"`
	ProductClass string `xml:"ProductClass"`
	SerialNumber string `xml:"SerialNumber"`
}

// EventList holds event codes (e.g. "0 BOOTSTRAP", "1 BOOT").
type EventList struct {
	Events []string `xml:"Event"`
}

// ParameterValueStruct is a single name/value from CPE.
type ParameterValueStruct struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
}

// ParameterList is used in Inform and GetParameterValuesResponse.
type ParameterList struct {
	Params []ParameterValueStruct `xml:"ParameterValueStruct"`
}

// GetParam returns a parameter value by name from the Inform ParameterList.
func (inf *Inform) GetParam(name string) string {
	for _, p := range inf.ParameterList.Params {
		if p.Name == name {
			return p.Value
		}
	}
	return ""
}

// EventCodes returns the event code strings from the last Inform (e.g. ["0 BOOTSTRAP", "1 BOOT"]).
func (inf *Inform) EventCodes() []string {
	if inf.Event.Events == nil {
		return nil
	}
	return inf.Event.Events
}

// HasEvent returns true if the given event code is present in this Inform.
func (inf *Inform) HasEvent(code string) bool {
	for _, e := range inf.Event.Events {
		if e == code {
			return true
		}
	}
	return false
}

// ---- InformResponse (to CPE) ----

// InformResponse sent by ACS after Inform.
type InformResponse struct {
	XMLName      xml.Name `xml:"urn:dslforum-org:cwmp-1-0 InformResponse"`
	MaxEnvelopes int      `xml:"MaxEnvelopes"`
}

// ---- GetParameterValues (to CPE) ----

// GetParameterValues request.
type GetParameterValues struct {
	XMLName        xml.Name       `xml:"urn:dslforum-org:cwmp-1-0 GetParameterValues"`
	ParameterNames ParameterNames `xml:"ParameterNames"`
}

// ParameterNames holds string array.
type ParameterNames struct {
	Names []string `xml:"string"`
}

// ---- GetParameterValuesResponse (from CPE) ----

// GetParameterValuesResponse from CPE.
type GetParameterValuesResponse struct {
	XMLName       xml.Name      `xml:"urn:dslforum-org:cwmp-1-0 GetParameterValuesResponse"`
	ParameterList ParameterList `xml:"ParameterList"`
}

// ---- SetParameterValues (to CPE) ----

// SetParameterValues request.
type SetParameterValues struct {
	XMLName       xml.Name      `xml:"urn:dslforum-org:cwmp-1-0 SetParameterValues"`
	ParameterList ParameterList `xml:"ParameterList"`
	ParameterKey  string        `xml:"ParameterKey,omitempty"`
}

// ---- SetParameterValuesResponse (from CPE) ----

// SetParameterValuesResponse from CPE.
type SetParameterValuesResponse struct {
	XMLName xml.Name `xml:"urn:dslforum-org:cwmp-1-0 SetParameterValuesResponse"`
	Status  int      `xml:"Status"`
}

// ---- GetParameterNames (to CPE) ----

// GetParameterNames request.
type GetParameterNames struct {
	XMLName       xml.Name `xml:"urn:dslforum-org:cwmp-1-0 GetParameterNames"`
	ParameterPath string   `xml:"ParameterPath"`
	NextLevel     bool     `xml:"NextLevel"`
}

// ---- GetParameterNamesResponse (from CPE) ----

// ParameterInfoStruct for GetParameterNamesResponse.
type ParameterInfoStruct struct {
	Name     string `xml:"Name"`
	Writable bool   `xml:"Writable"`
}

// GetParameterNamesResponse from CPE.
type GetParameterNamesResponse struct {
	XMLName xml.Name              `xml:"urn:dslforum-org:cwmp-1-0 GetParameterNamesResponse"`
	Params  []ParameterInfoStruct `xml:"ParameterList>ParameterInfoStruct"`
}

// ---- Download (to CPE) ----

// Download request (firmware).
type Download struct {
	XMLName        xml.Name `xml:"urn:dslforum-org:cwmp-1-0 Download"`
	CommandKey     string   `xml:"CommandKey"`
	FileType       string   `xml:"FileType"`
	URL            string   `xml:"URL"`
	Username       string   `xml:"Username,omitempty"`
	Password       string   `xml:"Password,omitempty"`
	FileSize       int      `xml:"FileSize,omitempty"`
	TargetFileName string   `xml:"TargetFileName,omitempty"`
	DelaySeconds   int      `xml:"DelaySeconds,omitempty"`
	SuccessURL     string   `xml:"SuccessURL,omitempty"`
	FailureURL     string   `xml:"FailureURL,omitempty"`
}

// ---- DownloadResponse (from CPE) ----

// DownloadResponse from CPE.
type DownloadResponse struct {
	XMLName      xml.Name `xml:"urn:dslforum-org:cwmp-1-0 DownloadResponse"`
	Status       int      `xml:"Status"`
	StartTime    string   `xml:"StartTime,omitempty"`
	CompleteTime string   `xml:"CompleteTime,omitempty"`
}

// ---- Reboot (to CPE) ----

// Reboot request.
type Reboot struct {
	XMLName    xml.Name `xml:"urn:dslforum-org:cwmp-1-0 Reboot"`
	CommandKey string   `xml:"CommandKey"`
}

// ---- RebootResponse (from CPE) ----

// RebootResponse from CPE.
type RebootResponse struct {
	XMLName xml.Name `xml:"urn:dslforum-org:cwmp-1-0 RebootResponse"`
}

// ---- FactoryReset (to CPE) ----

// FactoryReset request.
type FactoryReset struct {
	XMLName    xml.Name `xml:"urn:dslforum-org:cwmp-1-0 FactoryReset"`
	CommandKey string   `xml:"CommandKey"`
}

// ---- AddObject / DeleteObject ----

// AddObject request.
type AddObject struct {
	XMLName      xml.Name `xml:"urn:dslforum-org:cwmp-1-0 AddObject"`
	ObjectName   string   `xml:"ObjectName"`
	ParameterKey string   `xml:"ParameterKey,omitempty"`
}

// AddObjectResponse from CPE.
type AddObjectResponse struct {
	XMLName        xml.Name `xml:"urn:dslforum-org:cwmp-1-0 AddObjectResponse"`
	InstanceNumber string   `xml:"InstanceNumber"`
	ParameterKey   string   `xml:"ParameterKey,omitempty"`
}

// DeleteObject request.
type DeleteObject struct {
	XMLName      xml.Name `xml:"urn:dslforum-org:cwmp-1-0 DeleteObject"`
	ObjectName   string   `xml:"ObjectName"`
	ParameterKey string   `xml:"ParameterKey,omitempty"`
}

// DeleteObjectResponse from CPE.
type DeleteObjectResponse struct {
	XMLName xml.Name `xml:"urn:dslforum-org:cwmp-1-0 DeleteObjectResponse"`
	Status  int      `xml:"Status"`
}

// ---- TransferComplete / AutonomousTransferComplete (from CPE) ----

// TransferComplete from CPE after download.
type TransferComplete struct {
	XMLName      xml.Name     `xml:"urn:dslforum-org:cwmp-1-0 TransferComplete"`
	CommandKey   string       `xml:"CommandKey"`
	FaultStruct  *FaultStruct `xml:"FaultStruct,omitempty"`
	StartTime    string       `xml:"StartTime,omitempty"`
	CompleteTime string       `xml:"CompleteTime,omitempty"`
}

// AutonomousTransferComplete from CPE.
type AutonomousTransferComplete struct {
	XMLName        xml.Name     `xml:"urn:dslforum-org:cwmp-1-0 AutonomousTransferComplete"`
	AnnounceURL    string       `xml:"AnnounceURL"`
	TransferURL    string       `xml:"TransferURL"`
	IsDownload     bool         `xml:"IsDownload"`
	FileType       string       `xml:"FileType"`
	FileSize       int          `xml:"FileSize"`
	TargetFileName string       `xml:"TargetFileName"`
	FaultStruct    *FaultStruct `xml:"FaultStruct,omitempty"`
}

// ---- Fault (from CPE) ----

// Fault in SOAP body (CWMP fault).
type Fault struct {
	XMLName     xml.Name     `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault"`
	FaultCode   string       `xml:"faultcode"`
	FaultString string       `xml:"faultstring"`
	Detail      *FaultDetail `xml:"detail,omitempty"`
}

// FaultDetail holds cwmp:Fault.
type FaultDetail struct {
	CWMPFault *CWMPFault `xml:"Fault"`
}

// CWMPFault (TR-069 fault codes).
type CWMPFault struct {
	FaultCode   int    `xml:"FaultCode"`
	FaultString string `xml:"FaultString"`
}

// FaultStruct in RPC responses (e.g. TransferComplete).
type FaultStruct struct {
	FaultCode   int    `xml:"FaultCode"`
	FaultString string `xml:"FaultString"`
}

// Fault codes (common).
const (
	FaultMethodNotSupported   = 8000
	FaultInvalidParameterName = 8005
	FaultRequestDenied        = 9001
)
