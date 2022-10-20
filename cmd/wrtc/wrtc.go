package wrtc

import (
	"encoding/json"
	"fmt"
	"github.com/matanbroner/goverlay/cmd/id"
	"github.com/matanbroner/goverlay/cmd/message"
	"github.com/matanbroner/goverlay/cmd/overlay"
	"github.com/matanbroner/goverlay/cmd/signer"
	"github.com/pion/webrtc/v3"
	"time"
)

type WebRTCWrapper struct {
	ID             *id.PublicKeyId
	Overlay        *overlay.Overlay
	Connections    []*WebRTCConnection
	ConnectionsMap map[string]*WebRTCConnection
	InstancesMap   map[string]*WebRTCConnection
}

func NewOverlay(id *id.PublicKeyId, o *overlay.Overlay) *WebRTCWrapper {
	w := &WebRTCWrapper{
		ID:      id,
		Overlay: o,
	}
	return w
}

func (w *WebRTCWrapper) Start(config *WebRTCWrapperConfig) error {
	existingConnection := w.GetConnection(config.PeerID, config.InstanceID)
	if existingConnection != nil {
		existingConnection.IsUsed = true
	}
	if config.PeerID == w.ID.ID && config.InstanceID.UUID == w.ID.InstanceID.UUID {
		// do not connect to self
		return nil
	}
	connection := &WebRTCConnection{
		PeerID:       config.PeerID,
		InstanceID:   config.InstanceID,
		IsUsed:       true,
		IsUsedByPeer: true,
		IsInitiator:  config.IsInitiator,
		IsGolden:     false,
		IsClosed:     false,
		Timestamp:    config.Timestamp,
		LastUsed:     time.Now(),
		IsOverlay:    config.Signaler.IsOverlay(),
	}
	connection.Signaler = config.Signaler

	config.Signaler.SetConnection(connection)
	w.Connections = append(w.Connections, connection)

	if config.PeerID == w.ID.ID {
		w.InstancesMap[config.InstanceID.UUID] = connection
	} else {
		w.ConnectionsMap[config.PeerID] = connection
	}
	pc, err := webrtc.NewPeerConnection(DefaultWebRTCConfig)
	if err != nil {
		return err
	}
	connection.PeerConnection = pc
	if !connection.IsInitiator {
		// setup chat on incoming data channel
		connection.PeerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
			connection.Channel = dc
		})
	}
	return nil
}

func (w *WebRTCWrapper) GetConnection(peerID string, instanceID *id.InstanceID) *WebRTCConnection {
	if instanceID != nil && peerID == w.ID.ID {
		return w.InstancesMap[instanceID.UUID]
	} else {
		return w.ConnectionsMap[peerID]
	}
}

func (w *WebRTCWrapper) SetupDataChannel(conn *WebRTCConnection) error {
	conn.Channel.OnMessage(func(m webrtc.DataChannelMessage) {
		if len(m.Data) == 0 {
			return
		}
		msg := &message.Message{}
		if err := json.Unmarshal(m.Data, msg); err != nil {
			fmt.Printf("wrtc message parse error: %s", err.Error())
		}
		if msg.Packed {
			packed := &signer.SignedData{}
			if err := json.Unmarshal(msg.EncodedData, packed); err != nil {
				fmt.Printf("wrtc message parse error: %s", err.Error())
			}
			unpacked, err := signer.Unpack(packed, w.ID)
			if err != nil {
				fmt.Println(err.Error())
			}
			msg.EncodedData = []byte(unpacked.Data)
		}
		if err := json.Unmarshal(msg.EncodedData, &msg.Data); err != nil {
			fmt.Printf("wrtc message parse error: %s", err.Error())
		}
		if m.IsString {
			switch msg.Data.Action {
			case message.MarkUsedByPeer:
				{
					conn.IsUsedByPeer = true
					break
				}
			case message.MarkUnusedByPeer:
				{
					conn.IsUsedByPeer = false
					if !conn.IsUsed {
						// w.Disconnect()
					}
				}
			case message.Disconnect:
				{
					// w.Disconnect()
				}
			case message.OverlayMessage:
				{
					if err := w.Overlay.OnMessage(msg); err != nil {
						fmt.Printf("wrtc overlay message handler error: %s", err.Error())
					}
				}
			default:
				fmt.Printf("wrtc unrecognized message action: %s", msg.Data.Action)
			}
		} else {
			// this.binaryMessageListeners.forEach(l => l(connection.peerId, event.data));
		}
	})
	conn.Channel.OnClose(func() {
		// w.RemoveConnection(conn)
	})
	if conn.Channel.ReadyState() == webrtc.DataChannelStateOpen {
		conn.Signaler.AddConnection()
		// w.UpdateListeners()
	} else {
		conn.Channel.OnOpen(func() {
			conn.Signaler.AddConnection()
			// w.UpdateListeners()
		})
	}
	conn.Channel.OnError(func(err error) {
		fmt.Printf("wrtc channel error: %s", err.Error())
	})
	return nil
}
