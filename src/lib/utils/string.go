package utils

import (
	"net/mail"
	"regexp"
	"strconv"
	"strings"
)

// ReplaceAllWhitespaces replaces all whitespace characters (spaces, tabs, newlines)
// in the input string with the specified replacement string.
// Multiple consecutive whitespaces are treated as a single occurrence.
func ReplaceAllWhitespaces(s, with string) string {
	re := regexp.MustCompile(`\s{1,}|\n`)
	return re.ReplaceAllString(s, with)
}

// StringToInt converts a string to an integer.
// If the conversion fails, it returns 0.
func StringToInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

// StringToInt64 converts a string to a 64-bit integer.
// If the conversion fails, it returns 0.
func StringToInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

// Int64ToString converts a 64-bit integer to its string representation.
func Int64ToString(n int64) string {
	return strconv.FormatInt(n, 10)
}

// GetString returns the first non-empty string from the provided values.
// If all values are empty strings, it returns an empty string.
// This is useful for providing fallback values.
func GetString(values ...string) string {
	for _, val := range values {
		if val != "" {
			return val
		}
	}

	return ""
}

// IsValidEmail validates whether the provided string is a valid email address.
// It uses Go's built-in mail.ParseAddress function for validation.
// Returns true if the email is valid, false otherwise.
func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// TrimPath cleans up the path by removing leading ./, /, and trimming spaces.
// If path is empty, it returns an empty string. If path is just ".", it returns "/".
func TrimPath(path string) string {
	path = strings.TrimSpace(path)

	if path == "" {
		return ""
	}

	// Remove initial ./
	path = strings.TrimPrefix(path, "./")

	// Remove standalone .
	if path == "." {
		return "/"
	}

	// Remove initial and final /
	return "/" + strings.Trim(path, "/")
}

// ParseSemver parses a semantic version string (e.g., "1.2.3") and returns
// the major, minor, and patch components as integers.
func ParseSemver(version string) (major, minor, patch string) {
	trimmed := strings.TrimPrefix(version, "v")
	parts := strings.SplitN(trimmed, ".", 3)
	major, minor, patch = "0", "0", "0"

	switch len(parts) {
	case 1:
		major = trimmed
	case 2:
		major = parts[0]
		minor = parts[1]
	case 3:
		major = parts[0]
		minor = parts[1]
		patch = parts[2]
	}

	return major, minor, patch
}

// NormalizeURL ensures that the given URL starts with "http://" or "https://"
// and removes any trailing slashes.
func NormalizeURL(url string) string {
	if !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
		url = "http://" + url
	}

	return strings.TrimRight(url, "/")
}
