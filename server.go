package main

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// NodeServer is the server that is implemented by this pod and that can be
// used to get information from this pod and and create new blocks.
type NodeServer struct {
	FiberApp *fiber.App
	genBlock chan Block
}

// NewNodeServer creates and returns a new instance of the NodeServer.
func NewNodeServer(genBlock chan Block) *NodeServer {
	server := &NodeServer{
		FiberApp: fiber.New(fiber.Config{ReadTimeout: 30 * time.Second}),
		genBlock: genBlock,
	}

	// set up the fiber server
	app := server.FiberApp
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World 👋!")
	})
	app.Use("/blocks", func(c *fiber.Ctx) error {
		if blockchain == nil {
			c.Send([]byte("blockchain is not initialized"))
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.Next()
	})
	app.Get("/blocks", server.handleGetBlocks)
	app.Get("/blocks/last", server.handleGetLastBlock)
	app.Post("/blocks", server.handlePostBlocks)

	return server
}

func (n *NodeServer) handleGetBlocks(c *fiber.Ctx) error {
	return c.JSON(blockchain.chain)
}

func (n *NodeServer) handlePostBlocks(c *fiber.Ctx) error {
	if len(c.Body()) == 0 {
		c.Send([]byte("body cannot be empty"))
		return c.SendStatus(fiber.ErrBadRequest.Code)
	}

	block := NewBlock(string(c.Body()), blockchain.chain[len(blockchain.chain)-1])
	if err := blockchain.PushBlock(*block); err != nil {
		c.Send([]byte(err.Error()))
		return c.SendStatus(fiber.ErrBadRequest.Code)
	}

	n.genBlock <- *block

	return c.SendStatus(fiber.StatusOK)
}

func (n *NodeServer) handleGetLastBlock(c *fiber.Ctx) error {
	return c.JSON(blockchain.GetLastBlock())
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
	app.Get("/readyz", func(c *fiber.Ctx) error {
		// Having probes will prevent peers from adding each other if they
		// haven't generated the genesis block yet, because they won't be
		// flagged as ready.
		if blockchain.Length() > 0 {
			return c.SendStatus(fiber.StatusOK)
		}

		return c.SendStatus(fiber.ErrInternalServerError.Code)
	})
	return app
}
