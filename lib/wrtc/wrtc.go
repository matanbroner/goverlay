package wrtc

import (
	"encoding/json"
	"fmt"
	"github.com/matanbroner/goverlay/lib/id"
	"github.com/matanbroner/goverlay/lib/message"
	"github.com/matanbroner/goverlay/lib/overlay"
	"github.com/matanbroner/goverlay/lib/signer"
	"github.com/matanbroner/goverlay/lib/util"
	"github.com/pion/webrtc/v3"
	"time"
)

const defaultChannel = "chat"

type WebRTCWrapper struct {
	ID              *id.PublicKeyId
	Overlay         *overlay.Overlay
	Connections     []*WebRTCConnection
	ConnectionsMap  map[string]*WebRTCConnection
	InstancesMap    map[string]*WebRTCConnection
	Listeners       []func()
	BinaryListeners []func(string, []byte)
	DeadTimestamps  []time.Time
}

func NewWebRTCWrapper(id *id.PublicKeyId, o *overlay.Overlay) *WebRTCWrapper {
	w := &WebRTCWrapper{
		ID:      id,
		Overlay: o,
	}
	return w
}

func (w *WebRTCWrapper) Start(config *WebRTCWrapperConfig) (*WebRTCConnection, error) {
	existingConnection := w.GetConnection(config.PeerID, config.InstanceID)
	if existingConnection != nil {
		existingConnection.IsUsed = true
	}
	if config.PeerID == w.ID.ID && config.InstanceID.UUID == w.ID.InstanceID.UUID {
		// do not connect to self
		return nil, nil
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
	connection.Index = len(w.Connections) - 1

	if config.PeerID == w.ID.ID {
		w.InstancesMap[config.InstanceID.UUID] = connection
	} else {
		w.ConnectionsMap[config.PeerID] = connection
	}
	pc, err := webrtc.NewPeerConnection(DefaultWebRTCConfig)
	if err != nil {
		return nil, err
	}
	connection.PeerConnection = pc
	if !connection.IsInitiator {
		// setup chat on incoming data channel
		connection.PeerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
			connection.Channel = dc
		})
	}
	connection.PeerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if !connection.IsClosed {
			connection.Signaler.Send(&message.Message{
				Data: message.MessageData{
					Candidate: *candidate,
				},
			})
		}
	})
	connection.PeerConnection.OnNegotiationNeeded(func() {
		if connection.IsInitiator {
			sd, err := connection.PeerConnection.CreateOffer(nil)
			if err != nil {
				fmt.Printf("wrtc offer creation error: %s\n", err.Error())
			}
			if err := connection.PeerConnection.SetLocalDescription(sd); err != nil {
				fmt.Printf("wrtc set local description error: %s\n", err.Error())
			}
			connection.Signaler.Send(&message.Message{
				Data: message.MessageData{
					SDP: *connection.PeerConnection.LocalDescription(),
				},
			})
		}
	})

	connection.PeerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		switch state {
		case webrtc.ICEConnectionStateClosed:
		case webrtc.ICEConnectionStateDisconnected:
		case webrtc.ICEConnectionStateFailed:
			if err := w.RemoveConnection(connection); err != nil {
				fmt.Printf("wrtc ice state change remove connection error: %s\n", err.Error())
			}
			if err := w.UpdateListeners(); err != nil {
				fmt.Printf("wrtc ice state update listeners error: %s\n", err.Error())
			}
		default:
			return
		}
	})
	if connection.IsInitiator {
		// create chat channel
		channel, err := connection.PeerConnection.CreateDataChannel(defaultChannel, nil)
		if err != nil {
			return nil, fmt.Errorf("wrtc create default channel error")
		}
		connection.Channel = channel
		if err := w.SetupDataChannel(connection); err != nil {
			return nil, fmt.Errorf("wrtc setup default channel error")
		}
	}
	return connection, nil
}

func (w *WebRTCWrapper) Stop() error {
	for k := range w.ConnectionsMap {
		if err := w.Disconnect(w.ConnectionsMap[k]); err != nil {
			return err
		}
	}
	for k := range w.InstancesMap {
		if err := w.Disconnect(w.InstancesMap[k]); err != nil {
			return err
		}
	}
	return nil
}

