package overlay

import (
	"github.com/matanbroner/goverlay/lib/id"
	"github.com/matanbroner/goverlay/lib/message"
	"github.com/matanbroner/goverlay/lib/wrtc"
)

type Overlay struct {
	ID *id.PublicKeyId
}

type Signaler interface {
	SetConnection(connection *wrtc.WebRTCConnection)
	IsOverlay() bool
	AddConnection()
	Send(m *message.Message)
}

func New() *Overlay {
	o := &Overlay{}

	return o
}

func (o *Overlay) OnMessage(m *message.Message) error {
	return nil
}

func (o *Overlay) ConnectionClosed(conn *wrtc.WebRTCConnection) error {
	return nil
}
