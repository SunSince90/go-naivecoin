package peers

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/SunSince90/go-naivecoin/pkg/block"
	"github.com/rs/zerolog/log"
)

// PeersManager manages peers and peer events.
type PeersManager struct {
	peers      map[string]*Peer
	blockchain *block.BlockChain
	lock       sync.Mutex
}

func NewPeersManager(blockchain *block.BlockChain) *PeersManager {
	return &PeersManager{
		peers:      map[string]*Peer{},
		lock:       sync.Mutex{},
		blockchain: blockchain,
	}
}

func (m *PeersManager) addPeer(addCtx context.Context, peer *Peer) error {
	m.lock.Lock()
	_, exists := m.peers[peer.Name]
	m.lock.Unlock()

	if exists {
		return fmt.Errorf("peer already present")
	}

	ctx, canc := context.WithTimeout(addCtx, 30*time.Second)
	peerLastBlock, err := peer.GetLastBlock(ctx)
	if err != nil {
		canc()
		return fmt.Errorf("could not last block from peer")
	}
	canc()

	myLastBlock := m.blockchain.GetLastBlock()
	switch diff := myLastBlock.Index - peerLastBlock.Index; {
	case diff == 0:
		// we have the same index
		if !bytes.Equal(myLastBlock.Hash, peerLastBlock.Hash) {
			return fmt.Errorf("peer's last block hash is invalid")
		}

		if !bytes.Equal(myLastBlock.PreviousBlockHash, peerLastBlock.PreviousBlockHash) {
			return fmt.Errorf("peer's previous block hash is invalid")
		}
	case diff < 0:
		// the peer's last block is higher than mine
		ctx, canc := context.WithTimeout(addCtx, 30*time.Second)
		peerBlockChain, err := peer.GetFullBlockChain(ctx)
		if err != nil {
			canc()
			return fmt.Errorf("could not get full blockchain from peer")
		}
		canc()

		if err := m.blockchain.ReplaceWith(peerBlockChain); err != nil {
			return fmt.Errorf("could not replace chain with peer's chain")
		}
	case diff > 0:
		// the peer's last block is lower than mine. I don't need to sync.
	}

	m.lock.Lock()
	m.peers[peer.Name] = peer
	m.lock.Unlock()

	log.Info().Str("peer-name", peer.Name).Msg("added peer")

	return nil
}

func (m *PeersManager) removePeer(name string) (*Peer, error) {
	peer, exists := func() (*Peer, bool) {
		m.lock.Lock()
		defer m.lock.Unlock()

		peer, exists := m.peers[name]
		if !exists {
			return nil, false
		}

		delete(m.peers, name)
		return peer, true
	}()
	if !exists {
		return nil, fmt.Errorf("peer was not found")
	}

	return peer, nil
}

func (m *PeersManager) ListenPeerEvents(peerEvents chan *PeerEvent) {
	// This context will be passed to each addPeer and used as a main
	// context: when the user wants to stop the program they will also close
	// the channel, so this context will be cancelled with it.
	ctx, canc := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	for ev := range peerEvents {
		switch ev.EventType {

		case EventNewPeer:
			go func(peer *Peer) {
				if err := m.addPeer(ctx, peer); err != nil {
					log.Err(err).Str("peer-name", peer.Name).Msg("could not add peer")
					return
				}

				wg.Add(1)
				peer.SubscribeBlockGeneration(ctx, m.blockchain)
				wg.Done()
			}(ev.Peer)

		case EventDeadPeer:
			go func(peer *Peer) {
				foundPeer, err := m.removePeer(peer.Name)
				if err != nil {
					log.Err(err).Str("peer-name", peer.Name).Msg("could not remove peer")
					return
				}

				foundPeer.CancelContext()
			}(ev.Peer)
		}
	}

	// each goroutine subscribed to block generation events from other peers
	// receives a new context derived from the one we created above:
	// by cancelling the "main" one above, we cancel the other ones too.
	log.Info().Msg("unsubscribing from all peers...")
	canc()
	wg.Wait()
	log.Info().Msg("all unsubscriptions done")
}
