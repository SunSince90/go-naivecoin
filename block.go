package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"time"
)

// Block is a block that will be mined/forged and inserted into our Blockchain.
type Block struct {
	// Index of this block.
	Index int64 `json:"index"`
	// Timestamp of when this block was created.
	Timestamp int64 `json:"timestamp"`
	// PreviousBlockHash is the hash of the previous block, i.e. the block
	// with an Index = thisIndex - 1.
	PreviousBlockHash []byte `json:"previousBlockHash"`
	// Data that we want to store in this block.
	// TODO: on future commits, we may want to store *encrypted* data here.
	Data string `json:"data"`
	// Hash of this block.
	Hash []byte `json:"hash"`
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

// ValidateBlock checks if the block is valid and returns an error if not.
func ValidateBlock(block, prevBlock Block) error {
	if block.Index != prevBlock.Index+1 {
		return fmt.Errorf("index is not valid")
	}

	if !bytes.Equal(block.PreviousBlockHash, prevBlock.Hash) {
		return fmt.Errorf("previous block hash does not match")
	}

	if !bytes.Equal(block.Hash, CalculateHash(block)) {
		return fmt.Errorf("hash is invalid")
	}

	return nil
}
