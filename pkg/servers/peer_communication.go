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
	blockchain  *block.BlockChain
	subscribers []chan *pb.Block
	pb.UnimplementedPeerCommunicationServer
}

// NewPeerCommunicationServer creates and returns a new instance of the
// PeerCommunicationServer.
func NewPeerCommunicationServer(blockchain *block.BlockChain) *PeerCommunicationServer {
	return &PeerCommunicationServer{
		blockchain:  blockchain,
		subscribers: []chan *pb.Block{},
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

// SubscribeNewBlocks *sends* data to peers that are subscribed to me.
// It is called SubscribeNewBlocks because that's what peers will do, but here
// we are on the serving side.
// TODO: update this name in future?
func (c *PeerCommunicationServer) SubscribeNewBlocks(_ *pb.SubscribeNewBlocksParams, commStream pb.PeerCommunication_SubscribeNewBlocksServer) error {
	// This can probably be written in a better way, but for now let's keep
	// this like this and see how we can improve this in future.
	myChan := make(chan *pb.Block, 10)
	c.subscribers = append(c.subscribers, myChan)

	for block := range myChan {
		if err := commStream.SendMsg(block); err != nil {
			if commStream.Context().Err() == context.DeadlineExceeded ||
				commStream.Context().Err() == context.Canceled {
				break
			}

			return err
		}
	}

	return nil
}

func (c *PeerCommunicationServer) ServeSubscriptions(genBlock chan *pb.Block) {
	for block := range genBlock {
		for _, sub := range c.subscribers {
			sub <- block
		}
	}

	log.Info().Msg("closing all subscriptions...")
	for _, sub := range c.subscribers {
		close(sub)
	}
	log.Info().Msg("all subscriptions closed")
}
