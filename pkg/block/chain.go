package block

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/SunSince90/go-naivecoin/pkg/pb"
	"github.com/rs/zerolog/log"
)

// BlockChain simply contains a slice of Block-s.
//
// TODO: explore persistency on future commits.
type BlockChain struct {
	chain []*pb.Block

	lock sync.Mutex
}

// NewBlockChain returns a new block chain.
func NewBlockChain() *BlockChain {
	genesis := newGenesisBlock()

	return &BlockChain{
		chain: []*pb.Block{genesis},
		lock:  sync.Mutex{},
	}
}

// PushBlock pushes the provided block to the blockchain.
func (b *BlockChain) PushBlock(block *pb.Block) error {
	if block == nil {
		return fmt.Errorf("block is nil")
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	// we lock here because we don't want the risk of getting the last block
	// and later add a block while some inserts it before us.

	lastBlock := b.chain[len(b.chain)-1]
	if err := validateBlock(block, lastBlock); err != nil {
		return err
	}

	b.chain = append(b.chain, block)

	return nil
}

// ReplaceWith replaces the chain with the one provided as argument.
func (b *BlockChain) ReplaceWith(newChain []*pb.Block) error {
	// ValidateChain may take a while, so we better check the len-s first
	if len(newChain) <= len(b.chain) {
		return fmt.Errorf("new chain is not longer than the current one")
	}

	if err := validateChain(newChain); err != nil {
		return err
	}

	b.lock.Lock()
	b.chain = newChain
	b.lock.Unlock()

	log.Info().Msg("chain replaced")

	return nil
}

// Length returns the current length of the chain.
// This is mostly used by Kubernetes probes to signal this pod as Ready, as
// this is also guarded by locks.
func (b *BlockChain) Length() int {
	b.lock.Lock()
	defer b.lock.Unlock()

	return len(b.chain)
}

// GetLastBlock returns the current last block on the chain.
func (b *BlockChain) GetLastBlock() *pb.Block {
	b.lock.Lock()
	defer b.lock.Unlock()

	return b.chain[len(b.chain)-1]
}

// GetChain returns the chain from the blockchain.
func (b *BlockChain) GetChain() []*pb.Block {
	b.lock.Lock()
	defer b.lock.Unlock()

	return b.chain
}

// validateChain checks if the provided chain is correct and returns an
// error if not.
func validateChain(chain []*pb.Block) error {
	genesis := newGenesisBlock()
	if chain[0].Index != 0 ||
		chain[0].Timestamp != 0 ||
		len(chain[0].PreviousBlockHash) > 0 ||
		chain[0].Data != genesisBlockData ||
		!bytes.Equal(genesis.Hash, chain[0].Hash) {
		return fmt.Errorf("genesis block is wrong")
	}

	for i := 1; i < len(chain); i++ {
		if err := validateBlock(chain[i], chain[i-1]); err != nil {
			return err
		}
	}

	return nil
}
