package utils

import (
	"net/url"
	"strings"
)

// MaskPassword masks the password in a URL for safe display in logs and UI
// Example: ssh://user:secretpass@host:22 -> ssh://user:***@host:22
func MaskPassword(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		// If parsing fails, try to manually mask common patterns
		return maskPasswordManual(rawURL)
	}

	// If no password, return as-is
	if u.User == nil {
		return rawURL
	}

	_, hasPassword := u.User.Password()
	if !hasPassword {
		return rawURL
	}

	// Replace password with ***
	username := u.User.Username()
	u.User = url.UserPassword(username, "***")

	return u.String()
}

// maskPasswordManual attempts to mask password in URLs that might not parse correctly
func maskPasswordManual(rawURL string) string {
	// Pattern: user:password@host
	if strings.Contains(rawURL, "@") && strings.Contains(rawURL, ":") {
		parts := strings.Split(rawURL, "@")
		if len(parts) >= 2 {
			authPart := parts[0]
			restPart := strings.Join(parts[1:], "@")

			// Find last : in auth part (separator between user and password)
			lastColon := strings.LastIndex(authPart, ":")
			if lastColon > 0 {
				protocol := ""
				username := authPart[:lastColon]

				// Check if there's a protocol (ssh://, http://, etc.)
				if protoIdx := strings.Index(username, "://"); protoIdx > 0 {
					protocol = username[:protoIdx+3]
					username = username[protoIdx+3:]
				}

				return protocol + username + ":***@" + restPart
			}
		}
	}

	return rawURL
}

// MaskPasswords masks passwords in a slice of URLs
func MaskPasswords(urls []string) []string {
	masked := make([]string, len(urls))
	for i, u := range urls {
		masked[i] = MaskPassword(u)
	}
	return masked
}
