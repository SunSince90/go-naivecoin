package main

import "time"

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
