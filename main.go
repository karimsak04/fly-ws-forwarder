package main

import (
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// List of disallowed IP ranges to prevent SSRF
var disallowedCIDRs = []string{
	"127.0.0.0/8",     // localhost
	"10.0.0.0/8",      // private networks
	"172.16.0.0/12",   // private networks
	"192.168.0.0/16",  // private networks
	"169.254.0.0/16",  // link-local
	"::1/128",         // IPv6 localhost
	"fc00::/7",        // IPv6 unique local
	"fe80::/10",       // IPv6 link-local
}

// Checks if a host resolves to a disallowed IP
func isIPAllowed(host string) bool {
	ipList, err := net.LookupIP(host)
	if err != nil {
		return false
	}
	for _, ip := range ipList {
		for _, cidr := range disallowedCIDRs {
			_, subnet, _ := net.ParseCIDR(cidr)
			if subnet.Contains(ip) {
				return false
			}
		}
	}
	return true
}

func acmeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Responding to ACME challenge:", r.URL.Path)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("acme-challenge-response"))
}

func forwardHandler(w http.ResponseWriter, r *http.Request) {
	targetRaw := r.Header.Get("X-Forward-To")
	if targetRaw == "" {
		http.Error(w, "Missing X-Forward-To header", http.StatusBadRequest)
		return
	}

	targetURL, err := url.Parse(targetRaw)
	if err != nil || targetURL.Scheme == "" || targetURL.Host == "" {
		http.Error(w, "Invalid target URL", http.StatusBadRequest)
		return
	}

	host := strings.Split(targetURL.Host, ":")[0]
	if !isIPAllowed(host) {
		http.Error(w, "Target IP not allowed", http.StatusForbidden)
		return
	}

	// Buffer request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	r.Body.Close()

	// Prepare a reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.URL.Path = targetURL.Path
		req.Host = targetURL.Host
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))

		// Copy headers except hop-by-hop
		req.Header = http.Header{}
		for key, values := range r.Header {
			if isHopByHopHeader(key) {
				continue
			}
			for _, v := range values {
				req.Header.Add(key, v)
			}
		}
	}
	proxy.ServeHTTP(w, r)
}

func isHopByHopHeader(key string) bool {
	hopHeaders := []string{
		"Connection", "Proxy-Connection", "Keep-Alive",
		"Transfer-Encoding", "TE", "Trailer", "Upgrade",
	}
	key = http.CanonicalHeaderKey(key)
	for _, h := range hopHeaders {
		if h == key {
			return true
		}
	}
	return false
}

func main() {
	http.HandleFunc("/.well-known/acme-challenge/", acmeHandler)
	http.HandleFunc("/", forwardHandler)

	log.Println("Secure WebSocket forwarder running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
