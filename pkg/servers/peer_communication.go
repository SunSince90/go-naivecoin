package servers

import (
	"context"

	"github.com/SunSince90/go-naivecoin/pkg/block"
	"github.com/SunSince90/go-naivecoin/pkg/pb"
	"github.com/rs/zerolog/log"
)

// PeerCommunicationServer is used to make pods communicate with each other
// and exchange information that should not be public but only useful for
// internal purposes, like synchronization and receive notifications of new
// blocks.
//
// This server should be used with gRPC.
type PeerCommunicationServer struct {
	genBlock   chan *pb.Block
	blockchain *block.BlockChain
	pb.UnimplementedPeerCommunicationServer
}

// NewPeerCommunicationServer creates and returns a new instance of the
// PeerCommunicationServer.
func NewPeerCommunicationServer(blockchain *block.BlockChain, genBlock chan *pb.Block) *PeerCommunicationServer {
	return &PeerCommunicationServer{
		genBlock:   genBlock,
		blockchain: blockchain,
	}
}

// GetLatestBlock returns the latest block that the node has in store.
func (c *PeerCommunicationServer) GetLatestBlock(ctx context.Context, _ *pb.GetLatestBlockParams) (*pb.Block, error) {
	block := c.blockchain.GetLastBlock()

	return block, nil
}

// GetFullBlockChain returns the full blockchain from the node.
//
// TODO: The difference with this and the public /blocks should be that this
// should also return unconfirmed blocks, if I am going to implement that.
func (c *PeerCommunicationServer) GetFullBlockChain(ctx context.Context, _ *pb.GetFullBlockChainParams) (*pb.BlockChain, error) {
	chain := c.blockchain.GetChain()

	return &pb.BlockChain{
		Blocks: chain,
	}, nil
}

// SubscribeNewBlocks establishes a server stream to get notifications when
// the peer generated a new block.
func (c *PeerCommunicationServer) SubscribeNewBlocks(_ *pb.SubscribeNewBlocksParams, commStream pb.PeerCommunication_SubscribeNewBlocksServer) error {
	for block := range c.genBlock {
		if err := commStream.SendMsg(block); err != nil {
			if commStream.Context().Err() == context.DeadlineExceeded ||
				commStream.Context().Err() == context.Canceled {
				break
			}

			log.Err(err).Msg("there is block to send")
			return err
		}
	}

	return nil
}
