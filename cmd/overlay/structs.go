package overlay

import (
	"github.com/matanbroner/goverlay/cmd/id"
	"github.com/matanbroner/goverlay/cmd/signer"
)

type Envelope struct {
	To            string
	From          *id.PublicKeyId
	Data          *signer.SignedData
	Action        string
	OverlayAction string
	Proxies       []string
}
