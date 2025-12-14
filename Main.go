package main

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func main() {
	startURL := "http://example.com/"
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

		// Redirect must have a Location header
		if location == "" {
			return "", fmt.Errorf("redirect (%d) but no Location header", statusCode)
		}

		fmt.Printf("↪ Redirect %d (%d): %s\n", i+1, statusCode, location)

		// Resolve relative redirects
		nextURL, err := resolveURL(currentURL, location)
		if err != nil {
			return "", err
		}

		currentURL = nextURL
	}

	return "", fmt.Errorf("too many redirects (limit %d)", maxRedirects)
}

// ------------------------
// Single HTTP request
// ------------------------

func makeRequest(rawURL string, customHeaders map[string]string) (int, map[string]string, string, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 0, nil, "", "", err
	}

	port := u.Port()
	if port == "" {
		port = "80"
	}

	conn, err := net.Dial("tcp", u.Host+":"+port)
	if err != nil {
		return 0, nil, "", "", err
	}
	defer conn.Close()

	// Build GET request
	reqBuilder := strings.Builder{}
	reqBuilder.WriteString(fmt.Sprintf("GET %s HTTP/1.1\r\n", u.RequestURI()))
	reqBuilder.WriteString(fmt.Sprintf("Host: %s\r\n", u.Host))
	reqBuilder.WriteString("Connection: close\r\n")

	// Add custom headers
	for k, v := range customHeaders {
		reqBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	reqBuilder.WriteString("\r\n")

	_, err = conn.Write([]byte(reqBuilder.String()))
	if err != nil {
		return 0, nil, "", "", err
	}

	reader := bufio.NewReader(conn)

	// Read status line
	statusLine, _ := reader.ReadString('\n')
	parts := strings.SplitN(statusLine, " ", 3)
	statusCode := 0
	if len(parts) >= 2 {
		statusCode, _ = strconv.Atoi(parts[1])
	}

	// Read headers
	headers := make(map[string]string)
	var location string

	for {
		line, _ := reader.ReadString('\n')
		if line == "\r\n" {
			break
		}

		colon := strings.Index(line, ":")
		if colon > 0 {
			key := strings.TrimSpace(line[:colon])
			value := strings.TrimSpace(line[colon+1:])
			headers[strings.ToLower(key)] = value

			if strings.ToLower(key) == "location" {
				location = value
			}
		}
	}

	// Read body
	var bodyBuilder strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		bodyBuilder.WriteString(line)
	}

	return statusCode, headers, bodyBuilder.String(), location, nil
}

// ------------------------
// URL Resolution (relative/absolute)
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
