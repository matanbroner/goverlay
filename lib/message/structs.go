package message

import (
	"github.com/pion/webrtc/v3"
	"time"
)

type MessageData struct {
	To           string                    `json:"to"`
	ToInstance   string                    `json:"toInstance"`
	From         string                    `json:"from"`
	FromInstance string                    `json:"fromInstance"`
	Action       string                    `json:"action"`
	Proxies      []string                  `json:"proxies"`
	Confirmed    string                    `json:"confirmed"`
	Value        []byte                    `json:"value"`
	SDP          webrtc.SessionDescription `json:"sdp"`
	Candidate    webrtc.ICECandidate       `json:"candidate"`
}

type Message struct {
	EncodedData []byte    `json:"data"`
	Packed      bool      `json:"packed"`
	ID          string    `json:"id"`
	AckID       string    `json:"ackID"`
	Timestamp   time.Time `json:"timestamp"`
	Data        MessageData
}
