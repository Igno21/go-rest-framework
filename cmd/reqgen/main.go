package main

import (
	"flag"

	"github.com/Igno21/go-rest-framework/cmd/reqgen/reqgen"
)

func main() {
	// set up application flags
	requests := flag.Int("r", 100, "The number of requests to send")
	proxyAddr := flag.String("a", "127.0.0.1", "The port of the reverse proxy")
	proxyPort := flag.String("p", "8080", "The port of the reverse proxy")
	singleRequest := flag.Bool("s", false, "Single request instances of backend (default false)")
	backendCount := flag.Int("b", 0, "Number of backend instances allowed at once; 0 is no limit (default 0)")

	// parse flags
	flag.Parse()

	reqgen.GenerateRequests(*requests, *proxyAddr, *proxyPort, *singleRequest, *backendCount)

}