func (w *WebRTCWrapper) HandleSignal(peer string, instanceID *id.InstanceID, m *message.Message, signaler *overlay.Signaler) error {
	if util.Contains(w.DeadTimestamps, m.Timestamp) {
		return fmt.Errorf("wrtc dead timestamp in signal message")
	}
	conn := w.GetConnection(peer, instanceID)
	if conn == nil {
		return fmt.Errorf("wrtc no active connection for signal")
	}
	if conn.Timestamp.Before(m.Timestamp) {
		fmt.Println("wrtc expired connection for signal message")
		if err := w.Disconnect(conn); err != nil {
			return err
		}
		conn = nil
	}
	if conn != nil && m.Timestamp.Before(conn.Timestamp) {
		return fmt.Errorf("wrtc message timestamp before connection timestamp")
	}
	var sdp *webrtc.SessionDescription
	if m.Data.SDP == (webrtc.SessionDescription{}) {
		sdp = &m.Data.SDP
	}
	if conn != nil && sdp != nil && sdp.Type == webrtc.SDPTypeOffer && w.ID.ID == peer {
		fmt.Println("wrtc collision")
		if err := w.Disconnect(conn); err != nil {
			return err
		}
		newConn, err := w.Start(&WebRTCWrapperConfig{
			IsInitiator: false,
			PeerID:      peer,
			InstanceID:  instanceID,
			Signaler:    *signaler,
			Timestamp:   m.Timestamp,
		})
		if err != nil {
			return err
		}
		conn = newConn
	}
	if conn == nil {
		if sdp == nil {
			return fmt.Errorf("wrtc nil connection and no sdp")
		} else if sdp.Type != webrtc.SDPTypeOffer {
			return fmt.Errorf("wrtc sdp not offer recieved")
		} else {
			newConn, err := w.Start(&WebRTCWrapperConfig{
				IsInitiator: false,
				PeerID:      peer,
				InstanceID:  instanceID,
				Signaler:    *signaler,
				Timestamp:   m.Timestamp,
			})
			if err != nil {
				return err
			}
			conn = newConn
		}
	}
	if sdp != nil {
		if err := conn.PeerConnection.SetRemoteDescription(*sdp); err != nil {
			return fmt.Errorf("wrtc error on set remote description: %s", err.Error())
		}
		if sdp.Type == webrtc.SDPTypeOffer {
			// answer the offer
			_, err := conn.PeerConnection.CreateAnswer(nil)
			if err != nil {
				return fmt.Errorf("wrtc error on create sdp answer: %s", err.Error())
			}
			conn.Signaler.Send(&message.Message{
				Data: message.MessageData{
					SDP: *conn.PeerConnection.LocalDescription(),
				},
			})
		}
		for _, pending := range conn.PendingIce {
			if err := w.AddIce(pending, conn); err != nil {
				return fmt.Errorf("wrtc error on add pending ice: %s", err.Error())
			}
		}
		conn.PendingIce = []webrtc.ICECandidate{}
	} else if m.Data.Candidate != (webrtc.ICECandidate{}) {
		if conn.PeerConnection.RemoteDescription().Type == webrtc.SDPTypeOffer {
			if err := w.AddIce(m.Data.Candidate, conn); err != nil {
				return fmt.Errorf("wrtc error on add ice: %s", err.Error())
			} else {
				conn.PendingIce = append(conn.PendingIce, m.Data.Candidate)
			}
		}
	}
	return nil
}

func (w *WebRTCWrapper) AddIce(c webrtc.ICECandidate, conn *WebRTCConnection) error {
	return conn.PeerConnection.AddICECandidate(c.ToJSON())
}

func (w *WebRTCWrapper) OpenConnections() []*WebRTCConnection {
	var active []*WebRTCConnection
	for _, conn := range w.Connections {
		if w.IsActive(conn.PeerID) {
			active = append(active, conn)
		}
	}
	return active
}

func (w *WebRTCWrapper) IsActive(peer string) bool {
	conn := w.GetConnection(peer, nil)
	return conn != nil && conn.Channel != nil && conn.Channel.ReadyState() == webrtc.DataChannelStateOpen
}

func (w *WebRTCWrapper) GetConnection(peerID string, instanceID *id.InstanceID) *WebRTCConnection {
	if instanceID != nil && peerID == w.ID.ID {
		return w.InstancesMap[instanceID.UUID]
	} else {
		return w.ConnectionsMap[peerID]
	}
}

func (w *WebRTCWrapper) RemoveConnection(conn *WebRTCConnection) error {
	w.DeadTimestamps = append(w.DeadTimestamps, conn.Timestamp)
	w.Connections = append(w.Connections[:conn.Index], w.Connections[conn.Index+1:]...)
	if w.ConnectionsMap[conn.PeerID] == conn {
		delete(w.ConnectionsMap, conn.PeerID)
	}
	if w.InstancesMap[conn.InstanceID.UUID] == conn {
		delete(w.InstancesMap, conn.InstanceID.UUID)
	}
	if err := w.Overlay.ConnectionClosed(conn); err != nil {
		return err
	}
	return w.UpdateListeners()
}

