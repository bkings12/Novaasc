package cwmp

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

func wrapBody(body []byte) []byte {
	return bytes.Join([][]byte{
		[]byte(`<r xmlns:cwmp="` + CWMPNS + `">`),
		body,
		[]byte(`</r>`),
	}, nil)
}

// ParseGetParameterValuesResponse parses SOAP body into GetParameterValuesResponse.
func ParseGetParameterValuesResponse(body []byte) (*GetParameterValuesResponse, error) {
	wrapped := wrapBody(body)
	var wrapper struct {
		GetParameterValuesResponse GetParameterValuesResponse `xml:"GetParameterValuesResponse"`
	}
	dec := xml.NewDecoder(bytes.NewReader(wrapped))
	dec.DefaultSpace = CWMPNS
	if err := dec.Decode(&wrapper); err != nil {
		return nil, err
	}
	return &wrapper.GetParameterValuesResponse, nil
}

// ParseSetParameterValuesResponse parses SOAP body into SetParameterValuesResponse.
func ParseSetParameterValuesResponse(body []byte) (*SetParameterValuesResponse, error) {
	wrapped := wrapBody(body)
	var wrapper struct {
		SetParameterValuesResponse SetParameterValuesResponse `xml:"SetParameterValuesResponse"`
	}
	dec := xml.NewDecoder(bytes.NewReader(wrapped))
	dec.DefaultSpace = CWMPNS
	if err := dec.Decode(&wrapper); err != nil {
		return nil, err
	}
	return &wrapper.SetParameterValuesResponse, nil
}

// ParseGetParameterNamesResponse parses SOAP body into GetParameterNamesResponse.
func ParseGetParameterNamesResponse(body []byte) (*GetParameterNamesResponse, error) {
	wrapped := wrapBody(body)
	var wrapper struct {
		GetParameterNamesResponse GetParameterNamesResponse `xml:"GetParameterNamesResponse"`
	}
	dec := xml.NewDecoder(bytes.NewReader(wrapped))
	dec.DefaultSpace = CWMPNS
	if err := dec.Decode(&wrapper); err != nil {
		return nil, err
	}
	return &wrapper.GetParameterNamesResponse, nil
}

// ParseTransferComplete parses SOAP body into TransferComplete.
func ParseTransferComplete(body []byte) (*TransferComplete, error) {
	wrapped := wrapBody(body)
	var wrapper struct {
		TransferComplete TransferComplete `xml:"TransferComplete"`
	}
	dec := xml.NewDecoder(bytes.NewReader(wrapped))
	dec.DefaultSpace = CWMPNS
	if err := dec.Decode(&wrapper); err != nil {
		return nil, err
	}
	return &wrapper.TransferComplete, nil
}

// FaultParse holds parsed fault for handler use.
type FaultParse struct {
	Detail struct {
		FaultCode   string `xml:"FaultCode"`
		FaultString string `xml:"FaultString"`
	}
}

// ParseFault parses SOAP Fault detail (CWMP fault). Body is the raw SOAP body.
func ParseFault(body []byte) (*FaultParse, error) {
	var fault Fault
	if err := xml.Unmarshal(body, &fault); err != nil {
		return nil, err
	}
	f := &FaultParse{}
	if fault.Detail != nil && fault.Detail.CWMPFault != nil {
		f.Detail.FaultCode = fmt.Sprintf("%d", fault.Detail.CWMPFault.FaultCode)
		f.Detail.FaultString = fault.Detail.CWMPFault.FaultString
	}
	return f, nil
}
