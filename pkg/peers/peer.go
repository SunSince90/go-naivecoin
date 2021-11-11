package peers

import (
	"context"
	"fmt"

	"github.com/SunSince90/go-naivecoin/pkg/block"
	"github.com/SunSince90/go-naivecoin/pkg/pb"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

// Peer is a representation of other nodes.
type Peer struct {
	// Name of the peer.
	Name string
	// IP of the peer
	IP string

	// Address
	sub pb.PeerCommunication_SubscribeNewBlocksClient
	// TODO: check this
	CancelContext context.CancelFunc
}

// GetLastBlock returns the last block that the peer has stored.
func (p *Peer) GetLastBlock(ctx context.Context) (*pb.Block, error) {
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", p.IP, 8082), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	cli := pb.NewPeerCommunicationClient(conn)

	return cli.GetLatestBlock(ctx, &pb.GetLatestBlockParams{})
}

// GetFullBlockChain returns the full chain from the peer.
func (p *Peer) GetFullBlockChain(ctx context.Context) ([]*pb.Block, error) {
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", p.IP, 8082), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	cli := pb.NewPeerCommunicationClient(conn)

	bc, err := cli.GetFullBlockChain(ctx, &pb.GetFullBlockChainParams{})
	if err != nil {
		return nil, err
	}

	return bc.Blocks, nil
}

// SubscribeBlockGeneration runs a uni-direction stream connection to the peer
// to get blocks generated by the peer.
//
// This needs to run in a separate goroutine.
func (p *Peer) SubscribeBlockGeneration(ctx context.Context, blockchain *block.BlockChain) error {
	l := log.With().
		Str("peer-name", p.Name).
		Str("peer-ip", p.IP).
		Logger()

	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", p.IP, 8082), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return err
	}
	defer conn.Close()

	cli := pb.NewPeerCommunicationClient(conn)

	subCtx, canc := context.WithCancel(ctx)
	sub, err := cli.SubscribeNewBlocks(subCtx, &pb.SubscribeNewBlocksParams{})
	if err != nil {
		canc()
		return err
	}

	// store the cancel function, that we can later use it to unsubscribe
	// from events
	p.CancelContext = canc

	l.Info().Msg("listening for block generation events from peer...")
	p.sub = sub
	for {
		block, err := sub.Recv()
		if err != nil {
			if sub.Context().Err() == context.DeadlineExceeded || sub.Context().Err() == context.Canceled {
				return nil
			}

			l.Err(err).Msg("error while receiving")
			return err
		}

		l.Info().Int64("index", block.Index).Str("data", block.Data).Msg("got block from peer")
		if err := blockchain.PushBlock(block); err != nil {
			l.Err(err).Msg("error while adding block to blockchain")
		} else {
			l.Info().Msg("added block generated by peer")
		}
	}
}