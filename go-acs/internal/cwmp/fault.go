package cwmp

// FaultCode names for logging.
const (
	FaultCodeMethodNotSupported   = 8000
	FaultCodeInvalidParameterName = 8005
	FaultCodeRequestDenied        = 9001
	FaultCodeInternalError        = 9002
)

// FaultString returns a human-readable string for a fault code.
func FaultString(code int) string {
	switch code {
	case FaultCodeMethodNotSupported:
		return "Method not supported"
	case FaultCodeInvalidParameterName:
		return "Invalid parameter name"
	case FaultCodeRequestDenied:
		return "Request denied"
	case FaultCodeInternalError:
		return "Internal error"
	default:
		return "Unknown fault"
	}
}
