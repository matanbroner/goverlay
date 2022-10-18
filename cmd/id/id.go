package id

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const bitSize = 4096

type PublicKeyId struct {
	PublicKey *rsa.PublicKey
	ID        string
}

func NewPublicKeyId(key *rsa.PublicKey) (*PublicKeyId, error) {
	if key == nil {
		privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
		if err != nil {
			return nil, err
		}
		key = privateKey.Public().(*rsa.PublicKey)
	}
	id := &PublicKeyId{
		PublicKey: key,
	}
	publicKeyBytes, err := json.Marshal(key)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(publicKeyBytes)
	id.ID = hex.EncodeToString(hash[:])

	return id, nil
}

func (id *PublicKeyId) MatchesPublicKey(key *rsa.PublicKey) (bool, error) {
	publicKeyBytes, err := json.Marshal(key)
	if err != nil {
		return false, err
	}
	hash := sha256.Sum256(publicKeyBytes)
	return hex.EncodeToString(hash[:]) == id.ID, nil
}
