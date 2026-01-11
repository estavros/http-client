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
	"sync"
	"time"
)

type CacheEntry struct {
	Body         string
	ETag         string
	LastModified string
	Headers      map[string]string
}

var httpCache = map[string]*CacheEntry{}

// =========================
// 🔥 Connection Pool
// =========================

type pooledConn struct {
	conn     net.Conn
	lastUsed time.Time
}

var connPool = struct {
	sync.Mutex
	conns map[string][]*pooledConn
}{
	conns: make(map[string][]*pooledConn),
}

func getConn(host, scheme string) (net.Conn, error) {
	key := scheme + "://" + host

	connPool.Lock()
	list := connPool.conns[key]
	if len(list) > 0 {
		pc := list[len(list)-1]
		connPool.conns[key] = list[:len(list)-1]
		connPool.Unlock()

		pc.conn.SetDeadline(time.Now().Add(15 * time.Second))
		return pc.conn, nil
	}
	connPool.Unlock()

	// Open new connection
	dialer := net.Dialer{Timeout: 10 * time.Second}
	port := "80"
	if scheme == "https" {
		port = "443"
	}

	if scheme == "https" {
		return tls.DialWithDialer(&dialer, "tcp", host+":"+port, &tls.Config{ServerName: host})
	}
	return dialer.Dial("tcp", host+":"+port)
}

func releaseConn(host, scheme string, conn net.Conn) {
	key := scheme + "://" + host

	connPool.Lock()
	connPool.conns[key] = append(connPool.conns[key], &pooledConn{
		conn:     conn,
		lastUsed: time.Now(),
	})
	connPool.Unlock()
}

// =========================

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

		statusCode, _, body, location, err := makeRequest(currentURL, headers)
		if err != nil {
			return "", err
		}

		if statusCode == 304 {
			fmt.Println("📦 Using cached response")
			return httpCache[currentURL].Body, nil
		}

		if statusCode < 300 || statusCode > 399 {
			return body, nil
		}

		if location == "" {
			return "", fmt.Errorf("redirect (%d) but no Location header", statusCode)
		}

		nextURL, _ := resolveURL(currentURL, location)
		currentURL = nextURL
	}

	return "", fmt.Errorf("too many redirects")
}

// ------------------------
// HTTP Request
// ------------------------

func makeRequest(rawURL string, customHeaders map[string]string) (int, map[string]string, string, string, error) {
	u, _ := url.Parse(rawURL)

	conn, err := getConn(u.Host, u.Scheme)
	if err != nil {
		return 0, nil, "", "", err
	}

	reader := bufio.NewReader(conn)

	req := strings.Builder{}
	req.WriteString(fmt.Sprintf("GET %s HTTP/1.1\r\n", u.RequestURI()))
	req.WriteString(fmt.Sprintf("Host: %s\r\n", u.Host))
	req.WriteString("Connection: keep-alive\r\n")
	req.WriteString("Accept-Encoding: gzip\r\n")

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
		conn.Close()
		return 0, nil, "", "", err
	}

	statusLine, _ := reader.ReadString('\n')
	parts := strings.Split(statusLine, " ")
	statusCode, _ := strconv.Atoi(parts[1])

	headers := map[string]string{}
	var location, etag, lastModified string
	var isGzip bool
	var keepAlive = true

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
		if key == "connection" && strings.Contains(strings.ToLower(val), "close") {
			keepAlive = false
		}
	}

	if statusCode == 304 {
		if keepAlive {
			releaseConn(u.Host, u.Scheme, conn)
		} else {
			conn.Close()
		}
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

	httpCache[rawURL] = &CacheEntry{
		Body:         body,
		ETag:         etag,
		LastModified: lastModified,
		Headers:      headers,
	}

	if keepAlive {
		releaseConn(u.Host, u.Scheme, conn)
	} else {
		conn.Close()
	}

	return statusCode, headers, body, location, nil
}

// ------------------------

func resolveURL(current, next string) (string, error) {
	base, _ := url.Parse(current)
	ref, _ := url.Parse(next)
	return base.ResolveReference(ref).String(), nil
}
