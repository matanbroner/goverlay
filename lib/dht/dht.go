package dht

import "github.com/matanbroner/goverlay/lib/overlay"

type DHT struct {
	overlay *overlay.Overlay
}

type OverlayMessageListener struct {
	dht *DHT
}

// DHT Methods

func New(overlay *overlay.Overlay) *DHT {
	d := &DHT{
		overlay: overlay,
	}

	return d
}

// OverlayMessageListener Methods

func (oml *OverlayMessageListener) onMessage(e *Envelope) {

}
