# GeoIP API with Fiber and MMDB

This project provides a fast and efficient REST API for retrieving geolocation information from IP addresses using Go, the Fiber framework, and MaxMind's MMDB database format. It's designed for high performance and low latency, making it suitable for applications that require quick IP lookups.

## Features

*   **Fast IP Lookups:** Leverages the highly optimized MMDB format and the `oschwald/geoip2-golang` library for rapid IP address lookups.  The MMDB database is loaded into memory for minimal latency.
*   **REST API:**  Provides a simple and easy-to-use REST API built with the Fiber framework.
*   **Multiple IP Support:**  Allows querying geolocation information for multiple IP addresses in a single request.
*   **DNS Lookup:**  Resolves hostnames to IP addresses and retrieves geolocation information.
*   **Comprehensive Geolocation Data:** Returns detailed information, including:
    *   Country (name and ISO code)
    *   City
    *   Continent (name and code)
    *   Latitude and Longitude
    *   Time Zone
    *   Postal Code
    *   Subdivision (Region/State - ISO code and name)
    *   ASN (Autonomous System Number) and Organization (if ASN database is provided)
    *   Proxy/VPN/Hosting/Tor detection
    *   Mobile Connection Detection (Note: Accuracy may vary, relies on `IsLegitimateProxy` from MMDB)

*   **Robust Error Handling:**  Handles invalid IP addresses, database errors, and other potential issues gracefully.
*   **Middleware:**  Includes middleware for:
    *   **Logging:** Logs requests and responses.
    *   **Rate Limiting:**  Protects the API from abuse using IP-based rate limiting.
    *   **Security Headers:**  Adds security headers using `helmet`.
    *   **CORS:**  Enables Cross-Origin Resource Sharing.
    *   **CSRF Protection:** Protects against Cross-Site Request Forgery attacks.
    *   **Compression:**  Compresses responses to reduce bandwidth usage.
    *   **Etag:**  Implements ETag caching to reduce redundant data transfer.
    *   **Caching:** Adds caching support.
    *  **Idempotency:** Provides idempotency key support
    *   **Panic Recovery:**  Recovers from panics to prevent server crashes.

* **Test Coverage:** Includes unit tests to ensure API correctness and stability.

## Prerequisites

*   Go 1.20 or later
*   MaxMind GeoLite2 City and (optionally) GeoLite2 ASN MMDB databases.  You can download these from MaxMind's website: [https://dev.maxmind.com/geoip/geolite2-free-geolocation-data](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) (You'll need to create a free account).  *Make sure to download the MMDB format, not the CSV format.*

## Installation

1.  **Clone the repository:**

    ```bash
    git clone <repository_url>
    cd <repository_directory>
    ```

2.  **Install dependencies:**

    ```bash
    go mod download
    ```

3.  **Place MMDB Files:** Place the `GeoLite2-City.mmdb` (and optionally `GeoLite2-ASN.mmdb`) file(s) in the project's root directory, *or* update the `MMDBPathCity` and `MMDBPathASN` values in the `main` function (in `main.go`) to point to the correct file paths.

## Usage

1.  **Build the application:**

    ```bash
    go build -o geoip-api
    ```

2.  **Run the application:**

    ```bash
    ./geoip-api
    ```

    The API will listen on port `6378` by default.

## API Endpoints

### `GET /location/:ip`

Retrieves geolocation information for a single IP address.

*   **Parameter:**
    *   `ip`: The IPv4 or IPv6 address to look up.

*   **Example:**

    ```bash
    curl http://localhost:6378/location/8.8.8.8
    ```

*   **Response (Success - 200 OK):**

    ```json
    {
        "query": "8.8.8.8",
        "status": "success",
        "continent": "North America",
        "continentCode": "NA",
        "country": "United States",
        "countryCode": "US",
        "region": "CA",
        "regionName": "California",
        "city": "Mountain View",
        "zip": "94043",
        "lat": 37.4223,
        "lon": -122.0841,
        "timezone": "America/Los_Angeles",
        "isp": "Google LLC",
        "org": "Google LLC",
        "as": "AS15169",
        "asname": "Google LLC",
        "mobile": false,
        "proxy": false,
        "hosting": false,
        "isAnonymous": false,
        "isTor": false
    }
    ```
*   **Response (Error - 400 Bad Request - Invalid IP):**

    ```json
    {"status": "error", "message": "Invalid IP address"}
    ```

*   **Response (Error - 404 Not Found - Location not found):**

    ```json
    {"status": "error", "message": "Location not found"}
    ```

### `POST /location`

Retrieves geolocation information for multiple IP addresses.

*   **Request Body:**  A JSON array of IP addresses (strings).

*   **Example:**

    ```bash
    curl -X POST http://localhost:6378/location \
         -H "Content-Type: application/json" \
         -d '["8.8.8.8", "1.1.1.1", "invalid-ip"]'
    ```

*   **Response (Success - 200 OK):**

    ```json
    [
        {
            "query": "8.8.8.8",
            "status": "success",
            "country": "United States",
            "countryCode": "US",
            "city": "Mountain View",
            "lat": 37.4223,
            "lon": -122.0841,
             "continent": "North America",
            "continentCode": "NA",
            "region": "CA",
            "regionName": "California",
            "zip": "94043",
            "timezone": "America/Los_Angeles",
             "isp": "Google LLC",
            "org": "Google LLC",
            "as": "AS15169",
             "asname": "Google LLC",
             "mobile": false,
            "proxy": false,
            "hosting": false,
             "isAnonymous": false,
             "isTor": false
        },
        {
            "query": "1.1.1.1",
            "status": "success",
            "country": "Australia",
             "continent": "Oceania",
            "countryCode": "AU",
             "region": "",
            "regionName": "",
             "city": "",
             "zip": "",
             "lat": -33.494,
             "lon": 143.2104,
             "timezone": "Australia/Sydney",
              "isp": "Cloudflare, Inc.",
            "org": "Cloudflare, Inc.",
            "as": "AS13335 Cloudflare, Inc.",
            "asname": "CLOUDFLARENET",
            "mobile": false,
            "proxy": false,
            "hosting": false,
             "isAnonymous":false,
              "isTor":false
        },
        {
            "query": "invalid-ip",
            "status": "error"
        }
    ]
    ```

### `GET /location/dns/:hostname`

Resolves a hostname to its IP addresses and retrieves geolocation information for the *first* resolved IP address.

*   **Parameter:**
    *   `hostname`: The hostname to resolve (e.g., `google.com`).

*   **Example:**

    ```bash
    curl http://localhost:6378/location/dns/google.com
    ```

*   **Response (Success - 200 OK):**  (Similar to the `/location/:ip` response)

*   **Response (Error - 404 Not Found):** If the hostname cannot be resolved or no location is found for the resolved IPs.

### `GET /health`

Simple health check endpoint.  Returns "OK" with a 200 status code if the server is running.

*   **Example:**

    ```bash
    curl http://localhost:6378/health
    ```

*   **Response (Success - 200 OK):**

    ```
    OK
    ```

## Testing

Run the tests using:

```bash
go test -v ./...
```