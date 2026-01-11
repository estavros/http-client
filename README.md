# Simple Go TCP HTTP Client

This project demonstrates how to build a **minimal HTTP/HTTPS client in Go using raw TCP/TLS sockets**, without relying on Go’s built-in `net/http` package.
It now includes **support for following HTTP redirects, smart caching, and persistent connections**, making it a more realistic low-level HTTP implementation.

---

## 🚀 Features

### Manual TCP-Level HTTP Handling

The client:

* Opens a raw TCP connection (or TLS for HTTPS)
* Sends a manually constructed HTTP GET request
* Parses the HTTP status line
* Reads all response headers
* Reads and returns the HTTP body

### Redirect Handling

* Detects HTTP 3xx responses
* Extracts and resolves `Location` headers
* Supports absolute and relative redirects
* Follows redirects up to a configurable limit
* Logs each redirect hop

### Smart HTTP Caching (ETag / Last-Modified)

* In-memory response caching keyed by full URL
* Automatically sends:

  * `If-None-Match` (ETag)
  * `If-Modified-Since` (Last-Modified)
* Correctly handles `304 Not Modified`
* Reuses cached response bodies transparently
* Mimics real browser cache revalidation behavior

### Persistent Connections (Keep-Alive)

* Reuses TCP/TLS connections for multiple requests to the same host
* Reduces latency for redirects and repeated requests
* Saves CPU and network overhead by avoiding repeated handshakes
* Automatically closes connections if the server indicates `Connection: close`
* Works seamlessly with caching and redirects

---

## 📁 Code Structure

### `makeRequest()`

Handles a single HTTP request over TCP:

* Opens or reuses a socket connection (from the connection pool)
* Sends GET request
* Parses status code, headers, body
* Returns redirect location (if any)
* Supports gzip decoding and connection reuse

### `fetchWithRedirects()`

Controls redirect flow:

* Sends initial request
* Follows redirects until receiving a non-redirect response or exceeding a limit
* Uses persistent connections where possible to reduce latency

### `resolveURL()`

Resolves relative or absolute redirect targets into a full URL.

---

## 📦 Cache Behavior

When a URL has been requested before, the client:

1. Stores the response body along with `ETag` and `Last-Modified` headers
2. Sends conditional headers on subsequent requests
3. If the server replies with `304 Not Modified`:

   * No response body is downloaded
   * The cached body is reused
4. If the resource has changed:

   * The cache entry is updated automatically

**Persistent connections** work seamlessly with the cache: repeated requests to the same host (even for redirects) are faster because the TCP/TLS handshake is reused.

---

## ⚙️ Customization

You can modify:

* `startURL` — to test any HTTP endpoint
* `maxRedirects` — maximum number of redirect hops
* HTTP headers — by editing the GET request construction
* `startURL` — can be either HTTP or HTTPS; the client automatically selects the correct port and connection method

The connection pool is fully automatic and requires no configuration. It respects `Connection: close` headers and safely closes sockets if the server does not allow keep-alive.

---

## 🛠️ Custom Headers

This client supports **custom HTTP headers**. You can add headers by modifying the `customHeaders` map in `main.go`. For example:

```go
customHeaders := map[string]string{
    "User-Agent":      "MyCustomClient/1.0",
    "X-Custom-Header": "MyValue",
}
```

---

## ⚠️ Limitations

* Only GET requests are supported
* Cookies and sessions are not managed automatically
* HTTP/2 is not supported (only HTTP/1.1)
* No proxy support

---

## 🧑‍💻 Running

```bash
go run main.go
```

> Multiple requests (e.g., redirects or repeated calls to the same host) will now run faster due to connection reuse.
