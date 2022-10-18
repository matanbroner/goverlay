package ws

type WebSocketConfig struct {
	Reconnect bool
}

type WebSocketMessage struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
}
