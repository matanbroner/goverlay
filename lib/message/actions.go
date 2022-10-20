package message

const Connect = "connect"
const Reconnect = "reconnect"
const Disconnect = "disconnect"
const Confirm = "confirm"
const GetBlock = "get-block"
const MarkUsedByPeer = "mark-used-by-peer"
const MarkUnusedByPeer = "mark-unused-by-peer"
const OverlayMessage = "overlay-message"

// DHT Actions
const DHTPut = "dht-put"
const DHTPutAck = "dht-put-ack"
const DHTGet = "dht-get"
const DHTGot = "dht-got"
