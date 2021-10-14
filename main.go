package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rs/zerolog"
)

var (
	log        zerolog.Logger
	blockchain *BlockChain
	peers      map[string]string
)

func main() {
	log = zerolog.New(os.Stderr).With().Timestamp().Logger()
	log.Info().Msg("starting...")

	myself := os.Getenv("NAME")
	ns := os.Getenv("NAMESPACE")
	if ns == "" || myself == "" {
		log.Error().Msg("could not find environment variables")
		os.Exit(1)
	}

	blockchain = NewBlockChain()
	server := NewServer()
	peers = make(map[string]string) // TODO: improve this

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
	wg := sync.WaitGroup{}
	wg.Add(2)

	// -- start the server
	go func() {
		defer wg.Done()
		err := server.Listen(":8080")
		if err != nil {
			log.Err(err).Msg("error while listening")
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

		_, err = SetPodController(mgr, myself)
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
	server.Shutdown()
	wg.Wait()

	log.Info().Msg("good bye!")
}
