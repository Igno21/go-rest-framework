package main

import (
	"flag"

	"github.com/Igno21/go-rest-framework/cmd/restapi/server"
)

func main() {
	originAddr := flag.String("a", "", "Address to start the origin server on")
	singleRequest := flag.Bool("s", false, "Single request instances of backend (default false)")

	flag.Parse()

	server.StartBackend(*originAddr, *singleRequest)
}
