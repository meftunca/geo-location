package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

// main Fonksiyonu
func main() {
	config := Config{
		MMDBPathCity: "GeoLite2-City.mmdb", // MMDB dosya yolunu belirtin
		MMDBPathASN:  "GeoLite2-ASN.mmdb",  // İsteğe bağlı ASN veritabanı
	}

	appInstance, err := initApp(config)
	if err != nil {
		log.Fatal(err)
	}
	defer appInstance.close() // Uygulama kapandığında MMDB reader'ları kapat

	app := fiber.New(fiber.Config{})
	setupMiddleware(app)
	setupRoutes(app, appInstance) // appInstance'ı geçir

	log.Fatal(app.Listen(":6378"))
}
