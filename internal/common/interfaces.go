package common

// StatusCallback interface for UI handlers to send status updates
type StatusCallback interface {
	UpdateServiceStatusMessage(serviceName, message string)
}
