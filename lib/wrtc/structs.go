package wrtc

import (
	"github.com/matanbroner/goverlay/lib/id"
	"github.com/matanbroner/goverlay/lib/overlay"
	"github.com/pion/webrtc/v3"
	"time"
)

type WebRTCWrapperConfig struct {
	IsInitiator bool
	PeerID      string
	InstanceID  *id.InstanceID
	Timestamp   time.Time
	Signaler    overlay.Signaler

	// websocketProxy
	// websocketProxyInstance
}

type WebRTCConnection struct {
	PeerID         string
	InstanceID     *id.InstanceID
	IsUsed         bool
	IsUsedByPeer   bool
	IsInitiator    bool
	IsGolden       bool
	IsClosed       bool
	IsOverlay      bool
	Timestamp      time.Time
	LastUsed       time.Time
	Signaler       overlay.Signaler
	PeerConnection *webrtc.PeerConnection
	Channel        *webrtc.DataChannel

	// pendingIce: []
}
