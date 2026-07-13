package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

var appSecret []byte

// SetAppKey sets an app key to be used as a default key.
func SetAppKey(key []byte) {
	appSecret = key
}

// Hash hashes the given array of bytes and returns a string out of it.
func Hash(text []byte) string {
	hasher := sha1.New()
	hasher.Write(text)
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

// SHA256Hash hashes the given array of bytes using SHA256 and returns a string out of it.
func SHA256Hash(b []byte) string {
	h := sha256.New()
	h.Write(b)
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

type EncryptToStringFunc = func(plaintext string, altKey ...[]byte) string

// Encrypts the given string, and then uses base64 to encode it.
// This function ignores the error and returns an empty string if it fails.
var EncryptToString = func(plaintext string, altKey ...[]byte) string {
	encrypted, err := Encrypt([]byte(plaintext), altKey...)

	if err != nil {
		return ""
	}

	return EncodeToString(encrypted)
}

// Encrypts the given string, and then uses base64 to encode it.
// This function ignores the error and returns an empty string if it fails.
func DecryptToString(encodedAndEncrypted string, altKey ...[]byte) string {
	decodedAndEncrypted, err := DecodeString(encodedAndEncrypted)

	if err != nil {
		return ""
	}

	decrypted, _ := Decrypt(decodedAndEncrypted, altKey...)
	return string(decrypted)
}

// Encrypt encrypts data using 256-bit AES-GCM.  This both hides the content of
// the data and provides a check that it hasn't been altered. Output takes the
// form nonce|ciphertext|tag where '|' indicates concatenation.
// altKey is the alternative key. If nothing is provided, this method will use
// the default encryption key. If provided, it will use ALWAYS the first key.
func Encrypt(plaintext []byte, altKey ...[]byte) (ciphertext []byte, err error) {
	var key []byte

	if len(altKey) > 0 {
		key = altKey[0]
	} else {
		key = appSecret
	}

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts data using 256-bit AES-GCM.  This both hides the content of
// the data and provides a check that it hasn't been altered. Expects input
// form nonce|ciphertext|tag where '|' indicates concatenation.
// altKey is the alternative key. If nothing is provided, this method will use
// the default encryption key. If provided, it will use ALWAYS the first key.
func Decrypt(ciphertext []byte, altKey ...[]byte) (plaintext []byte, err error) {
	var key []byte

	if len(altKey) > 0 {
		key = altKey[0]
	} else {
		key = appSecret
	}

	if len(key) == 0 {
		return nil, errors.New("no encryption key provided")
	}

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("malformed ciphertext")
	}

	return gcm.Open(nil,
		ciphertext[:gcm.NonceSize()],
		ciphertext[gcm.NonceSize():],
		nil,
	)
}

// EncryptID encrypts the given ID into a string
func EncryptID(ID types.ID) string {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint32(bs, uint32(ID))
	secret, _ := Encrypt(bs)
	return EncodeToString(secret)
}

// DecryptID decrypts the given encrypted string and returns a
// types.ID ID from that. If the decryption fails it returns an error.
func DecryptID(encryptedString string) (types.ID, error) {
	idBytes, _ := DecodeString(encryptedString)
	idBytes, err := Decrypt(idBytes)

	if err != nil {
		return 0, err
	}

	return types.ID(binary.LittleEndian.Uint32(idBytes)), nil
}

// EncodeToString casts an array of bytes to a string.
func EncodeToString(text []byte) string {
	return base64.URLEncoding.EncodeToString(text)
}

// DecodeString decodes the given string to an array of bytes.
func DecodeString(s string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(s)
}

// Scan implements the Scanner interface.
func ByteaScan(value any, out any) error {
	if value == nil {
		return nil
	}

	b, ok := value.([]byte)

	if !ok {
		return fmt.Errorf("failed to scan: invalid type %T", value)
	}

	decrypted, err := Decrypt(b)

	if err != nil {
		return err
	}

	return json.Unmarshal(decrypted, out)
}

// Value implements the Sql Driver interface.
func ByteaValue(ac any) (driver.Value, error) {
	if ac == nil {
		return nil, nil
	}

	js, err := json.Marshal(ac)

	if err != nil {
		return nil, err
	}

	return Encrypt(js)
}
