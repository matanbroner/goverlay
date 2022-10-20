package id

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"time"
)

const bitSize = 4096

type PublicKeyId struct {
	PrivateKey *rsa.PrivateKey
	ID         string
	PublicName string
	InstanceID *InstanceID
}

type InstanceID struct {
	UUID      string
	CreatedAt time.Time
	ID        string
}

// PublicKeyID Methods

func NewPublicKeyId(key *rsa.PrivateKey, publicName string) (*PublicKeyId, error) {
	if key == nil {
		privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
		if err != nil {
			return nil, err
		}
		key = privateKey
	}
	if publicName == "" {
		publicName = "_anonymous"
	}
	id := &PublicKeyId{
		PrivateKey: key,
	}
	publicKeyBytes, err := json.Marshal(key.Public())
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(publicKeyBytes)
	id.ID = hex.EncodeToString(hash[:])
	id.PublicName = publicName
	id.InstanceID = NewInstanceID()
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

// InstanceID Methods

func NewInstanceID() *InstanceID {
	id := &InstanceID{
		UUID:      uuid.New().String(),
		CreatedAt: time.Now(),
	}
	id.ID = fmt.Sprintf("%s/%s", id.UUID, id.CreatedAt)
	return id
}
