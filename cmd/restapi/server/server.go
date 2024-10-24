package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

func StartBackend(addr string, singleRequest bool) {
	// TODO: can we use buffered channels and a flag to signal we've finished our queue of requests and shutdown
	//       if len(chan) is 0 for x time shut it down?

	// TODO: create a channel cycle for tracking idle time when no request has been received before shuttind down
	//       a different approach to single request but also not live forever
	shutdown := make(chan bool, 1)
	var wg sync.WaitGroup

	originServer := http.Server{
		Addr: addr,
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/health":
				rw.WriteHeader(http.StatusOK)
				return
			case "/stop":
				shutdown <- true
				return
			default:
				if singleRequest {
					wg.Add(1)       // Increment WaitGroup counter
					defer wg.Done() // Decrement counter when handler exits
				}

				fmt.Printf("[origin server] received request at: %s\n", time.Now())
				fmt.Printf("Request from : %s\n", req.RemoteAddr)
				rw.Header().Set("X-Request-ID", req.Header.Get("X-Request-ID"))

				// simulate processing time
				time.Sleep(time.Millisecond * 50)
				_, _ = fmt.Fprint(rw, addr+" - origin server response")

				if singleRequest {
					shutdown <- true
				}
			}
		}),
	}

	go func() {
		fmt.Println("Starting origin server at", addr)
		if err := originServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting Origin Server %s: %v", addr, err)
		}
		fmt.Println("Stopped serving new connections", addr)
		shutdown <- true
	}()

	<-shutdown
	// Wait for the handler to finish
	wg.Wait() // Wait for all handlers to complete

	// Create a context with a timeout for the shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Gracefully shut down the server
	if err := originServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v\n", err)
	}

	<-shutdown

	fmt.Printf("Origin Server stopped: %s\n", addr)

}

// func simulateFailures() {
// 	// Simulate random failures
// 	if failures := false; failures {
// 		if rand.Float64() < 0.1 { // 20% chance of failure
// 			if rand.Float64() < 0.5 { // 50% chance of error response
// 				http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
// 			} else { // 50% chance of panic (with recovery)
// 				defer func() {
// 					if r := recover(); r != nil {
// 						fmt.Printf("[origin server %s] Recovered from panic: %v\n", addr, r)
// 						http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
// 					}
// 				}()
// 				panic("simulated panic")
// 			}
// 			return
// 		}
// 	}
// }
