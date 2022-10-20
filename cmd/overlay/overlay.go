package overlay

import (
	"github.com/matanbroner/goverlay/cmd/id"
	"github.com/matanbroner/goverlay/cmd/message"
	"github.com/matanbroner/goverlay/cmd/wrtc"
)

type Overlay struct {
	ID *id.PublicKeyId
}

type Signaler interface {
	SetConnection(connection *wrtc.WebRTCConnection)
	IsOverlay() bool
	AddConnection()
}

func New() *Overlay {
	o := &Overlay{}

	return o
}

func (o *Overlay) OnMessage(m *message.Message) error {
	return nil
}
