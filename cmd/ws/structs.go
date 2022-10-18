package ws

type WebSocketConfig struct {
	Reconnect bool
}

type WebSocketMessage struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}
