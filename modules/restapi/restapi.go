package restapi

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"time"
)

func StartBackend(port string) {
	originServerHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Printf("[origin server] received request at: %s\n", time.Now())
		fmt.Printf("Request from : %s\n", req.RemoteAddr)
		// Simulate random failures
		failures := false
		if failures {
			if rand.Float64() < 0.1 { // 20% chance of failure
				if rand.Float64() < 0.5 { // 50% chance of error response
					http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
				} else { // 50% chance of panic (with recovery)
					defer func() {
						if r := recover(); r != nil {
							fmt.Printf("[origin server %s] Recovered from panic: %v\n", port, r)
							http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
						}
					}()
					panic("simulated panic")
				}
				return
			}
		}
		time.Sleep(time.Millisecond * 15)
		rw.Header().Set("X-Request-ID", req.Header.Get("X-Request-ID"))
		_, _ = fmt.Fprint(rw, port+" - origin server response")
	})

	fmt.Printf("Starting backend: %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, originServerHandler))
}
