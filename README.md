# Simple Go TCP HTTP Client

This project demonstrates how to build a **minimal HTTP/HTTPS client in Go using raw TCP/TLS sockets**, without relying on Go’s built-in `net/http` package.  
It now includes **support for following HTTP redirects**, making it a more realistic low-level HTTP implementation.

---

## 🚀 Features

### Manual TCP-Level HTTP Handling
The client:
- Opens a raw TCP connection
- Sends a manually constructed HTTP GET request
- Parses the HTTP status line
- Reads all response headers
- Reads and returns the HTTP body

### Redirect Handling
- Detects HTTP 3xx responses
- Extracts and resolves `Location` headers
- Supports absolute and relative redirects
- Follows redirects up to a configurable limit
- Logs each redirect hop

### Smart HTTP Caching (ETag / Last-Modified)
- In-memory response caching keyed by full URL
- Automatically sends:
  - `If-None-Match` (ETag)
  - `If-Modified-Since` (Last-Modified)
- Correctly handles `304 Not Modified`
- Reuses cached response bodies transparently
- Mimics real browser cache revalidation behavior

## 📁 Code Structure

### `makeRequest()`
Handles a single HTTP request over TCP:
- Opens socket connection  
- Sends GET request  
- Parses status code, headers, body  
- Returns redirect location (if any)

### `fetchWithRedirects()`
Controls redirect flow:
- Sends initial request  
- Follows redirects until receiving a non-redirect response or exceeding a limit  

### `resolveURL()`
Resolves relative or absolute redirect targets into a full URL.

---

## 📦 Cache Behavior

When a URL has been requested before, the client:

1. Stores the response body along with `ETag` and `Last-Modified` headers
2. Sends conditional headers on subsequent requests
3. If the server replies with `304 Not Modified`:
   - No response body is downloaded
   - The cached body is reused
4. If the resource has changed:
   - The cache entry is updated automatically

This dramatically reduces bandwidth usage and improves performance while remaining fully HTTP-compliant.

---

## ⚙️ Customization

You can modify:
- `startURL` — to test any HTTP endpoint  
- `maxRedirects` — maximum number of redirect hops  
- HTTP headers — by editing the GET request construction
- `startURL` — can be either HTTP or HTTPS. The client will automatically select the correct port and connection method.

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

## 📌 Notes

- This client supports both **HTTP and HTTPS**. HTTPS requests are handled using TLS on port 443. 

---

## ⚠️ Limitations

- Only GET requests are supported.  
- Cookies and sessions are not managed automatically.  
- HTTP/2 is not supported (only HTTP/1.1).  
- No proxy support.

---

## 🧑‍💻 Running

```bash
go run main.go
