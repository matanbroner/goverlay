package signer

type PackableData struct {
	Data      string `json:"data"`
	PublicKey []byte `json:"publicKey"`
}

type SignedData struct {
	Signed    []byte
	Signature []byte
}