func (w *WebRTCWrapper) Disconnect(conn *WebRTCConnection) error {
	if conn == nil {
		return fmt.Errorf("wrtc disconnect nil connection")
	}
	if conn.Channel != nil {
		if err := conn.Channel.Close(); err != nil {
			return fmt.Errorf("wrtc channel close error: %s", err.Error())
		}
	}
	if conn.PeerConnection != nil && conn.PeerConnection.SignalingState() != webrtc.SignalingStateClosed {
		if err := conn.PeerConnection.Close(); err != nil {
			return fmt.Errorf("wrtc peer connection close error: %s", err.Error())
		}
	}
	return w.RemoveConnection(conn)
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
				fmt.Printf("wrtc message parse error: %s\n", err.Error())
			}
			unpacked, err := signer.Unpack(packed, w.ID)
			if err != nil {
				fmt.Println(err.Error())
			}
			msg.EncodedData = []byte(unpacked.Data)
		}
		if err := json.Unmarshal(msg.EncodedData, &msg.Data); err != nil {
			fmt.Printf("wrtc message parse error: %s\n", err.Error())
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
						if err := w.Disconnect(conn); err != nil {
							fmt.Printf("wrtc disconnect channel error: %s\n", err.Error())
						}
					}
				}
			case message.Disconnect:
				{
					if err := w.Disconnect(conn); err != nil {
						fmt.Printf("wrtc disconnect channel error: %s\n", err.Error())
					}
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
			for _, listener := range w.BinaryListeners {
				listener(conn.PeerID, m.Data)
			}
		}
	})
	conn.Channel.OnClose(func() {
		if err := w.RemoveConnection(conn); err != nil {
			fmt.Printf("wrtc remove connection error: %s\n", err.Error())
		}
	})
	if conn.Channel.ReadyState() == webrtc.DataChannelStateOpen {
		conn.Signaler.AddConnection()
		if err := w.UpdateListeners(); err != nil {
			fmt.Printf("wrtc update listeners error: %s\n", err.Error())
		}
	} else {
		conn.Channel.OnOpen(func() {
			conn.Signaler.AddConnection()
			if err := w.UpdateListeners(); err != nil {
				fmt.Printf("wrtc update listeners error: %s\n", err.Error())
			}
		})
	}
	conn.Channel.OnError(func(err error) {
		fmt.Printf("wrtc channel error: %s", err.Error())
	})
	return nil
}

func (w *WebRTCWrapper) Send(m *message.Message) error {
	instanceID := id.InstanceIDFromString(m.Data.FromInstance)
	if instanceID == nil {
		return fmt.Errorf("wrtc invalid instance id in outgoing message: %s", m.Data.FromInstance)
	}
	conn := w.GetConnection(m.Data.To, instanceID)
	if conn == nil {
		return fmt.Errorf("wrtc no active connection for (%s, %s)", m.Data.To, instanceID.UUID)
	}
	if conn.Channel.ReadyState() != webrtc.DataChannelStateOpen {
		return fmt.Errorf("wrtc channel not open")
	}
	m.Data.From = w.ID.ID
	m.Timestamp = time.Now()
	bytes, err := json.Marshal(m.Data)
	if err != nil {
		return fmt.Errorf("wrtc message marshall error: %s", err.Error())
	}
	m.EncodedData = bytes
	bytes, err = json.Marshal(m)
	if err != nil {
		return fmt.Errorf("wrtc message marshall error: %s", err.Error())
	}
	if err := conn.Channel.Send(bytes); err != nil {
		return fmt.Errorf("wrtc message send error: %s", err.Error())
	}
	return nil
}

func (w *WebRTCWrapper) SendRaw(peer string, bytes []byte) error {
	conn := w.GetConnection(peer, nil)
	if conn == nil {
		return fmt.Errorf("wrtc no active connection for (%s)", peer)
	}
	if conn.Channel.ReadyState() != webrtc.DataChannelStateOpen {
		return fmt.Errorf("wrtc channel not open")
	}
	if err := conn.Channel.Send(bytes); err != nil {
		return fmt.Errorf("wrtc message send error: %s", err.Error())
	}
	return nil
}

func (w *WebRTCWrapper) MarkUsed(peer string) error {
	conn := w.GetConnection(peer, nil)
	if conn == nil {
		return fmt.Errorf("wrtc no active connection for (%s)", peer)
	}
	if err := w.Send(&message.Message{
		Data: message.MessageData{
			To:     peer,
			Action: message.MarkUsedByPeer,
		},
		Timestamp: time.Now(),
	}); err != nil {
		return err
	}
	conn.IsUsed = true
	return nil
}

func (w *WebRTCWrapper) MarkUnused(peer string) error {
	conn := w.GetConnection(peer, nil)
	if conn == nil {
		return fmt.Errorf("wrtc no active connection for (%s)", peer)
	}
	if conn.IsUsed && conn.IsUsedByPeer {
		if err := w.Send(&message.Message{
			Data: message.MessageData{
				To:     peer,
				Action: message.MarkUnusedByPeer,
			},
			Timestamp: time.Now(),
		}); err != nil {
			return err
		}
		conn.IsUsed = false
	} else {
		return w.Disconnect(conn)
	}
	return nil
}

func (w *WebRTCWrapper) UpdateListeners() error {
	for _, listen := range w.Listeners {
		listen()
	}
	return nil
}
