package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Pre-compiled regex patterns for validation
var (
	ipv4Regex     = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	hostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-\.]*[a-zA-Z0-9])?$`)
	hasLetterRe   = regexp.MustCompile(`[a-zA-Z]`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// SupportedServerTypes returns the list of supported license server types
func SupportedServerTypes() map[string]bool {
	return map[string]bool{
		"flexlm": true,
		"rlm":    true,
	}
}

// ValidateHostname validates the license server hostname format (port@host or host:port)
// Returns the normalized hostname and any validation error
func ValidateHostname(hostname string) (string, error) {
	if hostname == "" {
		return "", fmt.Errorf("hostname is required")
	}

	hostname = strings.TrimSpace(hostname)

	// Check for port@host format (FlexLM style)
	if strings.Contains(hostname, "@") {
		parts := strings.SplitN(hostname, "@", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid hostname format: expected port@host")
		}

		portStr := parts[0]
		host := parts[1]

		port, err := strconv.Atoi(portStr)
		if err != nil {
			return "", fmt.Errorf("invalid port number: %s", portStr)
		}

		if port < 1 || port > 65535 {
			return "", fmt.Errorf("port must be between 1 and 65535")
		}

		if host == "" {
			return "", fmt.Errorf("host cannot be empty")
		}

		if !isValidHost(host) {
			return "", fmt.Errorf("invalid host: %s", host)
		}

		return hostname, nil
	}

	// Check for host:port format
	if strings.Contains(hostname, ":") {
		parts := strings.SplitN(hostname, ":", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid hostname format: expected host:port")
		}

		host := parts[0]
		portStr := parts[1]

		port, err := strconv.Atoi(portStr)
		if err != nil {
			return "", fmt.Errorf("invalid port number: %s", portStr)
		}

		if port < 1 || port > 65535 {
			return "", fmt.Errorf("port must be between 1 and 65535")
		}

		if host == "" {
			return "", fmt.Errorf("host cannot be empty")
		}

		if !isValidHost(host) {
			return "", fmt.Errorf("invalid host: %s", host)
		}

		return hostname, nil
	}

	// Just a hostname without port - valid for some server types
	if !isValidHost(hostname) {
		return "", fmt.Errorf("invalid hostname: %s", hostname)
	}

	return hostname, nil
}

// isValidHost checks if a host string is a valid hostname or IP address
func isValidHost(host string) bool {
	if host == "" {
		return false
	}

	// Check for IPv4 address pattern first (to reject invalid IPs like 192.168.1.300)
	if ipv4Regex.MatchString(host) {
		// Validate each octet
		parts := strings.Split(host, ".")
		for _, part := range parts {
			num, err := strconv.Atoi(part)
			if err != nil || num < 0 || num > 255 {
				return false
			}
		}
		return true
	}

	// Check for valid hostname pattern (alphanumeric, dots, hyphens)
	// Must start with alphanumeric and not be purely numeric (to distinguish from invalid IPs)
	if hostnameRegex.MatchString(host) {
		// Make sure it's not a malformed IP (contains at least one letter)
		return hasLetterRe.MatchString(host)
	}

	return false
}

// ValidateServerType validates the server type against supported types
func ValidateServerType(serverType string) error {
	if serverType == "" {
		return fmt.Errorf("server type is required")
	}

	serverType = strings.ToLower(strings.TrimSpace(serverType))
	if !SupportedServerTypes()[serverType] {
		supported := make([]string, 0, len(SupportedServerTypes()))
		for t := range SupportedServerTypes() {
			supported = append(supported, t)
		}
		return fmt.Errorf("unsupported server type '%s', supported types: %s", serverType, strings.Join(supported, ", "))
	}

	return nil
}

// ValidateEmail validates an email address format
func ValidateEmail(email string) error {
	if email == "" {
		return nil // Empty is allowed (optional field)
	}

	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format: %s", email)
	}

	return nil
}

// ValidatePort validates a port number
func ValidatePort(port int) error {
	if port < 0 || port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535")
	}
	return nil
}

// ValidatePositiveInt validates that an integer is positive
func ValidatePositiveInt(value int, fieldName string) error {
	if value < 0 {
		return fmt.Errorf("%s must be a positive number", fieldName)
	}
	return nil
}
