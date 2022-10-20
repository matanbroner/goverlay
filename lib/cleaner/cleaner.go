package cleaner

import (
	"fmt"
	"github.com/matanbroner/goverlay/lib/overlay"
	"github.com/matanbroner/goverlay/lib/wrtc"
	"github.com/pion/webrtc/v3"
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
	n.SetCleanupInterval()
	return n
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
