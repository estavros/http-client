package main

import (
	"bufio"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func main() {
	startURL := "https://example.com/"
	maxRedirects := 5

	// Example custom headers
	customHeaders := map[string]string{
		"X-Custom-Header": "MyValue",
		"User-Agent":      "MyCustomClient/1.0",
	}

	finalResp, err := fetchWithRedirects(startURL, maxRedirects, customHeaders)
	if err != nil {
		panic(err)
	}

	fmt.Println("\n=== Final Response ===")
	fmt.Println(finalResp)
}

// ------------------------
// Fetch + Redirect Support
// ------------------------

func fetchWithRedirects(rawURL string, maxRedirects int, headers map[string]string) (string, error) {
	currentURL := rawURL

	for i := 0; i <= maxRedirects; i++ {
		fmt.Println("➡ Requesting:", currentURL)

		statusCode, respHeaders, body, location, err := makeRequest(currentURL, headers)
		if err != nil {
			return "", err
		}

		// If not a redirect, return
		if statusCode < 300 || statusCode > 399 {
			fmt.Println("No redirect, returning response.")
			return body, nil
		}

		if location == "" {
			return "", fmt.Errorf("redirect (%d) but no Location header", statusCode)
		}

		fmt.Printf("↪ Redirect %d (%d): %s\n", i+1, statusCode, location)

		nextURL, err := resolveURL(currentURL, location)
		if err != nil {
			return "", err
		}

		currentURL = nextURL

		_ = respHeaders // reserved for later features
	}

	return "", fmt.Errorf("too many redirects (limit %d)", maxRedirects)
}

// ------------------------
// Single HTTP/HTTPS request with timeouts
// ------------------------

func makeRequest(rawURL string, customHeaders map[string]string) (int, map[string]string, string, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 0, nil, "", "", err
	}

	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	// Dialer with connection timeout
	dialer := net.Dialer{
		Timeout: 10 * time.Second,
	}

	// Connect
	var conn net.Conn
	if u.Scheme == "https" {
		conn, err = tls.DialWithDialer(&dialer, "tcp", u.Host+":"+port, &tls.Config{
			ServerName: u.Host,
		})
	} else {
		conn, err = dialer.Dial("tcp", u.Host+":"+port)
	}
	if err != nil {
		return 0, nil, "", "", err
	}
	defer conn.Close()

	// Set read/write deadline
	conn.SetDeadline(time.Now().Add(15 * time.Second))

	// Build request
	req := strings.Builder{}
	req.WriteString(fmt.Sprintf("GET %s HTTP/1.1\r\n", u.RequestURI()))
	req.WriteString(fmt.Sprintf("Host: %s\r\n", u.Host))
	req.WriteString("Connection: close\r\n")
	req.WriteString("Accept-Encoding: gzip\r\n")

	for k, v := range customHeaders {
		req.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	req.WriteString("\r\n")

	if _, err := conn.Write([]byte(req.String())); err != nil {
		return 0, nil, "", "", err
	}

	reader := bufio.NewReader(conn)

	// Status line
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return 0, nil, "", "", err
	}
	parts := strings.SplitN(statusLine, " ", 3)
	statusCode := 0
	if len(parts) >= 2 {
		statusCode, _ = strconv.Atoi(parts[1])
	}

	// Headers
	headers := make(map[string]string)
	var location string
	var isGzip bool

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, nil, "", "", err
		}
		if line == "\r\n" {
			break
		}

		colon := strings.Index(line, ":")
		if colon > 0 {
			key := strings.ToLower(strings.TrimSpace(line[:colon]))
			value := strings.TrimSpace(line[colon+1:])
			headers[key] = value

			if key == "location" {
				location = value
			}
			if key == "content-encoding" && strings.Contains(value, "gzip") {
				isGzip = true
			}
		}
	}

	// Body reader (decompress if needed)
	var bodyReader io.Reader = reader
	if isGzip {
		gz, err := gzip.NewReader(reader)
		if err != nil {
			return 0, nil, "", "", err
		}
		defer gz.Close()
		bodyReader = gz
	}

	bodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		return 0, nil, "", "", err
	}

	return statusCode, headers, string(bodyBytes), location, nil
}

// ------------------------
// URL Resolution
// ------------------------

func resolveURL(current, next string) (string, error) {
	base, err := url.Parse(current)
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(next)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(ref).String(), nil
}
