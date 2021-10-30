package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	npb "github.com/SunSince90/go-naivecoin/pkg/networking/pb"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

var (
	log        zerolog.Logger
	blockchain *BlockChain
)

func main() {
	log = zerolog.New(os.Stderr).With().Timestamp().Logger()

	myself := os.Getenv("NAME")
	myIP := os.Getenv("IP")
	ns := os.Getenv("NAMESPACE")
	if ns == "" || myself == "" || myIP == "" {
		log.Error().Msg("could not find environment variables")
		os.Exit(1)
	}

	log.Info().Str("my-name", myself).Msg("starting...")

	peerEvents := make(chan PeerEvent, 100)
	genBlock := make(chan Block, 10)
	blockchain = NewBlockChain()
	server := NewNodeServer(genBlock)
	probes := NewProbesServer()
	peersMgr := NewPeersManager(myself, myIP)
	commServer := NewCommunicationServer(genBlock)

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
	wg := sync.WaitGroup{}
	wg.Add(5)

	go func() {
		defer wg.Done()
		log.Info().Msg("listening for peer events...")
		peersMgr.ListenPeerEvents(peerEvents)
	}()

	go func() {
		defer wg.Done()
		log.Info().Msg("serving server on port 8080...")
		if err := server.FiberApp.Listen(":8080"); err != nil {
			log.Err(err).Msg("error while listening")
			return
		}
	}()

	go func() {
		defer wg.Done()
		log.Info().Msg("serving probes on port 8081...")
		if err := probes.Listen(":8081"); err != nil {
			log.Err(err).Msg("error while listening for probes requests")
			return
		}
	}()

	go func() {
		defer wg.Done()
		lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", myIP, 8082))
		if err != nil {
			log.Err(err).Msg("could not start communication server")
			return
		}
		s := grpc.NewServer()
		npb.RegisterPeerCommunicationServer(s, commServer)
		log.Info().Msg("serving peer communication server on port 8082...")
		if err := s.Serve(lis); err != nil {
			log.Err(err).Msg("could not serve communication server")
			return
		}
	}()

	// -- start the manager for our pod controller
	mgrCtx, mgrCanc := context.WithCancel(context.Background())
	go func() {
		defer wg.Done()

		mgr, err := GetControllerManager(ns)
		if err != nil {
			log.Err(err).Msg("error while creating controller manager")
			stopChan <- syscall.SIGINT
			return
		}

		podEventHandler := NewPodEventHandler(myself, peerEvents)
		_, err = SetPodController(mgr, podEventHandler)
		if err != nil {
			log.Err(err).Msg("error while creating pod controller")
			stopChan <- syscall.SIGINT
			return
		}

		// We're handling graceful shutdown on our own, so that's why we are
		// using our custom context and not just copy-pasting the example
		// from controller-runtime.
		if err := mgr.Start(mgrCtx); err != nil {
			log.Err(err).Msg("error while starting controller manager")
		}
	}()

	// -- graceful shutdown
	<-stopChan

	fmt.Println()
	log.Info().Msg("exit requested")
	log.Info().Msg("shutting down server...")

	mgrCanc()
	server.FiberApp.Shutdown()
	probes.Shutdown()
	close(peerEvents)
	close(genBlock)
	wg.Wait()

	log.Info().Msg("good bye!")
}
