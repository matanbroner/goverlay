package overlay

import "github.com/matanbroner/goverlay/cmd/id"

type Overlay struct {
	ID *id.PublicKeyId
}

func New() *Overlay {
	o := &Overlay{}

	return o
}
