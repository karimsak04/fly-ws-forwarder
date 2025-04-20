package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
)

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
}

func main() {
	// إضافة رد وهمي لتحقق ACME
	http.HandleFunc("/.well-known/acme-challenge/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("dummy-challenge-response"))
	})

	// معالج التوجيه العادي
	http.HandleFunc("/", forwardHandler)

	log.Println("WebSocket forwarder running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
