package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"reverseproxy/proxy"
	"time"
)

// createServer creates a new server with a pool of goroutines to handle requests

func main() {

	// Create proxy server
	server := proxy.CreateServer()

	// Create the reverse proxy handler
	reverseProxy := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Printf("[reverse proxy server] received request at: %s\n", time.Now())

		// Create new RRWrapper
		rrw := proxy.NewRRW()
		rrw.Request = req

		// Forward the request to the backend server
		fmt.Printf("Handling request for %s\n", rrw.Request.Host)
		response := server.ForwardRequest(rrw)
		if response == nil {
			http.Error(rw, "Backend server error", http.StatusInternalServerError)
			return
		}

		// Write the response back out
		rw.WriteHeader(response.StatusCode)
		io.Copy(rw, response.Body)
	})

	log.Fatal(http.ListenAndServe(":8080", reverseProxy))
}
