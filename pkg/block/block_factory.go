package block

import (
	"math/big"
	"sync"
	"time"

	"github.com/SunSince90/go-naivecoin/pkg/pb"
)

// BlockFactory is in charge of creating new blocks and blockchains according
// to some settings, e.g. the consensus method.
type BlockFactory struct {
	pow *ProofOfWork
}

// FactoryOptions defines options for the block factory.
type FactoryOptions func(*BlockFactory)

// WithProofOfWork instructs the blockfacotry to also initializes
// a proof of work.
func WithProofOfWork(settings *ProofOfWorkSettings) FactoryOptions {
	return func(bf *BlockFactory) {
		bf.pow = NewProofOfWork(settings)
	}
}

// NewBlockFactory initializes a new block factory with the provided settings
// and returns it to the caller so it can be used to create new blocks and
// blockchains.
func NewBlockFactory(options ...FactoryOptions) *BlockFactory {
	factory := &BlockFactory{}
	for _, o := range options {
		o(factory)
	}

	return factory
}

// TODO: WithProofOfStake

// NewBlock creates a new block with the provided data and returns it to the
// caller.
func (f *BlockFactory) NewBlock(data string, prevBlock *pb.Block) *pb.Block {
	b := &pb.Block{
		Index:             prevBlock.Index + 1,
		Timestamp:         time.Now().Unix(),
		PreviousBlockHash: prevBlock.Hash,
		Data:              data,
	}

	if f.pow != nil {
		diff, nonce, hash := f.pow.calculateHash(b)

		b.Difficulty = diff
		b.Nonce = nonce
		b.Hash = hash
	} else {
		b.Hash = calculateHash(b)
	}

	return b
}

// NewBlockChain creates a new BlockChain and returns it to the caller.
func (f *BlockFactory) NewBlockChain() *BlockChain {
	genesis := newGenesisBlock()

	bc := &BlockChain{
		chain:                []*pb.Block{genesis},
		lock:                 sync.Mutex{},
		cumulativeDifficulty: big.NewInt(0),
	}

	if f.pow != nil {
		bc.pow = f.pow
	}

	return bc
}
