package main

import (
	"flag"

	"github.com/Igno21/go-rest-framework/cmd/reverseproxy/server"
)

func main() {
	proxyAddr := flag.String("a", "127.0.0.1", "Address to start the proxy server on")
	proxyPort := flag.String("p", "8080", "Port to start the proxy server on")
	singleRequest := flag.Bool("s", false, "Single request instances of backend (default false)")
	backendCount := flag.Int("b", 0, "Number of backend instances allowed at once; 0 is no limit (default 0)")

	flag.Parse()

	server.StartProxy(*proxyAddr, *proxyPort, *singleRequest, *backendCount)
}
