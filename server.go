package main

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	websocket "github.com/gofiber/websocket/v2"
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

	app.Use("/connect", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/connect", websocket.New(server.handleWebSocket))

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

func (n *NodeServer) handleWebSocket(c *websocket.Conn) {
	var (
		mt  int
		msg []byte
		err error
	)
	defer c.Close()

	for {
		var peerMsg PeerMessage
		mt, msg, err = c.ReadMessage()
		if err != nil {
			log.Err(err).Msg("error while reading message")
			continue
		}

		switch mt {
		case websocket.CloseGoingAway, websocket.CloseMessage, websocket.CloseNormalClosure:
			log.Info().Msg("received close message from peer")
			c.Close()
		case websocket.BinaryMessage:
			log.Info().Msg("received message from peer")
		default:
			log.Error().Msg("cannot understand message from peer: unrecognized message type")
			continue
		}

		if err := json.Unmarshal(msg, &peerMsg); err != nil {
			log.Err(err).Msg("could not unmarshal message from peer")
			continue
		}

		if peerMsg.MessageType != SendLastBlock {
			// TODO: will there be any other message types?
			log.Error().Str("peer-message-type", string(peerMsg.MessageType)).Msg("unrecognized peer message type")
			continue
		}

		if len(peerMsg.Blocks) == 0 {
			log.Error().Msg("peer sent no blocks")
			continue
		}

		if err := blockchain.PushBlock(peerMsg.Blocks[0]); err != nil {
			log.Err(err).Msg("error while adding peer's block to my blockchain")
			continue
		}
		log.Info().Msg("block parsed and added")
	}
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
