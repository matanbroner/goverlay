package cleaner

import (
	"fmt"
	"github.com/matanbroner/goverlay/lib/id"
	"github.com/matanbroner/goverlay/lib/message"
	"github.com/matanbroner/goverlay/lib/overlay"
	"github.com/matanbroner/goverlay/lib/util"
	"github.com/matanbroner/goverlay/lib/wrtc"
	"github.com/pion/webrtc/v3"
	"math/big"
	"strconv"
	"time"
)

const CleanUpSeconds = 5
const ConnectionTimeoutSeconds = 30

type NetworkCleaner struct {
	Overlay        *overlay.Overlay
	QueryTime      int
	CleanupChannel chan struct{}
}

func NewNetworkCleaner(o *overlay.Overlay) *NetworkCleaner {
	n := &NetworkCleaner{
		Overlay:   o,
		QueryTime: 0,
	}
	return n
}

func (n *NetworkCleaner) Start() {
	n.SetCleanupInterval()
}

func (n *NetworkCleaner) Stop() {
	n.ClearCleanupInterval()
}

func (n *NetworkCleaner) Clean() {
	for _, msg := range n.Overlay.PendingMessages {
		msg.Data.Proxies = nil
		n.Overlay.Proxy(msg)
	}
	n.Overlay.PendingMessages = nil
	if !n.Overlay.Status.IsSubordinate {
		//this.checkFloodAndFingers(true);
		//this.timeoutConnections();
	}
}

func (n *NetworkCleaner) SetCleanupInterval() {
	ticker := time.NewTicker(CleanUpSeconds * time.Second)
	n.CleanupChannel = make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				n.Clean()
			case <-n.CleanupChannel:
				ticker.Stop()
				return
			}
		}
	}()
}

func (n *NetworkCleaner) ClearCleanupInterval() {
	if n.CleanupChannel != nil {
		close(n.CleanupChannel)
	}
	n.CleanupChannel = nil
}

func (n *NetworkCleaner) ExpireConnectionIfPending(conn *wrtc.WebRTCConnection) {
	time.AfterFunc(ConnectionTimeoutSeconds*time.Second, func() {
		if conn.IsPending() {
			if err := n.Overlay.WebRTCWrapper.Disconnect(conn); err != nil {
				fmt.Printf("cleaner expire connection err: %s\n", err.Error())
			}
		}
	})
}

func (n *NetworkCleaner) TimeoutConnections() {
	for _, conn := range n.Overlay.WebRTCWrapper.Connections {
		if conn.IsPending() {
			n.ExpireConnectionIfPending(conn)
		} else if conn.PeerConnection.ICEConnectionState() == webrtc.ICEConnectionStateClosed {
			if err := n.Overlay.WebRTCWrapper.Disconnect(conn); err != nil {
				fmt.Printf("cleaner expire connection err: %s\n", err.Error())
			}
		}
	}
}

func (n *NetworkCleaner) FingerIsInactive(finger string) bool {
	return n.Overlay.WebRTCWrapper.GetConnection(finger, nil) == nil
}

func (n *NetworkCleaner) CheckFloodAndFingers(retry bool) error {
	golden := n.Overlay.GoldenIDs()
	flood := n.Overlay.Flood
	fingers := n.Overlay.Fingers
	// only worry about fingers if flood is full
	if len(flood) == n.Overlay.MaxFloodSize {
		distance := id.DirectedDistanceBetweenIDs(flood[0], flood[len(flood)-1])
		var movingDistance *big.Int
		movingDistance.Rsh(id.HalfMax, uint(len(fingers)))
		if distance.Cmp(movingDistance) < 0 {
			n.Overlay.Fingers = append(fingers, id.PendingID)
			level := len(n.Overlay.Fingers) - 1
			n.Overlay.SendToClosest(&message.Message{
				Data: message.MessageData{
					Action: message.FindFinger,
					To:     id.IdealFinger(n.Overlay.ID.ID, level),
					Value:  []byte(strconv.Itoa(level)),
				},
			})
		}
		if len(fingers) > 0 && distance.Cmp(movingDistance) > 0 {
			// need to remove a finger
			util.Pop(n.Overlay.Fingers)
		}
		if retry {
			inactive := util.Filter(n.Overlay.Fingers, n.FingerIsInactive)
			for idx := range inactive {
				n.Overlay.Fingers[idx] = id.PendingID
				n.Overlay.SendToClosest(&message.Message{
					Data: message.MessageData{
						Action: message.FindFinger,
						To:     id.IdealFinger(n.Overlay.ID.ID, idx),
						Value:  []byte(strconv.Itoa(idx)),
					},
				})
			}
		}
	} else {
		// if not at full capacity, no need for fingers
		n.Overlay.Fingers = nil
	}
	var leftOverConnections []string
	for _, gid := range golden {
		if n.Overlay.InFlood(gid) {
			if err := n.Overlay.WebRTCWrapper.MarkUsed(gid); err != nil {
				return err
			}
		} else {
			if util.Contains(n.Overlay.Fingers, gid) {
				if err := n.Overlay.WebRTCWrapper.MarkUsed(gid); err != nil {
					return err
				}
			} else {
				conn := n.Overlay.WebRTCWrapper.ConnectionsMap[gid]
				if time.Now().Sub(conn.LastUsed) > 60*time.Second {
					leftOverConnections = append(leftOverConnections, gid)
				} else {
					if err := n.Overlay.WebRTCWrapper.MarkUsed(gid); err != nil {
						return err
					}
				}
			}
		}
	}
	if len(n.Overlay.Flood) == n.Overlay.MaxFloodSize {
		for _, gid := range leftOverConnections {
			if err := n.Overlay.WebRTCWrapper.MarkUnused(gid); err != nil {
				return err
			}
		}
	}
	return nil
}
