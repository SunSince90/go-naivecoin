package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

var blockchain *BlockChain

func handleGetBlocks(c *fiber.Ctx) error {
	if blockchain == nil {
		c.Send([]byte("blockchain is not initialized"))
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.JSON(blockchain.chain)
}

func handlePostBlocks(c *fiber.Ctx) error {
	if blockchain == nil {
		c.Send([]byte("blockchain is not initialized"))
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	if len(c.Body()) == 0 {
		c.Send([]byte("body cannot be empty"))
		return c.SendStatus(fiber.ErrBadRequest.Code)
	}

	block := NewBlock(string(c.Body()), blockchain.chain[len(blockchain.chain)-1])
	if err := blockchain.PushBlock(*block); err != nil {
		c.Send([]byte(err.Error()))
		return c.SendStatus(fiber.ErrBadRequest.Code)
	}

	return c.SendStatus(fiber.StatusOK)
}

func main() {
	log := zerolog.New(os.Stderr).With().Timestamp().Logger()
	log.Info().Msg("starting...")

	blockchain = NewBlockChain()

	app := fiber.New(fiber.Config{ReadTimeout: 30 * time.Second})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World 👋!")
	})
	app.Get("/blocks", handleGetBlocks)
	app.Post("/blocks", handlePostBlocks)

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		defer close(stopChan)
		err := app.Listen(":8080")
		if err != nil {
			log.Err(err).Msg("error while listening")
		}
	}()

	<-stopChan
	fmt.Println()
	log.Info().Msg("exit requested")
	log.Info().Msg("shutting down server...")
	app.Shutdown()

	log.Info().Msg("good bye!")
}
