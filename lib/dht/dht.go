package dht

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/matanbroner/goverlay/lib/message"
	"github.com/matanbroner/goverlay/lib/overlay"
)

type DHT struct {
	Overlay        *overlay.Overlay
	Data           map[string]map[string]string
	Callbacks      map[string]func(map[string]string)
	MessageCounter int
}

type OverlayListener struct {
	DHT *DHT
}

// DHT Methods

func New(overlay *overlay.Overlay) *DHT {
	d := &DHT{
		Overlay:        overlay,
		Data:           make(map[string]map[string]string),
		Callbacks:      make(map[string]func(map[string]string)),
		MessageCounter: 0,
	}
	overlay.AddListener(NewOverlayListener(d))
	return d
}

func (d *DHT) Get(key string, cb func(map[string]string)) {
	hashed := d.HashKey(key)
	if d.Overlay.InFloodRange(hashed) {
		if submap, ok := d.Data[key]; !ok {
			cb(submap)
		} else {
			cb(nil)
		}
	} else {
		msgId := uuid.New().String()
		d.Callbacks[msgId] = cb
		d.Overlay.SendToClosest(&message.Message{
			Data: message.MessageData{
				Action: message.DHTGet,
				To:     hashed,
			},
			ID: msgId,
		})
	}
}

func (d *DHT) Put(key string, value string, id string, cb func(map[string]string)) {
	hashed := d.HashKey(key)
	d.MessageCounter += 1
	if d.Overlay.InFloodRange(hashed) {
		if _, ok := d.Data[key]; !ok {
			d.Data[key] = make(map[string]string)
		}
		d.Data[key][id] = value
		cb(nil)
	} else {
		msgId := uuid.New().String()
		d.Callbacks[msgId] = cb
		d.Overlay.SendToClosest(&message.Message{
			Data: message.MessageData{
				Action: message.DHTPut,
				Value:  []byte(value),
				To:     hashed,
			},
			ID: msgId,
		})
	}
}

func (d *DHT) HashKey(key string) string {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

// OverlayMessageListener Methods

func NewOverlayListener(d *DHT) *OverlayListener {
	return &OverlayListener{
		DHT: d,
	}
}

func (oml *OverlayListener) OnMessage(m *message.Message) {
	switch m.Data.Action {
	case message.DHTPut:
		{
			submap := oml.DHT.Data[m.Data.To]
			if submap == nil {
				submap := make(map[string]string)
				oml.DHT.Data[m.Data.To] = submap
			}
			submap[m.Data.From] = string(m.Data.Value)
			if err := oml.DHT.Overlay.SendMessage(&message.Message{
				Data: message.MessageData{
					Action: message.DHTPutAck,
				},
				AckID: m.ID,
			}); err != nil {
				fmt.Printf("dht send message error: %s\n", err.Error())
			}
		}
	case message.DHTPutAck:
		{
			if cb, ok := oml.DHT.Callbacks[m.AckID]; ok {
				cb(nil)
			}
		}
	case message.DHTGet:
		{
			submap := oml.DHT.Data[m.Data.To]
			if submap == nil {
				submap = make(map[string]string)
			}
			bytes, err := json.Marshal(submap)
			if err != nil {
				fmt.Printf("dht marhsal value error: %s\n", err.Error())
			}
			if err := oml.DHT.Overlay.SendMessage(&message.Message{
				Data: message.MessageData{
					To:     m.Data.From,
					Action: message.DHTGot,
					Value:  bytes,
				},
				AckID: m.ID,
			}); err != nil {
				fmt.Printf("dht send message error: %s\n", err.Error())
			}
		}
	case message.DHTGot:
		{
			if cb, ok := oml.DHT.Callbacks[m.AckID]; ok {
				submap := map[string]string{}
				if err := json.Unmarshal(m.Data.Value, &submap); err != nil {
					fmt.Printf("dht unmarhsal submap error: %s\n", err.Error())
				}
				cb(submap)
			}
		}
	}
}
