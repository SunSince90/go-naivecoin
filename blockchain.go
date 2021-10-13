package main

import (
	"fmt"
	"time"
)

const (
	genesisBlockData string = "this is the genesis block!"
)

// BlockChain simply contains a slice of Block.
//
// TODO: explore persistency on future commits.
// TODO: make this a singleton?
type BlockChain struct {
	chain []Block
}

// NewBlockChain returns a new block chain.
func NewBlockChain() *BlockChain {
	genesis := Block{
		Index:             0,
		Timestamp:         time.Now().Unix(),
		PreviousBlockHash: []byte{},
		Data:              genesisBlockData,
		Hash:              []byte{},
	}
	genesis.Hash = CalculateHash(genesis)

	return &BlockChain{
		chain: []Block{genesis},
	}
}

// PushBlock pushes the provided block to the blockchain.
//
// TODO: pass a Block or a *Block?
func (b *BlockChain) PushBlock(block Block) error {
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

func (b *BlockChain) ReplaceWith(newChain []Block) error {
	// ValidateChain may take a while, so we better check the lens first
	if len(newChain) <= len(b.chain) {
		return fmt.Errorf("new chain is not longer than the current one")
	}

	if err := ValidateChain(newChain); err != nil {
		return err
	}

	b.chain = newChain
	return nil
}
