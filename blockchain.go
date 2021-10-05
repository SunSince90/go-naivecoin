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
