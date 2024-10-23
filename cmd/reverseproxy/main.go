package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Igno21/go-rest-framework/cmd/reverseproxy/proxy"
)

// TODO: Move everything up a level.

func main() {
	proxyAddr := flag.String("a", "127.0.0.1", "Address to start the proxy server on")
	proxyPort := flag.String("p", "8080", "Port to start the proxy server on")
	singleRequest := flag.Bool("s", false, "Single request instances of backend (default false)")
	backendCount := flag.Int("b", 0, "Number of backend instances allowed at once; 0 is no limit (default 0)")

	flag.Parse()

	address := *proxyAddr + ":" + *proxyPort
	shutdown := make(chan bool)
	// Create proxy
	fmt.Printf("Creating proxy with singleRequest %v and backendCount %d\n", *singleRequest, *backendCount)
	proxy := proxy.CreateProxy(*singleRequest, *backendCount)

	// Create a server instance
	proxyServer := &http.Server{
		Addr: address,
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/health":
				rw.WriteHeader(http.StatusOK)
			case "/stop":
				proxy.Stop()
				rw.WriteHeader(http.StatusOK)
				shutdown <- true
			default:
				// Forward the request to the backend server
				fmt.Printf("Handling request for %s\n", req.RemoteAddr)
				response := proxy.ForwardRequest(req)
				if response == nil {
					http.Error(rw, "Backend server error", http.StatusInternalServerError)
					return
				}

				// Write the response back out
				rw.WriteHeader(response.StatusCode)
				if response.Body != nil {
					io.Copy(rw, response.Body)
				}
				fmt.Printf("Request Complete\n")
			}
		}),
	}

	go func() {
		fmt.Println("Starting origin server at", address)
		if err := proxyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v\n", err)
		}
		fmt.Println("Stopped serving new connections", address)
		shutdown <- true
	}()

	<-shutdown
	// Create a context with a timeout for the shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Gracefully shut down the server
	if err := proxyServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v\n", err)
	}

	<-shutdown
	fmt.Printf("Proxy server stopped: %s\n", address)
}
