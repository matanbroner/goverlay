package signer

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func generatePrivateKey() (*rsa.PrivateKey, error) {
	// https://stackoverflow.com/a/64105068
	bitSize := 4096

	// Generate RSA key.
	key, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func TestPack(t *testing.T) {
	privateKey, err := generatePrivateKey()
	if err != nil {
		t.Fatalf(err.Error())
	}
	message := "Hello, World!"
	packed, err := Pack(message, privateKey)
	if err != nil {
		t.Fatalf(err.Error())
	}
	packedData := &PackableData{}
	if err := json.Unmarshal(packed.Signed, packedData); err != nil {
		t.Fatalf(err.Error())
	}
	publicKeyBytes, err := json.Marshal(privateKey.Public())
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.Equal(t, publicKeyBytes, packedData.PublicKey)
	assert.Equal(t, message, packedData.Data)
}

func TestUnpack(t *testing.T) {
	privateKey, err := generatePrivateKey()
	if err != nil {
		t.Fatalf(err.Error())
	}
	message := "Hello, World!"
	packed, err := Pack(message, privateKey)
	if err != nil {
		t.Fatalf(err.Error())
	}
	// test for id mismatch
	_, err = Unpack("wrong-id", packed)
	assert.NotNil(t, err)

	publicKeyBytes, err := json.Marshal(privateKey.Public())
	if err != nil {
		t.Fatalf(err.Error())
	}
	publicKeySum := sha256.Sum256(publicKeyBytes)
	unpacked, err := Unpack(string(publicKeySum[:]), packed)
	assert.Nil(t, err)
	assert.Equal(t, unpacked.Data, message)
	assert.Equal(t, unpacked.PublicKey, publicKeyBytes)
}
