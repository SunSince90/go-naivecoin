package main

import (
	"sync"
)

// Peer is a representation of other nodes.
type Peer struct {
	// Name of the peer.
	Name string
	// IP of the peer
	IP string

	// Address
}

// PeerEventType represents a type of event that could occur for a peer, i.e. when
// a new peer is found or when it is dead.
type PeerEventType string

const (
	// EventNewPeer represents an event about a new peer found.
	EventNewPeer PeerEventType = "NEW_PEER"
	// EventDeadPeer represents an event about a dead/dying peer.
	EventDeadPeer PeerEventType = "DEAD_PEER"

	// We don't really need an UPDATED_PEER as in this example project we are
	// running stateless loads on Kubernetes, thus they never change address
	// or name: when they die K8s will schedule a new pod with a new name and
	// new address.
	// This needs to be corrected in case the project is made to be
	// stateful.
)

// PeerEvent is a structure that is delivered to the channel.
type PeerEvent struct {
	// EventType is the type of the event occurring for this peer.
	EventType PeerEventType
	// Peer that caused this event.
	Peer *Peer
}

// PeersManager manages peers and peer events.
type PeersManager struct {
	peers map[string]*Peer

	lock sync.Mutex
}

// NewPeersManager creates and returns a new instance of a PeersManager.
func NewPeersManager() *PeersManager {
	return &PeersManager{
		peers: map[string]*Peer{},
		lock:  sync.Mutex{},
	}
}

// ListenPeerEvents runs loop on the provided channel to wait for events
// about peers.
func (m *PeersManager) ListenPeerEvents(peerEvents chan PeerEvent) {
	for ev := range peerEvents {
		m.lock.Lock()

		switch ev.EventType {
		case EventNewPeer:
			m.peers[ev.Peer.Name] = ev.Peer
		case EventDeadPeer:
			delete(m.peers, ev.Peer.Name)
		}

		m.lock.Unlock()
	}
}
