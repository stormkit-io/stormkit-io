package utils

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql/driver"
	"encoding/pem"
	"errors"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"golang.org/x/crypto/ssh"
)

var size = 4096

// PrivateKey is a custom private key which supports database serialization
// and json marshaling. The private key will be stored encrypted in the database
// and will be decrypted upon request.
type PrivateKey struct {
	*rsa.PrivateKey
}

// SetKeySize sets the key size. This is used for testing
// as if the key size is large it's gonna take a while to execute
// all the tests.
func SetKeySize(s int) {
	size = s
}

// RestoreKeySize restores to default. This is used for testing
// as if the key size is large it's gonna take a while to execute
// all the tests.
func RestoreKeySize() {
	size = 4096
}

// NewPrivateKey generates a new private key.
func NewPrivateKey() *PrivateKey {
	pk, err := rsa.GenerateKey(rand.Reader, size)

	if err != nil {
		slog.Error(err.Error())
		return nil
	}

	return &PrivateKey{pk}
}

// NewPrivateKeyFromDecryptedBytes returns a new private key instance
// from the decrypted bytes.
func NewPrivateKeyFromDecryptedBytes(b []byte) (*PrivateKey, error) {
	pkey := &PrivateKey{}

	if err := pkey.Scan(b); err != nil {
		return nil, err
	}

	return pkey, nil
}

// PrivKey returns the original private key.
func (pk *PrivateKey) PrivKey() *rsa.PrivateKey {
	return pk.PrivateKey
}

// SSHPubKey returns an ssh public key.
func (pk *PrivateKey) SSHPubKey() string {
	signer, err := ssh.NewSignerFromKey(pk.PrivateKey)

	if err != nil {
		return ""
	}

	return string(ssh.MarshalAuthorizedKey(signer.PublicKey()))
}

// SSHPrivKey returns an ssh private key.
func (pk *PrivateKey) SSHPrivKey() string {
	return string(pemEncode(pk.PrivateKey))
}

// Encrypt encryptes the private key.
func (pk *PrivateKey) Encrypt() ([]byte, error) {
	return Encrypt(pemEncode(pk.PrivateKey))
}

// Value implements database/sql interface.
func (pk *PrivateKey) Value() (driver.Value, error) {
	return pk.Encrypt()
}

// Scan implements database/sql interface.
func (pk *PrivateKey) Scan(value interface{}) error {
	var block *pem.Block
	var err error

	key, ok := value.([]byte)

	if !ok {
		return errors.New("invalid type")
	}

	if key, err = Decrypt(key); err != nil {
		return err
	}

	if block, err = pemDecode(key); err != nil {
		return err
	}

	pk.PrivateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	return err
}

// pemEncode encodes the given interface into an array of bytes.
func pemEncode(data interface{}) []byte {
	var pemBlock *pem.Block
	switch key := data.(type) {
	case *ecdsa.PrivateKey:
		keyBytes, _ := x509.MarshalECPrivateKey(key)
		pemBlock = &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}
	case *rsa.PrivateKey:
		pemBlock = &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
	case *x509.CertificateRequest:
		pemBlock = &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: key.Raw}
	case []byte:
		pemBlock = &pem.Block{Type: "CERTIFICATE", Bytes: []byte(data.([]byte))}
	}

	return pem.EncodeToMemory(pemBlock)
}

// pemDecodes decodes the given key into a pem Block.
func pemDecode(data []byte) (*pem.Block, error) {
	pemBlock, _ := pem.Decode(data)

	if pemBlock == nil {
		return nil, errors.New("pem decode did not yield a valid block. Is the certificate in the right format?")
	}

	return pemBlock, nil
}
