package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/SunSince90/go-naivecoin/pkg/pb"
)

const (
	genesisBlockData string = "this is the genesis block!"
)

// calculateHash calculates and returns the sha256 of the provided block.
func calculateHash(block *pb.Block) []byte {
	header := bytes.Join([][]byte{
		func() []byte {
			bytesVal := make([]byte, 8)
			binary.LittleEndian.PutUint64(bytesVal, uint64(block.Index))
			return bytesVal
		}(),
		func() []byte {
			bytesVal := make([]byte, 8)
			binary.LittleEndian.PutUint64(bytesVal, uint64(block.Timestamp))
			return bytesVal
		}(),
		block.PreviousBlockHash,
		[]byte(block.Data),
	}, []byte{})

	hash := sha256.Sum256(header)
	return hash[:]
}

func newGenesisBlock() *pb.Block {
	genesis := &pb.Block{
		Index:             0,
		Timestamp:         0, // Setting this as 0 will prevent errors when we get blocks from other peers
		PreviousBlockHash: []byte{},
		Data:              genesisBlockData,
		Hash:              []byte{},
	}
	genesis.Hash = calculateHash(genesis)
	return genesis
}

// validateBlock checks if the block is valid and returns an error if not.
func validateBlock(block, prevBlock *pb.Block) error {
	if block.Index != prevBlock.Index+1 {
		return fmt.Errorf("index is not valid")
	}

	if !bytes.Equal(block.PreviousBlockHash, prevBlock.Hash) {
		return fmt.Errorf("previous block hash does not match")
	}

	return nil
}
