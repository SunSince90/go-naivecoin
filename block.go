package main

import (
	"bytes"
	"crypto/sha256"
	"time"
)

// Block is a block that will be mined/forged and inserted into our Blockchain.
type Block struct {
	// Index of this block.
	Index int64
	// Timestamp of when this block was created.
	Timestamp int64
	// PreviousBlockHash is the hash of the previous block, i.e. the block
	// with an Index = thisIndex - 1.
	PreviousBlockHash []byte
	// Data that we want to store in this block.
	// TODO: on future commits, we may want to store *encrypted* data here.
	Data string
	// Hash of this block.
	Hash []byte
}

// NewBlock creates a new block with the given parameters, creates a hash
// of its data and returns it.
func NewBlock(data string, prevBlock Block) *Block {
	b := Block{
		Index:             prevBlock.Index + 1,
		Timestamp:         time.Now().Unix(),
		PreviousBlockHash: prevBlock.Hash,
		Data:              data,
	}

	b.Hash = CalculateHash(b)
	return &b
}

// CalculateHash calculates and returns the sha256 of the provided block.
func CalculateHash(block Block) []byte {
	header := bytes.Join([][]byte{
		FromInt64ToBytes(block.Index),
		FromInt64ToBytes(block.Timestamp),
		block.PreviousBlockHash,
		[]byte(block.Data),
	}, []byte{})

	hash := sha256.Sum256(header)
	return hash[:]
}
