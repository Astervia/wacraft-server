package auth_middleware

import (
	"net"

	"github.com/gofiber/fiber/v2"
)

// Check if an IP is in the allowed CIDRs
func IPFilterMiddleware(allowedCIDRs []string) fiber.Handler {
	// Parse allowed CIDRs once
	var allowedNets []*net.IPNet
	for _, cidr := range allowedCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			allowedNets = append(allowedNets, network)
		}
	}

	return func(c *fiber.Ctx) error {
		ip := net.ParseIP(c.IP())
		if ip == nil {
			return c.SendStatus(fiber.StatusForbidden)
		}

		for _, network := range allowedNets {
			if network.Contains(ip) {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).SendString("Access denied: unauthorized IP")
	}
}
