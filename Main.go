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

type CacheEntry struct {
	Body         string
	ETag         string
	LastModified string
	Headers      map[string]string
}

var httpCache = map[string]*CacheEntry{}

func main() {
	startURL := "https://example.com/"
	maxRedirects := 5

	customHeaders := map[string]string{
		"X-Custom-Header": "MyValue",
		"User-Agent":     "MyCustomClient/1.0",
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

		// 304 → return cached body
		if statusCode == 304 {
			fmt.Println("📦 Using cached response")
			return httpCache[currentURL].Body, nil
		}

		// If not redirect → return body
		if statusCode < 300 || statusCode > 399 {
			return body, nil
		}

		if location == "" {
			return "", fmt.Errorf("redirect (%d) but no Location header", statusCode)
		}

		nextURL, err := resolveURL(currentURL, location)
		if err != nil {
			return "", err
		}
		currentURL = nextURL

		_ = respHeaders
	}

	return "", fmt.Errorf("too many redirects")
}

// ------------------------
// HTTP Request with cache revalidation
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

	dialer := net.Dialer{Timeout: 10 * time.Second}

	var conn net.Conn
	if u.Scheme == "https" {
		conn, err = tls.DialWithDialer(&dialer, "tcp", u.Host+":"+port, &tls.Config{ServerName: u.Host})
	} else {
		conn, err = dialer.Dial("tcp", u.Host+":"+port)
	}
	if err != nil {
		return 0, nil, "", "", err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(15 * time.Second))

	req := strings.Builder{}
	req.WriteString(fmt.Sprintf("GET %s HTTP/1.1\r\n", u.RequestURI()))
	req.WriteString(fmt.Sprintf("Host: %s\r\n", u.Host))
	req.WriteString("Connection: close\r\n")
	req.WriteString("Accept-Encoding: gzip\r\n")

	// Attach cache validators
	if c, ok := httpCache[rawURL]; ok {
		if c.ETag != "" {
			req.WriteString("If-None-Match: " + c.ETag + "\r\n")
		}
		if c.LastModified != "" {
			req.WriteString("If-Modified-Since: " + c.LastModified + "\r\n")
		}
	}

	for k, v := range customHeaders {
		req.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	req.WriteString("\r\n")

	if _, err := conn.Write([]byte(req.String())); err != nil {
		return 0, nil, "", "", err
	}

	reader := bufio.NewReader(conn)

	statusLine, _ := reader.ReadString('\n')
	parts := strings.Split(statusLine, " ")
	statusCode, _ := strconv.Atoi(parts[1])

	headers := map[string]string{}
	var location, etag, lastModified string
	var isGzip bool

	for {
		line, _ := reader.ReadString('\n')
		if line == "\r\n" {
			break
		}
		colon := strings.Index(line, ":")
		key := strings.ToLower(strings.TrimSpace(line[:colon]))
		val := strings.TrimSpace(line[colon+1:])
		headers[key] = val

		if key == "location" {
			location = val
		}
		if key == "etag" {
			etag = val
		}
		if key == "last-modified" {
			lastModified = val
		}
		if key == "content-encoding" && strings.Contains(val, "gzip") {
			isGzip = true
		}
	}

	// 304 → keep cache
	if statusCode == 304 {
		return statusCode, headers, "", "", nil
	}

	var bodyReader io.Reader = reader
	if isGzip {
		gz, _ := gzip.NewReader(reader)
		defer gz.Close()
		bodyReader = gz
	}

	bodyBytes, _ := io.ReadAll(bodyReader)
	body := string(bodyBytes)

	// Store in cache
	httpCache[rawURL] = &CacheEntry{
		Body:         body,
		ETag:         etag,
		LastModified: lastModified,
		Headers:      headers,
	}

	return statusCode, headers, body, location, nil
}

// ------------------------

func resolveURL(current, next string) (string, error) {
	base, _ := url.Parse(current)
	ref, _ := url.Parse(next)
	return base.ResolveReference(ref).String(), nil
}
