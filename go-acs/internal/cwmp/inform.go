package cwmp

import (
	"bytes"
	"encoding/xml"
)

// ParseInform deserializes Inform from SOAP body (raw XML).
// CPE often sends <cwmp:Inform>; the Body innerXML may not include xmlns:cwmp,
// so we wrap the fragment in a root that declares the CWMP namespace so the decoder
// resolves the prefix correctly.
func ParseInform(body []byte) (*Inform, error) {
	wrapped := bytes.Join([][]byte{
		[]byte(`<r xmlns:cwmp="` + CWMPNS + `">`),
		body,
		[]byte(`</r>`),
	}, nil)
	var wrapper struct {
		Inform Inform `xml:"Inform"`
	}
	dec := xml.NewDecoder(bytes.NewReader(wrapped))
	dec.DefaultSpace = CWMPNS
	if err := dec.Decode(&wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Inform, nil
}

// BuildInformResponseBody returns XML body for InformResponse (MaxEnvelopes=1).
// Use explicit cwmp: prefix so CPEs (e.g. MikroTik) accept the response.
func BuildInformResponseBody() []byte {
	return []byte(`<cwmp:InformResponse xmlns:cwmp="urn:dslforum-org:cwmp-1-0"><MaxEnvelopes>1</MaxEnvelopes></cwmp:InformResponse>`)
}
