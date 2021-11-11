package servers

import (
	"time"

	"github.com/SunSince90/go-naivecoin/pkg/block"
	"github.com/gofiber/fiber/v2"
)

// ProbesServer implements probes for Kubernetes.
//
// A probe server is useful because it will prevent Kubernetes from marking a
// pod as ready until some condition is verified, e.g. inits.
// In our case, we will let Kubernetes know that a pod is ready only once it
// created the genesis block.
type ProbesServer struct {
	FiberApp   *fiber.App
	blockchain *block.BlockChain
}

func (p *ProbesServer) healthz(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusOK)
}

func (p *ProbesServer) readyz(c *fiber.Ctx) error {
	if p.blockchain.Length() > 0 {
		return c.SendStatus(fiber.StatusOK)
	}

	return c.SendStatus(fiber.ErrInternalServerError.Code)
}

// NewProbesServer returns a server that that implements probes for Kubernetes.
// This server needs to run on a different port from the one from
// NewPublicServer().
func NewProbesServer(blockchain *block.BlockChain) *ProbesServer {
	p := &ProbesServer{
		blockchain: blockchain,
	}

	app := fiber.New(fiber.Config{ReadTimeout: 5 * time.Second})
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World 👋!")
	})
	app.Get("/healthz", p.healthz)
	app.Get("/readyz", p.readyz)

	p.FiberApp = app
	return p
}
