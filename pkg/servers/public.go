package servers

import (
	"time"

	"github.com/SunSince90/go-naivecoin/pkg/block"
	"github.com/SunSince90/go-naivecoin/pkg/pb"
	"github.com/gofiber/fiber/v2"
)

// PublicServer exposes some information about the pod and that should be equal
// to all pods, e.g. the blocks or blockchain.
type PublicServer struct {
	FiberApp     *fiber.App
	genBlock     chan *pb.Block
	blockchain   *block.BlockChain
	blockFactory *block.BlockFactory
}

// NewPublicServer creates and returns a new instance of the PublicServer.
func NewPublicServer(blockchain *block.BlockChain, genBlock chan *pb.Block, blockFactory *block.BlockFactory) *PublicServer {
	server := &PublicServer{
		FiberApp:     fiber.New(fiber.Config{ReadTimeout: 30 * time.Second}),
		genBlock:     genBlock,
		blockchain:   blockchain,
		blockFactory: blockFactory,
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
	app.Post("/blocks", server.handlePostBlocks)
	// Probably more paths will come...
	return server
}

func (n *PublicServer) handleGetBlocks(c *fiber.Ctx) error {
	return c.JSON(n.blockchain.GetChain())
}

func (n *PublicServer) handlePostBlocks(c *fiber.Ctx) error {
	if len(c.Body()) == 0 {
		c.Send([]byte("body cannot be empty"))
		return c.SendStatus(fiber.ErrBadRequest.Code)
	}

	block := n.blockFactory.NewBlock(string(c.Body()), n.blockchain.GetLastBlock())
	if err := n.blockchain.PushBlock(block); err != nil {
		c.Send([]byte(err.Error()))
		return c.SendStatus(fiber.ErrBadRequest.Code)
	}

	n.genBlock <- block

	return c.SendStatus(fiber.StatusOK)
}
