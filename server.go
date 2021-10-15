package main

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

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

// NewServer creates a server and returns.
func NewServer() *fiber.App {
	app := fiber.New(fiber.Config{ReadTimeout: 30 * time.Second})
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World 👋!")
	})
	app.Get("/blocks", handleGetBlocks)
	app.Post("/blocks", handlePostBlocks)
	return app
}

// NewProbesServer returns a server that that implements probes for Kubernetes.
// This server needs to run on a different port from the one from NewServer().
func NewProbesServer() *fiber.App {
	app := fiber.New(fiber.Config{ReadTimeout: 5 * time.Second})
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World 👋!")
	})
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Post("/readyz", func(c *fiber.Ctx) error {
		// TODO: guard this with a sync.Mutex?
		// Having probes will prevent peers from adding each other if they
		// haven't generated the genesis block yet, because they won't be
		// flagged as ready.
		if len(blockchain.chain) > 0 {
			return c.SendStatus(fiber.StatusOK)
		}

		return c.SendStatus(fiber.ErrInternalServerError.Code)
	})
	return app
}
