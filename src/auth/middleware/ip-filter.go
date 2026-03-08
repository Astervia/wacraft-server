package auth_middleware

import (
	"net"

	"github.com/Astervia/wacraft-server/src/config/env"
	"github.com/gofiber/fiber/v2"
)

// IPAllowlistMiddleware blocks any IP not contained in allowedCIDRs.
// Returns a passthrough handler when allowedCIDRs is empty.
func IPAllowlistMiddleware(allowedCIDRs []string) fiber.Handler {
	if len(allowedCIDRs) == 0 {
		return func(c *fiber.Ctx) error { return c.Next() }
	}

	allowedNets := parseCIDRs(allowedCIDRs)

	return func(c *fiber.Ctx) error {
		ip := net.ParseIP(c.IP())
		if ip == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
		}
		for _, network := range allowedNets {
			if network.Contains(ip) {
				return c.Next()
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied: unauthorized IP"})
	}
}

// IPDenylistMiddleware blocks any IP contained in deniedCIDRs.
// Returns a passthrough handler when deniedCIDRs is empty.
func IPDenylistMiddleware(deniedCIDRs []string) fiber.Handler {
	if len(deniedCIDRs) == 0 {
		return func(c *fiber.Ctx) error { return c.Next() }
	}

	deniedNets := parseCIDRs(deniedCIDRs)

	return func(c *fiber.Ctx) error {
		ip := net.ParseIP(c.IP())
		if ip == nil {
			return c.Next()
		}
		for _, network := range deniedNets {
			if network.Contains(ip) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
			}
		}
		return c.Next()
	}
}

// NewAllowlistMiddleware returns an allowlist middleware configured from env.IPAllowlist.
func NewAllowlistMiddleware() fiber.Handler {
	return IPAllowlistMiddleware(env.IPAllowlist)
}

// NewDenylistMiddleware returns a denylist middleware configured from env.IPDenylist.
func NewDenylistMiddleware() fiber.Handler {
	return IPDenylistMiddleware(env.IPDenylist)
}

func parseCIDRs(cidrs []string) []*net.IPNet {
	var nets []*net.IPNet
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			nets = append(nets, network)
		}
	}
	return nets
}
