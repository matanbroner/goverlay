package message

import (
	"github.com/pion/webrtc/v3"
	"time"
)

type MessageData struct {
	To            string   `json:"to"`
	ToInstance    string   `json:"toInstance"`
	From          string   `json:"from"`
	FromInstance  string   `json:"fromInstance"`
	Message       string   `json:"message"`
	Action        string   `json:"action"`
	OverlayAction string   `json:"overlayAction"`
	Proxies       []string `json:"proxies"`
	Confirmed     string   `json:"confirmed"`
	Data          string   `json:"data"`
	SDP           webrtc.SessionDescription
	Candidate     webrtc.ICECandidate
}

type Message struct {
	EncodedData []byte    `json:"data"`
	Packed      bool      `json:"packed"`
	Timestamp   time.Time `json:"timestamp"`
	Data        MessageData
}
