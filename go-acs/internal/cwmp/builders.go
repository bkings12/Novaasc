package cwmp

import (
	"fmt"
	"strings"

	"github.com/novaacs/go-acs/internal/task"
)

const cwmpNS = `xmlns:cwmp="urn:dslforum-org:cwmp-1-0"`

// BuildGetParameterValues returns full SOAP envelope for GetParameterValues RPC.
func BuildGetParameterValues(id string, names []string) ([]byte, error) {
	var sb strings.Builder
	for _, n := range names {
		sb.WriteString("<string>")
		sb.WriteString(EscapeXML(n))
		sb.WriteString("</string>")
	}
	inner := fmt.Sprintf(`<cwmp:GetParameterValues %s><ParameterNames>%s</ParameterNames></cwmp:GetParameterValues>`, cwmpNS, sb.String())
	return BuildEnvelope([]byte(inner), id)
}

// BuildSetParameterValues returns full SOAP envelope for SetParameterValues RPC.
func BuildSetParameterValues(id string, params map[string]string) ([]byte, error) {
	var sb strings.Builder
	for name, val := range params {
		sb.WriteString("<ParameterValueStruct><Name>")
		sb.WriteString(EscapeXML(name))
		sb.WriteString("</Name><Value xsi:type=\"xsd:string\">")
		sb.WriteString(EscapeXML(val))
		sb.WriteString("</Value></ParameterValueStruct>")
	}
	inner := fmt.Sprintf(`<cwmp:SetParameterValues %s xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema"><ParameterList>%s</ParameterList><ParameterKey></ParameterKey></cwmp:SetParameterValues>`, cwmpNS, sb.String())
	return BuildEnvelope([]byte(inner), id)
}

// BuildGetParameterNames returns full SOAP envelope for GetParameterNames RPC.
func BuildGetParameterNames(id string, path string, nextLevel bool) ([]byte, error) {
	next := "0"
	if nextLevel {
		next = "1"
	}
	inner := fmt.Sprintf(`<cwmp:GetParameterNames %s><ParameterPath>%s</ParameterPath><NextLevel>%s</NextLevel></cwmp:GetParameterNames>`, cwmpNS, EscapeXML(path), next)
	return BuildEnvelope([]byte(inner), id)
}

// BuildReboot returns full SOAP envelope for Reboot RPC.
func BuildReboot(id string, commandKey string) ([]byte, error) {
	inner := fmt.Sprintf(`<cwmp:Reboot %s><CommandKey>%s</CommandKey></cwmp:Reboot>`, cwmpNS, EscapeXML(commandKey))
	return BuildEnvelope([]byte(inner), id)
}

// BuildFactoryReset returns full SOAP envelope for FactoryReset RPC.
func BuildFactoryReset(id string) ([]byte, error) {
	inner := fmt.Sprintf(`<cwmp:FactoryReset %s><CommandKey></CommandKey></cwmp:FactoryReset>`, cwmpNS)
	return BuildEnvelope([]byte(inner), id)
}

// BuildDownload returns full SOAP envelope for Download RPC.
func BuildDownload(id string, args *task.DownloadArgs) ([]byte, error) {
	if args == nil {
		inner := fmt.Sprintf(`<cwmp:Download %s><CommandKey></CommandKey><FileType>1 Firmware Upgrade Image</FileType><URL></URL><Username></Username><Password></Password><FileSize>0</FileSize><TargetFileName></TargetFileName><DelaySeconds>0</DelaySeconds></cwmp:Download>`, cwmpNS)
		return BuildEnvelope([]byte(inner), id)
	}
	fileType := args.FileType
	if fileType == "" {
		fileType = "1 Firmware Upgrade Image"
	}
	inner := fmt.Sprintf(`<cwmp:Download %s><CommandKey>%s</CommandKey><FileType>%s</FileType><URL>%s</URL><Username>%s</Username><Password>%s</Password><FileSize>%d</FileSize><TargetFileName>%s</TargetFileName><DelaySeconds>%d</DelaySeconds></cwmp:Download>`,
		cwmpNS,
		EscapeXML(args.CommandKey),
		EscapeXML(fileType),
		EscapeXML(args.URL),
		EscapeXML(args.Username),
		EscapeXML(args.Password),
		args.FileSize,
		EscapeXML(args.TargetFile),
		args.DelaySeconds,
	)
	return BuildEnvelope([]byte(inner), id)
}

// BuildTransferCompleteResponse returns SOAP envelope for TransferCompleteResponse (empty body).
func BuildTransferCompleteResponse(id string) ([]byte, error) {
	inner := fmt.Sprintf(`<cwmp:TransferCompleteResponse %s></cwmp:TransferCompleteResponse>`, cwmpNS)
	return BuildEnvelope([]byte(inner), id)
}
