package utils_test

import (
	"bytes"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CryptSuite struct {
	suite.Suite
}

func (s *CryptSuite) TestEncodingAndDecoding() {
	myBytes := []byte("example.org")
	encoded := utils.EncodeToString(myBytes)
	decoded, err := utils.DecodeString(encoded)

	a := assert.New(s.T())
	a.NoError(err)
	a.Zero(bytes.Compare(decoded, myBytes))
	a.Equal("ZXhhbXBsZS5vcmc=", encoded)
}

func (s *CryptSuite) TestEncodingEmpty() {
	assert.Equal(s.T(), "", utils.EncodeToString(nil))
}

func (s *CryptSuite) TestEncryptDecrypt() {
	a := assert.New(s.T())

	myBytes := []byte("schedule")
	myKey := []byte("my-key-with-that-is-32-len------")
	encrypted, err := utils.Encrypt(myBytes, myKey)

	a.NoError(err)

	decrypted, err := utils.Decrypt(encrypted, myKey)

	a.NoError(err)
	a.Zero(bytes.Compare(decrypted, myBytes))
}

func (s *CryptSuite) TestHash() {
	assert.Equal(s.T(), "e1AsOh9IyGCa4hLN-2Od7jlnP14=", utils.Hash([]byte("Hello world")))
}

func TestCrypt(t *testing.T) {
	suite.Run(t, &CryptSuite{})
}
