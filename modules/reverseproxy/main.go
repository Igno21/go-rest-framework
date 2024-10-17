package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reverseproxy/proxy"
	"syscall"
	"time"
)

func main() {
	proxyPort := flag.String("p", "8080", "Port to start the proxy server on")
	// Create proxy server
	proxyServer := proxy.CreateServer()

	// Create the reverse proxy handler
	reverseProxy := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Printf("[reverse proxy server] received request at: %s\n", time.Now())

		// Forward the request to the backend server
		fmt.Printf("Handling request for %s\n", req.RemoteAddr)
		response := proxyServer.ForwardRequest(req)
		if response == nil {
			http.Error(rw, "Backend server error", http.StatusInternalServerError)
			return
		}

		// Write the response back out
		rw.WriteHeader(response.StatusCode)
		io.Copy(rw, response.Body)
	})

	// Create a server instance
	srv := &http.Server{
		Addr:    ":" + *proxyPort,
		Handler: reverseProxy,
	}

	// Start the proxy server in a goroutine
	go func() {
		fmt.Println("Starting reverse proxy server on :" + *proxyPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v\n", err)
		}
	}()

	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a signal
	<-sigChan
	fmt.Println("Shutting down server...")
	proxyServer.Shutdown()

	// Create a context with a timeout for the shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Gracefully shut down the server
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v\n", err)
	}

	fmt.Println("Server stopped")
}
