# Simple Go TCP HTTP Client

This project demonstrates how to implement a minimal HTTP client in Go using **raw TCP sockets**, without relying on the built-in `net/http` package.

It manually:

- Opens a TCP connection to a server  
- Constructs and sends an HTTP GET request  
- Reads and parses the HTTP status line  
- Follows HTTP redirects  
- Prints custom status messages  
- Prints the full HTTP response  

---

## 🚀 Features

### ✔ Minimal HTTP Client (No `net/http`)
The client communicates directly over TCP, giving full visibility into how HTTP works under the hood.

### ✔ Custom Request Headers
Send any custom header you need.

### ✔ Status Code Parsing
Friendly messages for:

- `200 OK`
- `404 Not Found`
- `500 Internal Server Error`
- All other codes shown as generic warnings

### ✔ **Redirect Following (New!)**
The client automatically follows standard HTTP redirect codes:

- `301`, `302`, `303`, `307`, `308`

Redirects are resolved correctly whether they are:

- Absolute URLs  
- Relative paths  

A redirect limit prevents infinite loops.

### ✔ Full Response Output
Prints the complete server response, including:

- Status line  
- Headers  
- Body  

---

## 📄 How It Works

1. Connects to a TCP server (e.g., `example.com:80`)
2. Builds and sends an HTTP GET request
3. Reads the HTTP status line
4. Checks for redirects
5. If a redirect is found:
   - Extracts the `Location` header  
   - Resolves relative URLs  
   - Repeats the request  
6. Prints final response body

---

## 🧪 Example Output

