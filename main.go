package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

func handle(w http.ResponseWriter, r *http.Request) {
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

	req.Header = r.Header.Clone() // clone to avoid unexpected side effects
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
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // fallback if PORT is not set
	}

	http.HandleFunc("/", handle)
	log.Println("WebSocket forwarder running on port:", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
