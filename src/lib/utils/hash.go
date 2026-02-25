package utils

import (
	crand "crypto/rand"
	"encoding/base64"
	"math/rand"
	"strconv"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// SecureRandomToken generates a cryptographically secure random token of the given byte length.
// It uses crypto/rand for secure random generation, making it suitable for security-sensitive
// applications like OAuth2 PKCE verifiers, session tokens, and API keys.
// The output is base64 URL-encoded (without padding) and will be approximately 4/3 the length
// of the input byte length.
//
// Example: SecureRandomToken(32) generates a 32-byte random value encoded as ~43 character string.
func SecureRandomToken(byteLength int) (string, error) {
	b := make([]byte, byteLength)

	if _, err := crand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// RandomToken returns a random token based on the given length. It builds the token
// from lowercase, uppercase English characters and integers.
//
// WARNING: This function uses math/rand/v2 which is NOT cryptographically secure.
// For security-sensitive applications (OAuth2 PKCE, session tokens, API keys),
// use SecureRandomToken instead.
func RandomToken(n int) string {
	src := rand.NewSource(time.Now().UnixNano())

	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
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
