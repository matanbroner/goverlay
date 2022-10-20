package overlay

import (
	"github.com/google/uuid"
	"github.com/matanbroner/goverlay/lib/id"
	"github.com/matanbroner/goverlay/lib/message"
	"github.com/matanbroner/goverlay/lib/wrtc"
	"time"
)

type Overlay struct {
	ID              *id.PublicKeyId
	Listeners       []Listener
	Status          OverlayStatusMap
	PendingMessages []*message.Message
	WebRTCWrapper   *wrtc.WebRTCWrapper
}

type Signaler interface {
	SetConnection(connection *wrtc.WebRTCConnection)
	IsOverlay() bool
	AddConnection()
	Send(m *message.Message)
}

type Listener interface {
	OnMessage(m *message.Message)
}

func New(i *id.PublicKeyId) *Overlay {
	o := &Overlay{
		Status: OverlayStatusMap{
			IsBadNet:      false,
			IsInitialized: false,
			IsSubordinate: false,
		},
		ID:        i,
		Listeners: []Listener{},
	}

	return o
}

func (o *Overlay) OnMessage(m *message.Message) error {
	return nil
}

func (o *Overlay) SendMessage(m *message.Message) error {
	m.ID = uuid.New().String()
	m.Timestamp = time.Now()
	return nil
}

func (o *Overlay) ConnectionClosed(conn *wrtc.WebRTCConnection) error {
	return nil
}

func (o *Overlay) AddListener(l Listener) {
	o.Listeners = append(o.Listeners, l)
}

func (o *Overlay) InFloodRange(key string) bool {
	return false
}

func (o *Overlay) SendToClosest(m *message.Message) {
	// make sure to set id here, maybe by calling SendMessage
}

func (o *Overlay) Proxy(m *message.Message) {

}
