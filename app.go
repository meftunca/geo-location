package main

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/oschwald/geoip2-golang"

	"github.com/gofiber/fiber/v2"
)

// Location struct'ını MMDB'den gelen verilere göre güncelleyelim
type Location struct {
	Query         string  `json:"query"`
	Status        string  `json:"status"` // "success" or "error"
	Continent     string  `json:"continent,omitempty"`
	ContinentCode string  `json:"continentCode,omitempty"`
	Country       string  `json:"country,omitempty"`
	CountryCode   string  `json:"countryCode,omitempty"`
	Region        string  `json:"region,omitempty"` // Subdivision'ın ISO kodu
	RegionName    string  `json:"regionName,omitempty"`
	City          string  `json:"city,omitempty"`
	Zip           string  `json:"zip,omitempty"`
	Lat           float64 `json:"lat,omitempty"`
	Lon           float64 `json:"lon,omitempty"`
	Timezone      string  `json:"timezone,omitempty"`
	ISP           string  `json:"isp,omitempty"`    // ISP bilgisi ayrı bir veritabanında (GeoLite2-ASN)
	Org           string  `json:"org,omitempty"`    // Organization bilgisi de ASN veritabanında
	AS            string  `json:"as,omitempty"`     // ASN bilgisi
	ASName        string  `json:"asname,omitempty"` // ASN adı
	Mobile        bool    `json:"mobile,omitempty"` // Traits'ten gelen bilgiler
	Proxy         bool    `json:"proxy,omitempty"`  // Traits
	PostalCode    string  `json:"postalCode,omitempty"`
}

// Config - MMDB dosya yolunu içeriyor.
type Config struct {
	MMDBPathCity string // Şehir veritabanı yolu
	MMDBPathASN  string // ASN veritabanı yolu (isteğe bağlı)
}

// App struct'ı, MMDB reader'ları içeriyor.
type App struct {
	config    Config
	cityDB    *geoip2.Reader
	asnDB     *geoip2.Reader // İsteğe bağlı
	closeOnce sync.Once
}

// initApp, uygulamayı ve MMDB reader'ları başlatır.
func initApp(config Config) (*App, error) {
	cityDB, err := geoip2.Open(config.MMDBPathCity)
	if err != nil {
		return nil, fmt.Errorf("failed to open city database: %w", err)
	}

	var asnDB *geoip2.Reader
	if config.MMDBPathASN != "" { //Eğer ASN veritabanı yolu verilmişse onu da aç.
		asnDB, err = geoip2.Open(config.MMDBPathASN)
		if err != nil {
			//ASN Db'si olmadan da çalışabilmeli.
			log.Printf("Warning: failed to open ASN database (%s): %v", config.MMDBPathASN, err)
		}
	}

	return &App{config: config, cityDB: cityDB, asnDB: asnDB}, nil
}

// close, MMDB reader'ları kapatır.
func (a *App) close() {
	a.closeOnce.Do(func() {
		if a.cityDB != nil {
			if err := a.cityDB.Close(); err != nil {
				log.Printf("Error closing city database: %v", err)
			}
		}
		if a.asnDB != nil {
			if err := a.asnDB.Close(); err != nil {
				log.Printf("Error closing ASN database: %v", err)
			}
		}
	})
}

func (a *App) mmdbLookupASN(asn net.IP) (*Location, error) {
	if a.asnDB == nil {
		return nil, fmt.Errorf("ASN database not available")
	}

	asnRecord, err := a.asnDB.ASN(asn)
	if err != nil {
		fmt.Println("ASN lookup failed: %w", err)
		return nil, fmt.Errorf("ASN lookup failed: %w", err)
	}

	return &Location{
		Status: "success",
		AS:     fmt.Sprintf("AS%d", asnRecord.AutonomousSystemNumber),
		ASName: asnRecord.AutonomousSystemOrganization,
		ISP:    asnRecord.AutonomousSystemOrganization,
		Org:    asnRecord.AutonomousSystemOrganization,
	}, nil
}

