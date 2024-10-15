package restapi

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func StartBackend(port string) {
	originServerHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Printf("[origin server] received request at: %s\n", time.Now())
		// Simulate processing time
		_, _ = fmt.Fprint(rw, "origin server response")
	})

	fmt.Printf("Starting backend: %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, originServerHandler))
}
