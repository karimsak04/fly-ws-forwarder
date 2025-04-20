package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
)

func acmeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Responding to ACME challenge:", r.URL.Path)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("acme-challenge-response"))
}

func forwardHandler(w http.ResponseWriter, r *http.Request) {
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

	// Copy body to preserve reusability
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	req, err := http.NewRequest(r.Method, u.String(), io.NopCloser(io.MultiReader(io.NewSectionReader(io.NewBuffer(bodyBytes), 0, int64(len(bodyBytes))))))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	req.Header = r.Header.Clone()
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
	http.HandleFunc("/.well-known/acme-challenge/", acmeHandler)
	http.HandleFunc("/", forwardHandler)

	log.Println("WebSocket forwarder running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
