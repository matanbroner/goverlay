package ws

import (
	"encoding/json"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/gorilla/websocket"
	"github.com/matanbroner/goverlay/lib/id"
	"github.com/matanbroner/goverlay/lib/message"
	"github.com/matanbroner/goverlay/lib/overlay"
	"github.com/matanbroner/goverlay/lib/signer"
	"log"
	"os"
	"os/signal"
	"time"
)

type WebSocketWrapper struct {
	ID                           *id.PublicKeyId
	Overlay                      *overlay.Overlay
	Host                         string
	ConnectionMap                *map[string]string
	RetrySeconds                 int
	SuccessfulFirstConnectionSet *mapset.Set[string]
	Socket                       *websocket.Conn
	Graceful                     bool
	MessageOutChannel            chan message.Message
	DoneChannel                  chan struct{}
	InterruptChannel             chan os.Signal

	// this.webrtc = overlay.webrtc;
	// this.webrtc = overlay.webrtc;
	// this.onNetworkUpdate = onNetworkUpdate;
	//	this.overlay.messageListeners.push(new OverlayMessageListener(this));
	//	this.pingHandle = setInterval(()=> {
	//	this.ping();
	//}, 30 * 1000);
}

func NewWebSocketWrapper(o *overlay.Overlay, host string) *WebSocketWrapper {
	connectionMap := make(map[string]string)
	successConnectionSet := mapset.NewSet[string]()
	return &WebSocketWrapper{
		ID:                           o.ID,
		Host:                         host,
		Overlay:                      o,
		ConnectionMap:                &connectionMap,
		RetrySeconds:                 1,
		SuccessfulFirstConnectionSet: &successConnectionSet,
	}
}

func (ws *WebSocketWrapper) Connect(config *WebSocketConfig) error {
	if config == nil {
		config = &WebSocketConfig{
			Reconnect: false,
		}
	}

	ws.Graceful = false
	sock, _, err := websocket.DefaultDialer.Dial(ws.Host, nil)
	if err != nil {
		return err
	}
	ws.Socket = sock

	ws.DoneChannel = make(chan struct{})
	ws.MessageOutChannel = make(chan message.Message)
	ws.InterruptChannel = make(chan os.Signal, 1)
	signal.Notify(ws.InterruptChannel, os.Interrupt)

	go func() {
		defer close(ws.DoneChannel)
		for {
			_, m, err := ws.Socket.ReadMessage()
			if err != nil {
				log.Println("ws error:", err.Error())
				return
			}
			if err := ws.onMessage(m); err != nil {
				log.Println("ws error:", err.Error())
				return
			}
		}
	}()

	go ws.handleChannels()

	action := message.Connect
	if config.Reconnect {
		action = message.Reconnect
	}

	m := message.Message{
		Data: message.MessageData{
			Action:       action,
			From:         ws.ID.ID,
			FromInstance: ws.ID.InstanceID.ID,
		},
		Packed: true,
	}

	ws.MessageOutChannel <- m

	//if (this.blockCallback) {
	//	this.sendGetBlock();
	//}

	// TODO: handle graceful reconnection, ws.onclose -> reconnect
	// see https://github.com/recws-org/recws

	return nil
}

func (ws *WebSocketWrapper) Disconnect() error {
	if ws.Socket != nil {
		if err := ws.Socket.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (ws *WebSocketWrapper) Confirm(id string) error {
	ws.MessageOutChannel <- message.Message{
		Data: message.MessageData{
			Action:       message.Confirm,
			Confirmed:    id,
			From:         ws.ID.ID,
			FromInstance: ws.ID.InstanceID.ID,
		},
		Packed: true,
	}
	return nil
}

func (ws *WebSocketWrapper) Send(to string, toInstance string, data string, packed bool) error {
	ws.MessageOutChannel <- message.Message{
		Data: message.MessageData{
			To:           to,
			ToInstance:   toInstance,
			From:         ws.ID.ID,
			FromInstance: ws.ID.InstanceID.ID,
			Data:         data,
		},
		Packed: packed,
	}
	return nil
}

func (ws *WebSocketWrapper) SendGetBlock() error {
	ws.MessageOutChannel <- message.Message{
		Data: message.MessageData{
			Action:       message.GetBlock,
			From:         ws.ID.ID,
			FromInstance: ws.ID.InstanceID.ID,
		},
		Packed: true,
	}
	return nil
}

func (ws *WebSocketWrapper) encodeRawMessage(message string, pack bool) ([]byte, error) {
	payload := json.RawMessage(message)
	payloadBytes, err := payload.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if pack {
		signed, err := signer.Pack(string(payloadBytes), ws.ID.PrivateKey)
		if err != nil {
			return nil, err
		}
		signed.VerifyID = ws.ID.ID
		signedBytes, err := json.Marshal(signed)
		if err != nil {
			return nil, err
		}
		return signedBytes, nil
	} else {
		return payloadBytes, nil
	}
}

func (ws *WebSocketWrapper) onMessage(m []byte) error {
	parsed := &message.Message{}
	if err := json.Unmarshal(m, parsed); err != nil {
		return err
	}
	if parsed.Packed {
		packed := &signer.SignedData{}
		if err := json.Unmarshal(parsed.EncodedData, packed); err != nil {
			return err
		}
		unpacked, err := signer.Unpack(packed, ws.ID)
		if err != nil {
			return err
		}
		parsed.EncodedData = []byte(unpacked.Data)
	}
	if err := json.Unmarshal(parsed.EncodedData, &parsed.Data); err != nil {
		return err
	}
	switch parsed.Data.Action {
	case "p":

	}

	return nil
}

func (ws *WebSocketWrapper) handleChannels() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ws.DoneChannel:
			return
		case m := <-ws.MessageOutChannel:
			bytes, err := json.Marshal(m.Data)
			if err != nil {
				log.Println("ws encode error:", err.Error())
			}
			if m.Packed {
				packed, err := signer.Pack(string(bytes), ws.ID.PrivateKey)
				if err != nil {
					log.Println("ws pack error:", err.Error())
				}
				packedBytes, err := json.Marshal(packed)
				if err != nil {
					log.Println("ws encode error:", err.Error())
				}
				bytes = packedBytes
			}
			m.EncodedData = bytes
			if err := ws.Socket.WriteJSON(m); err != nil {
				log.Println("ws write error:", err.Error())
				return
			}
		case <-ws.InterruptChannel:
			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := ws.Socket.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("ws interrupt error:", err.Error())
				return
			}
			select {
			case <-ws.DoneChannel:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
