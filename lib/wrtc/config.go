package wrtc

import "github.com/pion/webrtc/v3"

var DefaultWebRTCConfig = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.stunprotocol.org"},
		},
	},
}
