package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func main() {
	// Handle ACME challenge
	http.HandleFunc("/.well-known/acme-challenge/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received ACME challenge request:", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("acme-challenge-response"))
	})

	// Handle all other routes via forwarder
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Skip ACME challenge paths to avoid fallback
		if strings.HasPrefix(r.URL.Path, "/.well-known/acme-challenge/") {
			http.NotFound(w, r)
			return
		}

		target := r.Header.Get("X-Forward-To")
		if target == "" {
			http.Error(w, "Missing X-Forward-To header", http.StatusBadRequest)
			return
		}

		u, err := url.Parse(target)
		if err != nil {
			http.Error(w, "Invalid target URL", http.StatusBadRequest)
			return
		}

		req, err := http.NewRequest(r.Method, u.String(), r.Body)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		req.Header = r.Header
		req.Host = u.Host

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, "Request failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for key, val := range resp.Header {
			for _, v := range val {
				w.Header().Add(key, v)
			}
		}

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})

	log.Println("WebSocket forwarder running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
