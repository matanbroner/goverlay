package ws

import (
	"encoding/json"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/gorilla/websocket"
	"github.com/matanbroner/goverlay/cmd/id"
	"github.com/matanbroner/goverlay/cmd/overlay"
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
	MessageOutChannel            chan WebSocketMessage
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
	return nil
}

func (ws *WebSocketWrapper) Disconnect() error {
	if ws.Socket != nil {
		if err := ws.Socket.Close(); err != nil {
			return err
		}
	}
	ws.DoneChannel = make(chan struct{})
	ws.MessageOutChannel = make(chan WebSocketMessage)
	ws.InterruptChannel = make(chan os.Signal, 1)
	signal.Notify(ws.InterruptChannel, os.Interrupt)

	go func() {
		defer close(ws.DoneChannel)
		for {
			_, message, err := ws.Socket.ReadMessage()
			if err != nil {
				log.Println("ws error:", err.Error())
				return
			}
			if err := ws.onMessage(message); err != nil {
				log.Println("ws error:", err.Error())
				return
			}
		}
	}()

	go ws.handleChannels()

	return nil
}

func (ws *WebSocketWrapper) onMessage(message []byte) error {
	parsed := &WebSocketMessage{}
	if err := json.Unmarshal(message, parsed); err != nil {
		return err
	}
	switch parsed.Type {
	case "PING":
		{
			ws.MessageOutChannel <- WebSocketMessage{Type: "PONG"}
		}
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
