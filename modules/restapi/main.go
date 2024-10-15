package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

func main() {
	originServerHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Printf("[origin server] received request at: %s\n", time.Now())
		time.Sleep(250)
		_, _ = fmt.Fprint(rw, "origin server response")
	})

	// Start listening on the first available port after 8080
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen on a port after 8080: %v", err)
	}
	defer listener.Close()

	// Get the actual port the listener is bound to
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port
	fmt.Printf("Origin server listening on port %d\n", port)

	log.Fatal(http.ListenAndServe(":8081", originServerHandler))
}
