package main

import (
	"fmt"
	"net"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/idempotency"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// setupMiddleware (Middleware Kurulumu)
func setupMiddleware(app *fiber.App) {
	// ... (Middleware kurulumu aynı kalıyor) ...
	app.Use(
		logger.New(),
		limiter.New(limiter.Config{
			Max:        100,
			Expiration: 30 * time.Second,
			KeyGenerator: func(c *fiber.Ctx) string {
				return c.IP() // IP bazlı limitleme (IPv4 ve IPv6'yı otomatik olarak ele alır)
			},
			LimitReached: func(c *fiber.Ctx) error {
				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"status": "error", "message": "Rate limit exceeded"})
			},
		}),
		helmet.New(),
		cors.New(),
		csrf.New(),
		compress.New(),
		etag.New(),
		cache.New(),
		idempotency.New(),
		recover.New(),
	)
}

// getLocationByIP (IP'ye Göre Konum Bilgisi Alma - Handler Fonksiyonu)
func (a *App) getLocationByIP() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ipAddress := c.Params("ip", c.IP())
		ip := net.ParseIP(ipAddress)
		if ip == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Invalid IP address"})
		}

		location, err := a.mmdbLookup(ip)
		if err != nil {
			//Burada daha detaylı loglama yapılabilir.
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"status": "error", "message": "Location not found"})
		}
		location.Query = ipAddress //Sorgulanan IP'yi ekle.
		return c.JSON(location)
	}
}

// /asn/:asn
func (a *App) getASNByNumber() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// asn := c.Params("asn")
		ip := net.ParseIP(c.Params("ip", c.IP()))
		fmt.Println("ip", ip)
		asnData, err := a.mmdbLookupASN(ip)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"status": "error", "message": "ASN not found"})
		}
		return c.JSON(asnData)
	}
}

// healthCheck (Sağlık Kontrolü - Handler Fonksiyonu)
func healthCheck() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendString("OK")
	}
}

// setupRoutes (Route'ları Tanımlama)
func setupRoutes(app *fiber.App, a *App) { // App struct'ını parametre olarak al
	locationGroup := app.Group("/location")
	locationGroup.Get("/", a.getLocationByIP())               // a'yı kullan
	locationGroup.Get("/:ip", a.getLocationByIP())            // a'yı kullan
	locationGroup.Post("/", a.getLocationsByIPs())            // a'yı kullan
	locationGroup.Get("/dns/:hostname", a.getLocationByDNS()) // a'yı kullan
	// asn
	asnGroup := app.Group("/asn")
	asnGroup.Get("/", a.getASNByNumber())
	asnGroup.Get("/:ip", a.getASNByNumber())

	app.Get("/health", healthCheck())
}
