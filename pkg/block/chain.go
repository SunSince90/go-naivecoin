package block

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"

	"github.com/SunSince90/go-naivecoin/pkg/pb"
	"github.com/rs/zerolog/log"
)

// BlockChain manages a slice of Blocks.
//
// TODO: explore persistency on future commits.
type BlockChain struct {
	chain                []*pb.Block
	pow                  *ProofOfWork
	cumulativeDifficulty *big.Int
	lock                 sync.Mutex
}

// PushBlock validates the the provided block and -- if successful -- pushes it
// to the blockchain.
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

	if b.pow != nil {
		if err := b.pow.validateBlockHash(block); err != nil {
			return err
		}
		if err := b.pow.validateBlockTimestamps(block, lastBlock); err != nil {
			return err
		}
	}

	b.chain = append(b.chain, block)

	if b.pow != nil {
		exp := big.NewInt(0).Exp(big.NewInt(2), big.NewInt(block.Difficulty), nil)
		b.cumulativeDifficulty = b.cumulativeDifficulty.Add(b.cumulativeDifficulty, exp)

		if block.Index%int64(b.pow.blockGenInt) == 0 {
			b.pow.adjustDifficulty(b.chain)
		}
	}

	return nil
}

// ReplaceWith validates the given chain and replaces the one stored inside
// the blockchain with the the one in the parameter.
func (b *BlockChain) ReplaceWith(newChain []*pb.Block) error {

	b.lock.Lock()
	defer b.lock.Unlock()

	if b.pow != nil {
		cdiff, err := b.pow.validateChain(newChain)
		if err != nil {
			return err
		}

		switch b.cumulativeDifficulty.Cmp(cdiff) {
		case 0:
			log.Info().Msg("peer's chain is valid and has the same cumulative difficulty as mine: stopping here")
			return nil
		case 1:
			// This should actually never happen, but let's cover this case anyways
			return fmt.Errorf("peer's cumulative difficulty is lower than mine, stopping here")
		default: // case -1
			b.chain = newChain
			log.Info().Msg("chain replaced with my peer's chain")
			return nil
		}
	}

	// For non proof of work
	// ValidateChain may take a while, so we better check the len-s first
	if len(newChain) < len(b.chain) {
		return fmt.Errorf("new chain is not longer than the current one")
	}

	if err := validateChain(newChain); err != nil {
		return err
	}

	if len(newChain) == len(b.chain) {
		log.Info().Msg("peer chain is valid and same length as mine, stopping here...")
		return nil
	}

	log.Info().Msg("chain replaced with my peer's chain")
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