// mmdbLookup, IP adresi için MMDB'den bilgi çeker.
func (a *App) mmdbLookup(ip net.IP) (*Location, error) {
	cityRecord, err := a.cityDB.City(ip) // City veritabanından sorgula
	if err != nil {
		return nil, fmt.Errorf("city lookup failed: %w", err)
	}

	loc := &Location{
		Status:        "success",
		Continent:     cityRecord.Continent.Names["en"],
		ContinentCode: cityRecord.Continent.Code,
		Country:       cityRecord.Country.Names["en"],
		CountryCode:   cityRecord.Country.IsoCode,
		City:          cityRecord.City.Names["en"],
		Zip:           cityRecord.Postal.Code,
		Lat:           cityRecord.Location.Latitude,
		Lon:           cityRecord.Location.Longitude,
		Timezone:      cityRecord.Location.TimeZone,
		Proxy:         cityRecord.Traits.IsAnonymousProxy,
		PostalCode:    cityRecord.Postal.Code,
	}

	// Subdivision (Eyalet/Bölge) bilgisini ekle (eğer varsa)
	if len(cityRecord.Subdivisions) > 0 {
		loc.Region = cityRecord.Subdivisions[0].IsoCode // ISO kodu
		loc.RegionName = cityRecord.Subdivisions[0].Names["en"]
	}

	// ASN bilgisi (isteğe bağlı, eğer ASN veritabanı varsa)
	if a.asnDB != nil {
		asnRecord, err := a.asnDB.ASN(ip)
		if err == nil { // Hata yoksa ASN bilgilerini ekle
			loc.AS = fmt.Sprintf("AS%d", asnRecord.AutonomousSystemNumber)
			loc.ASName = asnRecord.AutonomousSystemOrganization
			loc.ISP = asnRecord.AutonomousSystemOrganization // ISP ve Org aynı olabilir
			loc.Org = asnRecord.AutonomousSystemOrganization
		} //Hata varsa loglayabilirsin ama devam et.
	}

	return loc, nil
}

// getLocationsByIPs (Çoklu IP'ye Göre Konum Bilgisi Alma - Handler Fonksiyonu)
func (a *App) getLocationsByIPs() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var ipAddresses []string
		if err := c.BodyParser(&ipAddresses); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Invalid request body"})
		}

		results := make([]Location, len(ipAddresses))
		var wg sync.WaitGroup
		wg.Add(len(ipAddresses))

		for i, ipAddress := range ipAddresses {
			go func(i int, ipAddress string) {
				defer wg.Done()

				ip := net.ParseIP(ipAddress)
				if ip == nil {
					results[i] = Location{Query: ipAddress, Status: "error"}
					return
				}

				location, err := a.mmdbLookup(ip)
				if err != nil {
					results[i] = Location{Query: ipAddress, Status: "error"} //Hata durumunda status'u error yap.
				} else {
					location.Query = ipAddress //Sorgulanan IP adresini ekle.
					results[i] = *location     // Başarılı ise, Location'ı kopyala
				}

			}(i, ipAddress)
		}

		wg.Wait()
		return c.JSON(results)
	}
}

// getLocationByDNS (Hostname'e Göre Konum Bilgisi Alma - Handler Fonksiyonu)
func (a *App) getLocationByDNS() fiber.Handler {
	return func(c *fiber.Ctx) error {
		hostname := c.Params("hostname")

		ips, err := net.LookupIP(hostname)
		if err != nil || len(ips) == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"status": "error", "message": "Could not resolve hostname or no IPs found"})
		}

		// Birden fazla IP dönebileceği için, her biri için arama yapıyoruz.
		// İlk bulunanı döndürüyoruz.
		for _, ip := range ips {
			location, err := a.mmdbLookup(ip)
			if err == nil { // Hata yoksa, sonucu döndür
				location.Query = hostname //Sorgulanan hostname'i ekliyoruz.
				return c.JSON(location)
			}
		}

		//Eşleşme bulunamadı.
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"status": "error", "message": "Location not found for any resolved IP"})
	}
}
