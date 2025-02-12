package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/oschwald/geoip2-golang"
	"github.com/stretchr/testify/assert" // assert kütüphanesini kullanıyoruz (isteğe bağlı, ama önerilir)
)

// Testler için kullanılacak sahte (mock) MMDB reader'ları
type mockCityDB struct{}
type mockASNDB struct{}

func (m *mockCityDB) City(ip net.IP) (*geoip2.City, error) {
	// Basit bir mock implementasyonu.  Gerçek bir MMDB dosyası yerine,
	// belirli IP adresleri için önceden tanımlanmış değerler döndürür.
	ipStr := ip.String()
	switch ipStr {
	case "8.8.8.8":

		return &geoip2.City{
			City: struct {
				Names     map[string]string `maxminddb:"names"`
				GeoNameID uint              `maxminddb:"geoname_id"`
			}{
				GeoNameID: 58,
				Names:     map[string]string{"en": "Mountain View"},
			},
			Country: struct {
				Names             map[string]string `maxminddb:"names"`
				IsoCode           string            `maxminddb:"iso_code"`
				GeoNameID         uint              `maxminddb:"geoname_id"`
				IsInEuropeanUnion bool              `maxminddb:"is_in_european_union"`
			}{
				GeoNameID: 6252001,
				IsoCode:   "US",
				Names:     map[string]string{"en": "United States"},
			},
			Location: struct {
				TimeZone       string  `maxminddb:"time_zone"`
				Latitude       float64 `maxminddb:"latitude"`
				Longitude      float64 `maxminddb:"longitude"`
				MetroCode      uint    `maxminddb:"metro_code"`
				AccuracyRadius uint16  `maxminddb:"accuracy_radius"`
			}{
				Latitude:  37.751,
				Longitude: -97.822,
			},
		}, nil
	case "2001:4860:4860::8888": // IPv6 örneği
		return &geoip2.City{
			City: struct {
				Names     map[string]string `maxminddb:"names"`
				GeoNameID uint              `maxminddb:"geoname_id"`
			}{
				GeoNameID: 58,
				Names:     map[string]string{"en": "Mountain View"},
			},
			Country: struct {
				Names             map[string]string `maxminddb:"names"`
				IsoCode           string            `maxminddb:"iso_code"`
				GeoNameID         uint              `maxminddb:"geoname_id"`
				IsInEuropeanUnion bool              `maxminddb:"is_in_european_union"`
			}{
				GeoNameID: 6252001,
				IsoCode:   "US",
				Names:     map[string]string{"en": "United States"},
			},
			Location: struct {
				TimeZone       string  `maxminddb:"time_zone"`
				Latitude       float64 `maxminddb:"latitude"`
				Longitude      float64 `maxminddb:"longitude"`
				MetroCode      uint    `maxminddb:"metro_code"`
				AccuracyRadius uint16  `maxminddb:"accuracy_radius"`
			}{
				Latitude:  37.751,
				Longitude: -97.822,
			},
		}, nil

	case "1.1.1.1": //Başka bir örnek
		return &geoip2.City{
			Country: struct {
				Names             map[string]string `maxminddb:"names"`
				IsoCode           string            `maxminddb:"iso_code"`
				GeoNameID         uint              `maxminddb:"geoname_id"`
				IsInEuropeanUnion bool              `maxminddb:"is_in_european_union"`
			}{
				GeoNameID: 2077456,
				IsoCode:   "AU",
				Names:     map[string]string{"en": "Australia"},
			},
		}, nil

	default:
		return nil, fmt.Errorf("IP not found")
	}
}

func (m *mockCityDB) Country(ip net.IP) (*geoip2.Country, error) {
	return nil, nil
}
func (m *mockCityDB) ASN(ip net.IP) (*geoip2.ASN, error) { return nil, nil }
func (m *mockCityDB) Close() error                       { return nil }

func (m *mockASNDB) City(ip net.IP) (*geoip2.City, error)       { return nil, nil }
func (m *mockASNDB) Country(ip net.IP) (*geoip2.Country, error) { return nil, nil }
func (m *mockASNDB) ASN(ip net.IP) (*geoip2.ASN, error) {
	if ip.String() == "1.1.1.1" {
		return &geoip2.ASN{
			AutonomousSystemNumber:       13335,
			AutonomousSystemOrganization: "CLOUDFLARENET",
		}, nil
	}
	return nil, fmt.Errorf("ASN not found")
}
func (m *mockASNDB) Close() error { return nil }

