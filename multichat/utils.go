package multichat

import "strings"

// ClassifyProviderError returns a string representing the error type for provider errors.
func ClassifyProviderError(err error) string {
	if err == nil {
		return ""
	}
	errMsg := err.Error()
	switch {
	case strings.Contains(errMsg, "failed to send request"):
		return "Network"
	case strings.Contains(errMsg, "API returned status"):
		return "API"
	case strings.Contains(errMsg, "failed to parse response"):
		return "Parse"
	case strings.Contains(errMsg, "API error"):
		return "Service"
	default:
		return "Unknown"
	}
}
