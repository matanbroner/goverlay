package message

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
}

type Message struct {
	EncodedData []byte `json:"data"`
	Packed      bool   `json:"packed"`
	Data        MessageData
}
