package peers

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
