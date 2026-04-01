package cwmp

import (
	"encoding/xml"
	"strings"
)

// EscapeXML escapes text for use in XML content (e.g. cwmp:ID).
func EscapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// SOAP envelope and CWMP namespaces (TR-069).
const (
	SOAPEnvNS   = "http://schemas.xmlsoap.org/soap/envelope/"
	CWMPNS      = "urn:dslforum-org:cwmp-1-0"
	SOAPEnvPref = "soap"
	CWPMPref    = "cwmp"
)

// Envelope represents a SOAP 1.1 envelope.
type Envelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	NS      string   `xml:"xmlns:soap,attr"`
	NS1     string   `xml:"xmlns:cwmp,attr"`
	Header  *Header  `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header,omitempty"`
	Body    Body     `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
}

// Header holds SOAP header (e.g. CWMP ID).
type Header struct {
	ID string `xml:"urn:dslforum-org:cwmp-1-0 ID"`
}

// Body holds the CWMP RPC (single child).
type Body struct {
	Content []byte `xml:",innerxml"`
}

// UnmarshalEnvelope parses SOAP envelope and returns raw body content and CWMP ID.
func UnmarshalEnvelope(data []byte) (body []byte, cwmpID string, err error) {
	var env struct {
		XMLName xml.Name `xml:"Envelope"`
		Header  *struct {
			ID string `xml:"ID"`
		} `xml:"Header>ID"`
		Body struct {
			Content []byte `xml:",innerxml"`
		} `xml:"Body"`
	}
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	dec.DefaultSpace = SOAPEnvNS
	if err = dec.Decode(&env); err != nil {
		return nil, "", err
	}
	if env.Header != nil {
		cwmpID = env.Header.ID
	}
	return bytesTrim(env.Body.Content), cwmpID, nil
}

func bytesTrim(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

// BuildEnvelope wraps body XML in a SOAP 1.1 envelope with soap:/cwmp: prefixes.
// Many CPEs (e.g. MikroTik) require this exact form; Go's Marshal often uses default NS.
func BuildEnvelope(body []byte, cwmpID string) ([]byte, error) {
	id := EscapeXML(cwmpID)
	const tpl = `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
		`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:cwmp="urn:dslforum-org:cwmp-1-0">` + "\n" +
		`<soap:Header><cwmp:ID soap:mustUnderstand="1">%s</cwmp:ID></soap:Header>` + "\n" +
		`<soap:Body>%s</soap:Body>` + "\n" +
		`</soap:Envelope>`
	out := strings.Replace(tpl, "%s", id, 1)
	out = strings.Replace(out, "%s", string(body), 1)
	return []byte(out), nil
}

// BuildEnvelopeWithoutHeader builds envelope with no header (body only).
func BuildEnvelopeWithoutHeader(body []byte) ([]byte, error) {
	const tpl = `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
		`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:cwmp="urn:dslforum-org:cwmp-1-0">` + "\n" +
		`<soap:Body>%s</soap:Body>` + "\n" +
		`</soap:Envelope>`
	return []byte(strings.Replace(tpl, "%s", string(body), 1)), nil
}
