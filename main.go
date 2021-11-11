package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/SunSince90/go-naivecoin/pkg/block"
	"github.com/SunSince90/go-naivecoin/pkg/controllers"
	"github.com/SunSince90/go-naivecoin/pkg/pb"
	"github.com/SunSince90/go-naivecoin/pkg/peers"
	"github.com/SunSince90/go-naivecoin/pkg/servers"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

func main() {
	os.Exit(run())
}

func run() int {
	log := zerolog.New(os.Stderr).With().Timestamp().Logger()
	log.Info().Msg("starting...")

	myip := os.Getenv("IP")
	if myip == "" {
		log.Error().Msg("could not find ip from environment variables")
		return 1
	}

	// create channels
	peerEvents := make(chan *peers.PeerEvent, 100)
	genBlock := make(chan *pb.Block, 10)

	// create structures
	blockchain := block.NewBlockChain()
	publicServer := servers.NewPublicServer(blockchain, genBlock)
	probesServer := servers.NewProbesServer(blockchain)
	grpcServer := grpc.NewServer()
	peerManager := peers.NewPeersManager(blockchain)
	mgr, err := controllers.NewControllerManager()
	if err != nil {
		log.Err(err).Msg("error while creating a controller manager")
		return 2
	}
	_, err = controllers.NewPodReconciler(mgr, peerEvents)
	if err != nil {
		log.Err(err).Msg("error while creating the pod controller")
		return 3
	}

	// run the services
	ctx, canc := context.WithCancel(context.Background())
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
	wg := sync.WaitGroup{}
	wg.Add(5)

	go func() {
		defer wg.Done()
		log.Info().Msg("listening for peer events...")
		peerManager.ListenPeerEvents(peerEvents)
	}()

	go func() {
		defer wg.Done()
		if err := publicServer.FiberApp.Listen(":8080"); err != nil {
			log.Err(err).Msg("error while serving public server")
		}

		close(genBlock)
	}()

	go func() {
		defer wg.Done()
		if err := probesServer.FiberApp.Listen(":8081"); err != nil {
			log.Err(err).Msg("error while serving public server")
		}
	}()

	go func() {
		defer wg.Done()
		lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", myip, 8082))
		if err != nil {
			log.Err(err).Msg("could not start communication server")
			return
		}
		log.Info().Msg("serving peer communications...")

		commServer := servers.NewPeerCommunicationServer(blockchain, genBlock)
		pb.RegisterPeerCommunicationServer(grpcServer, commServer)
		if err := grpcServer.Serve(lis); err != nil {
			log.Err(err).Msg("could not serve communication server")
			return
		}
	}()

	go func() {
		defer wg.Done()
		log.Info().Msg("starting pod controller...")

		if err := mgr.Start(ctx); err != nil {
			log.Err(err).Msg("error while starting controller manager")
		}

		close(peerEvents)
	}()

	<-stopChan
	canc()

	fmt.Println()
	log.Info().Msg("exit requested")

	log.Info().Msg("shutting down public server...")

	if err := publicServer.FiberApp.Shutdown(); err != nil {
		log.Err(err).Msg("error while shutting down public server")
	}

	log.Info().Msg("shutting down probes server...")
	if err := probesServer.FiberApp.Shutdown(); err != nil {
		log.Err(err).Msg("error while shutting down probes server")
	}

	log.Info().Msg("shutting down peers server...")
	grpcServer.GracefulStop()

	wg.Wait()
	log.Info().Msg("clean up done, goodbye!")
	return 0
}
