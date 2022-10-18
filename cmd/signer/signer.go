package signer

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

func Pack(data string, privateKey *rsa.PrivateKey) (*SignedData, error) {
	publicKey := privateKey.Public().(*rsa.PublicKey)
	publicKeyBytes, err := json.Marshal(publicKey)
	if err != nil {
		return nil, err
	}
	packData := &PackableData{
		Data:      data,
		PublicKey: publicKeyBytes,
	}

	packDataBytes, err := json.Marshal(packData)
	if err != nil {
		return nil, err
	}
	hashed := sha256.Sum256(packDataBytes)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return nil, err
	}
	return &SignedData{
		Signed:    packDataBytes,
		Signature: signature,
	}, nil
}

func Unpack(id string, data *SignedData) (*PackableData, error) {
	packedData := &PackableData{}
	publicKey := &rsa.PublicKey{}
	if err := json.Unmarshal(data.Signed, packedData); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(packedData.PublicKey, publicKey); err != nil {
		return nil, err
	}
	hashed := sha256.Sum256(data.Signed)
	expectId := sha256.Sum256(packedData.PublicKey)
	if string(expectId[:]) != id {
		return nil, fmt.Errorf("id does not match on unpack: %s != %s", string(expectId[:]), id)
	} else if rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed[:], data.Signature) != nil {
		return nil, fmt.Errorf("hash signature does not match on unpack")
	} else {
		return packedData, nil
	}
}