func TestGetLocationByIP(t *testing.T) {
	// Mock App oluştur
	appInstance := &App{
		cityDB: &geoip2.Reader{}, // Mock City veritabanı
		asnDB:  &geoip2.Reader{}, // Mock ASN veritabanı (isteğe bağlı)
	}

	app := fiber.New()
	setupRoutes(app, appInstance)

	t.Run("Valid IPv4", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/location/8.8.8.8", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var loc Location
		err = json.NewDecoder(resp.Body).Decode(&loc)
		assert.NoError(t, err)
		assert.Equal(t, "success", loc.Status)
		assert.Equal(t, "United States", loc.Country)
		assert.Equal(t, "US", loc.CountryCode)
		assert.Equal(t, "Mountain View", loc.City) // Mock'tan gelen şehir

		// Diğer alanları da kontrol et...
	})

	t.Run("Valid IPv6", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/location/2001:4860:4860::8888", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var loc Location
		err = json.NewDecoder(resp.Body).Decode(&loc)
		assert.NoError(t, err)
		assert.Equal(t, "success", loc.Status)
		assert.Equal(t, "United States", loc.Country)
		assert.Equal(t, "US", loc.CountryCode)
		assert.Equal(t, "Mountain View", loc.City)
	})

	t.Run("Invalid IP", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/location/invalid-ip", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		// Yanıt body'sini kontrol et (isteğe bağlı)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Invalid IP address")
	})

	t.Run("IP Not Found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/location/127.0.0.1", nil) // Mock'ta olmayan bir IP
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Location not found")
	})
}

func TestGetLocationsByIPs(t *testing.T) {
	appInstance := &App{
		cityDB: &geoip2.Reader{}, // Mock City veritabanı
		asnDB:  &geoip2.Reader{}, // Mock ASN veritabanı (isteğe bağlı)
	}
	app := fiber.New()
	setupRoutes(app, appInstance)

	t.Run("Valid IP List", func(t *testing.T) {
		reqBody := strings.NewReader(`["8.8.8.8", "1.1.1.1"]`) // JSON array
		req := httptest.NewRequest("POST", "/location", reqBody)
		req.Header.Set("Content-Type", "application/json") // ÖNEMLİ!
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var locations []Location
		err = json.NewDecoder(resp.Body).Decode(&locations)
		assert.NoError(t, err)
		assert.Len(t, locations, 2)
		assert.Equal(t, "success", locations[0].Status)
		assert.Equal(t, "United States", locations[0].Country)
		assert.Equal(t, "success", locations[1].Status)
		assert.Equal(t, "Australia", locations[1].Country)

	})

	t.Run("Invalid IP in List", func(t *testing.T) {
		reqBody := strings.NewReader(`["8.8.8.8", "invalid-ip"]`)
		req := httptest.NewRequest("POST", "/location", reqBody)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode) // Hala 200 OK dönmeli

		var locations []Location
		err = json.NewDecoder(resp.Body).Decode(&locations)
		assert.NoError(t, err)
		assert.Len(t, locations, 2)
		assert.Equal(t, "success", locations[0].Status)
		assert.Equal(t, "error", locations[1].Status) // Hatalı IP için "error"
	})
	t.Run("Empty IP List", func(t *testing.T) {
		reqBody := strings.NewReader(`[]`) // Boş bir liste
		req := httptest.NewRequest("POST", "/location", reqBody)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode) // Boş liste için de 200 OK

		var locations []Location
		err = json.NewDecoder(resp.Body).Decode(&locations)
		assert.NoError(t, err)
		assert.Len(t, locations, 0) // Sonuç listesi de boş olmalı
	})

	t.Run("Invalid Request Body", func(t *testing.T) {
		reqBody := strings.NewReader(`{invalid-json}`) // Bozuk JSON
		req := httptest.NewRequest("POST", "/location", reqBody)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestGetLocationByDNS(t *testing.T) {
	appInstance := &App{
		cityDB: &geoip2.Reader{}, // Mock City veritabanı
		asnDB:  &geoip2.Reader{}, // Mock ASN veritabanı (isteğe bağlı)
	}
	app := fiber.New()
	setupRoutes(app, appInstance)

	t.Run("Valid Hostname", func(t *testing.T) {

		req := httptest.NewRequest("GET", "/location/dns/google.com", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var loc Location
		err = json.NewDecoder(resp.Body).Decode(&loc)
		assert.NoError(t, err)
		assert.Equal(t, "success", loc.Status)
		assert.Equal(t, "United States", loc.Country)

	})

	t.Run("Invalid Hostname", func(t *testing.T) {

		req := httptest.NewRequest("GET", "/location/dns/invalid-hostname", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
	t.Run("Unresolvable Hostname, No IPs", func(t *testing.T) {

		req := httptest.NewRequest("GET", "/location/dns/noips.example.com", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Could not resolve hostname or no IPs found")
	})
}

func TestHealthCheck(t *testing.T) {
	app := fiber.New()
	app.Get("/health", healthCheck())

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "OK", string(body))
}
