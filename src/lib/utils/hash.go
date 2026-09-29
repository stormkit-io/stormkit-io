package utils

import (
	crand "crypto/rand"
	"encoding/base64"
	"strconv"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
)

// SecureRandomToken generates a cryptographically secure random token of the given byte length.
// It uses crypto/rand for secure random generation, making it suitable for security-sensitive
// applications like OAuth2 PKCE verifiers, session tokens, and API keys.
// The output is base64 URL-encoded (without padding) and will be approximately 4/3 the length
// of the input byte length.
//
// Example: SecureRandomToken(32) generates a 32-byte random value encoded as ~43 character string.
//
// The encoding is unpadded, which is where that 43 comes from — the padded
// EncodeToString in crypt.go would yield 44 and change the shape of every
// token this issues.
func SecureRandomToken(byteLength int) (string, error) {
	b := make([]byte, byteLength)

	if _, err := crand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// RandomToken returns a cryptographically secure random token of n characters
// built from lowercase and uppercase English letters and digits.
func RandomToken(n int) string {
	token := make([]byte, 0, n)
	buf := make([]byte, n)

	for len(token) < n {
		// crypto/rand.Read always fills buf and never returns an error.
		_, _ = crand.Read(buf)

		// Rejection sampling keeps every character equally likely.
		for _, c := range buf {
			if len(token) == n {
				break
			}

			if idx := int(c & letterIdxMask); idx < len(letterBytes) {
				token = append(token, letterBytes[idx])
			}
		}
	}

	return string(token)
}

// StringToID takes a string number as an argument, parses it
// and returns a types.ID value.
func StringToID(id string) types.ID {
	idInt, err := strconv.ParseInt(id, 10, 64)

	if err != nil {
		return 0
	}

	return types.ID(idInt)
}
