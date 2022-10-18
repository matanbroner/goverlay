package ws

import (
	"github.com/gorilla/websocket"
	"github.com/matanbroner/goverlay/cmd/id"
	"github.com/matanbroner/goverlay/cmd/overlay"
	"log"
	"net/http"
	"testing"
)

func init() {
	go startWsServer()
}

func startWsServer() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		u := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		}
		// upgrade this connection to a WebSocket
		_, err := u.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
		}
		if err != nil {
			log.Println(err)
		}
	})
	log.Fatal(http.ListenAndServe(":9999", nil))
}

func TestConnect(t *testing.T) {
	pkeyID, err := id.NewPublicKeyId(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	o := &overlay.Overlay{
		ID: pkeyID,
	}
	ws := WebSocketWrapper{
		Overlay: o,
		Host:    "ws://localhost:9999/ws",
	}
	if err := ws.Connect(nil); err != nil {
		t.Fatal(err)
	}
}
