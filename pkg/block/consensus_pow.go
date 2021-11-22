package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/SunSince90/go-naivecoin/pkg/pb"
	"github.com/rs/zerolog/log"
)

// ProofOfWorkSettings defines settings for the Proof of Work consensus.
type ProofOfWorkSettings struct {
	// InitialDifficulty is the difficulty set when starting the program.
	InitialDifficulty int `yaml:"initialDifficulty"`
	// BlockGenerationInterval defines how many seconds should the algorithm
	// mine new blocks.
	BlockGenerationInterval int `yaml:"blockGenerationInterval"`
	// DifficultyAdjustmentInterval defines how many blocks should the
	// mining difficulty should be re-adjusted.
	DifficultyAdjustmentInterval int `yaml:"difficultyAdjustmentInterval"`
}

// ProofOfWork implements the Proof of Work consensus.
type ProofOfWork struct {
	difficulty  int
	blockGenInt int
	diffAdjInt  int
}

// NewProofOfWork creates a new Proof of Work consensus implementation and
// returns to the caller. This should be stored inside a block factory.
func NewProofOfWork(settings *ProofOfWorkSettings) *ProofOfWork {
	blockGenInt := func() int {
		if settings != nil && settings.BlockGenerationInterval >= 0 {
			return settings.BlockGenerationInterval
		}

		// default value
		return 10
	}()
	diffAdjInt := func() int {
		if settings != nil && settings.DifficultyAdjustmentInterval >= 0 {
			return settings.DifficultyAdjustmentInterval
		}

		// default value
		return 10
	}()
	difficulty := func() int {
		if settings != nil && settings.InitialDifficulty >= 0 {
			return settings.InitialDifficulty
		}

		// default value
		return 3
	}()

	return &ProofOfWork{
		difficulty:  difficulty,
		blockGenInt: blockGenInt,
		diffAdjInt:  diffAdjInt,
	}
}

func (p *ProofOfWork) calculateHash(block *pb.Block) (int64, int64, []byte) {
	target := big.NewInt(1)
	targetBits := p.difficulty * 4 // remember that it's hexadecimal representation

	target.Lsh(target, uint(256-targetBits))
	var nonce int64 = 0
	var hash [32]byte

	for nonce < math.MaxInt64 {
		data := p.prepareData(block, nonce)
		hash = sha256.Sum256(data)

		if big.NewInt(0).SetBytes(hash[:]).Cmp(target) == -1 {
			break
		}

		nonce++
	}

	return int64(p.difficulty), int64(nonce), hash[:]
}

func (p *ProofOfWork) prepareData(block *pb.Block, nonce int64) []byte {
	data := bytes.Join(
		[][]byte{
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
			func() []byte {
				bytesVal := make([]byte, 8)
				binary.LittleEndian.PutUint64(bytesVal, uint64(p.difficulty))
				return bytesVal
			}(),
			func() []byte {
				bytesVal := make([]byte, 8)
				binary.LittleEndian.PutUint64(bytesVal, uint64(nonce))
				return bytesVal
			}(),
		},
		[]byte{},
	)

	return data
}

func (p *ProofOfWork) validateBlockHash(block *pb.Block) error {
	target := big.NewInt(1)
	targetBits := p.difficulty * 4

	target.Lsh(target, uint(256-targetBits))

	data := p.prepareData(block, block.Nonce)
	hash := sha256.Sum256(data)

	if big.NewInt(0).SetBytes(hash[:]).Cmp(target) != -1 {
		return fmt.Errorf("hash is not valid")
	}

	return nil
}

func (p *ProofOfWork) validateBlockTimestamps(newBlock, prevBlock *pb.Block) error {
	now := time.Now().Unix()

	if newBlock.Timestamp > now+60 /*|| prevBlock.Timestamp < prevBlock.Timestamp+60*/ {
		return fmt.Errorf("timestamp is not valid")
	}

	return nil
}

func (p *ProofOfWork) validateChain(chain []*pb.Block) (*big.Int, error) {
	genesis := newGenesisBlock()
	if chain[0].Index != 0 ||
		chain[0].Timestamp != 0 ||
		len(chain[0].PreviousBlockHash) > 0 ||
		chain[0].Data != genesisBlockData ||
		!bytes.Equal(genesis.Hash, chain[0].Hash) {
		return nil, fmt.Errorf("genesis block is wrong")
	}

	cumulativeDifficulty := big.NewInt(0)
	for i := 1; i < len(chain); i++ {
		if err := validateBlock(chain[i], chain[i-1]); err != nil {
			return nil, err
		}
		if err := p.validateBlockHash(chain[i]); err != nil {
			return nil, err
		}
		if err := p.validateBlockTimestamps(chain[i], chain[i-1]); err != nil {
			return nil, err
		}

		exp := big.NewInt(0).Exp(big.NewInt(2), big.NewInt(chain[i].Difficulty), nil)
		cumulativeDifficulty = big.NewInt(0).Add(cumulativeDifficulty, exp)
	}

	return cumulativeDifficulty, nil

}

func (p *ProofOfWork) adjustDifficulty(chain []*pb.Block) {
	prevAdjBlock := chain[len(chain)-p.blockGenInt]
	expectedTime := p.blockGenInt * p.diffAdjInt
	lastBlock := chain[len(chain)-1]

	switch diff := lastBlock.Timestamp - prevAdjBlock.Timestamp; {
	case diff < int64(expectedTime)/2:
		log.Info().Msg("incrementing difficulty by one")
		p.difficulty++
	case diff > int64(expectedTime)*2:
		log.Info().Msg("decreasing difficulty by one")
		p.difficulty--
	}
}
