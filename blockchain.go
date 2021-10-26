package main

import (
	"fmt"
	"sync"
)

const (
	genesisBlockData string = "this is the genesis block!"
)

// BlockChain simply contains a slice of Block-s.
//
// TODO: explore persistency on future commits.
// TODO: make this a singleton?
type BlockChain struct {
	chain []Block

	lock sync.Mutex
}

// NewBlockChain returns a new block chain.
func NewBlockChain() *BlockChain {
	genesis := Block{
		Index:             0,
		Timestamp:         0, // Setting this as 0 will prevent errors when we get blocks from other peers
		PreviousBlockHash: []byte{},
		Data:              genesisBlockData,
		Hash:              []byte{},
	}
	genesis.Hash = CalculateHash(genesis)

	return &BlockChain{
		chain: []Block{genesis},
		lock:  sync.Mutex{},
	}
}

// PushBlock pushes the provided block to the blockchain.
//
// TODO: pass a Block or a *Block?
func (b *BlockChain) PushBlock(block Block) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if err := ValidateBlock(block, b.chain[len(b.chain)-1]); err != nil {
		return err
	}

	// TODO: chain is unexported, so technically it shouldn't be exposed
	// outside, but here everything is under the same main package.
	// Move this to another pkg?
	b.chain = append(b.chain, block)
	return nil
}

// ValidateChain checks if the provided chain is correct and returns an
// error if not.
func ValidateChain(chain []Block) error {
	if chain[0].Data != genesisBlockData {
		// We are comparing the data inside of the genesis block.
		// The probability of hashes being the same with different data inside
		// is very very low, but I don't want to re-calculate the hash :)
		// TODO: make the genesis block a const?
		return fmt.Errorf("genesis block is wrong")
	}

	for i := 1; i < len(chain); i++ {
		if err := ValidateBlock(chain[i], chain[i-1]); err != nil {
			return err
		}
	}

	return nil
}

// ReplaceWith replaces the chain with the one provided as argument.
func (b *BlockChain) ReplaceWith(newChain []Block) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	// ValidateChain may take a while, so we better check the lens first
	if len(newChain) <= len(b.chain) {
		return fmt.Errorf("new chain is not longer than the current one")
	}

	if err := ValidateChain(newChain); err != nil {
		return err
	}

	b.chain = newChain
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
func (b *BlockChain) GetLastBlock() Block {
	b.lock.Lock()
	defer b.lock.Unlock()

	return b.chain[len(b.chain)-1]
}
