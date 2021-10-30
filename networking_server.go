package main

import (
	"context"

	npb "github.com/SunSince90/go-naivecoin/pkg/networking/pb"
)

type CommunicationServer struct {
	genBlock chan Block
	npb.UnimplementedPeerCommunicationServer
}

func NewCommunicationServer(genBlock chan Block) *CommunicationServer {
	return &CommunicationServer{
		genBlock: genBlock,
	}
}

func (c *CommunicationServer) GetLatestBlock(ctx context.Context, _ *npb.GetLatestBlockParams) (*npb.Block, error) {
	b := blockchain.GetLastBlock()

	return &npb.Block{
		Index:             b.Index,
		Timestamp:         b.Timestamp,
		PreviousBlockHash: b.PreviousBlockHash,
		Data:              b.Data,
		Hash:              b.Hash,
	}, nil
}

func (c *CommunicationServer) GetFullBlockChain(ctx context.Context, _ *npb.GetFullBlockChainParams) (*npb.BlockChain, error) {
	// TODO: this is not very efficient, it should be either changed to
	// BlockChain (from main) rather than *npb.BlockChain or just accept this
	// as it is, as this does not (should not, actually...) happen frequently
	// TODO: this may need to be guarded by a lock
	bc := &npb.BlockChain{
		Blocks: make([]*npb.Block, len(blockchain.chain)),
	}

	for i, block := range blockchain.chain {
		bc.Blocks[i] = ToGrpcBlock(&block)
	}

	return bc, nil
}

func (c *CommunicationServer) SubscribeNewBlocks(_ *npb.SubscribeNewBlocksParams, commStream npb.PeerCommunication_SubscribeNewBlocksServer) error {
	for block := range c.genBlock {
		err := commStream.SendMsg(ToGrpcBlock(&block))
		if err != nil {
			log.Err(err).Msg("could not send block to peer")
		}
	}

	return nil
}

func ToGrpcBlock(block *Block) *npb.Block {
	return &npb.Block{
		Index:             block.Index,
		Timestamp:         block.Timestamp,
		PreviousBlockHash: block.PreviousBlockHash,
		Data:              block.Data,
		Hash:              block.Hash,
	}
}

func ToBlock(block *npb.Block) *Block {
	return &Block{
		Index:             block.Index,
		Timestamp:         block.Timestamp,
		PreviousBlockHash: block.PreviousBlockHash,
		Data:              block.Data,
		Hash:              block.Hash,
	}
}
