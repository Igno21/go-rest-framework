package proxy

import (
	"fmt"
	"go-rest-framework/modules/restapi"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type HttpProxy struct {
	Request  chan *http.Request
	Response chan *http.Response
}

type ProxyPool struct {
	mu               sync.Mutex
	singleRequest    bool
	backendCount     int
	availableServer  map[string]*HttpProxy
	proxyCount       map[string]int
	proxyId          int
	proxiedMistmatch int
}

func CreateProxy(singleRequest bool, backendCount int) *ProxyPool {
	return &ProxyPool{
		singleRequest:    singleRequest,
		backendCount:     backendCount,
		availableServer:  make(map[string]*HttpProxy),
		proxyCount:       make(map[string]int),
		proxyId:          1,
		proxiedMistmatch: 0,
	}
}

// addBackend adds a backend server to the pool and starts a goroutine to handle its requests
func (pp *ProxyPool) addBackend(addr string) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if _, ok := pp.availableServer[addr]; ok {
		fmt.Printf("backend exists")
		return // Backend already exists
	}

	// go routine simulates a backend
	// TODO: race condition exists if previous backend hasn't fully stopped before starting
	//       a new one on the same address.
	go restapi.StartBackend(addr, pp.singleRequest)

	httpProxy := &HttpProxy{
		make(chan *http.Request, 1),
		make(chan *http.Response, 1),
	}

	pp.availableServer[addr] = httpProxy

	go func() {
		defer func() {
			pp.mu.Lock()
			delete(pp.availableServer, addr)
			pp.mu.Unlock()
		}()

		for request := range httpProxy.Request {
			// Health check the backend server
			// If server is not responding, attemp to start it
			// If we're still not healthy, respond with an http.InternalServerError
			// healthy := pp.healthCheck(addr)
			// if !healthy {
			// 	fmt.Printf("Restart backend %s\n", addr)
			// 	pp.addBackend(addr)
			// 	for attempt := 0; attempt < 3 && !healthy; attempt++ {
			// 		healthy = pp.healthCheck(addr)
			// 		time.Sleep(time.Millisecond * 100)
			// 	}
			// 	if !healthy {
			// 		fmt.Printf("Backend failed %s\n", addr)
			// 		httpProxy.Response <- &http.Response{StatusCode: http.StatusInternalServerError}
			// 		continue
			// 	}
			// }

			fmt.Printf("Processing request for %s through %s to %s\n", request.RemoteAddr, request.Host, addr)
			originalURL, err := url.Parse(request.URL.String())
			if err != nil {
				fmt.Printf("Error parsing URL: %v\n", err)
				continue
			}

			// Modify the URL to target the origin server
			originalURL.Host = addr
			originalURL.Scheme = "http"
			request.URL = originalURL
			request.RequestURI = ""

			// Forward the request to the origin server
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				// Handle error (e.g., create an error response)
				fmt.Printf("Error forwarding request: %v\n", err)
				httpProxy.Response <- &http.Response{StatusCode: http.StatusInternalServerError}
				continue
			}

			fmt.Printf("[reverse proxy] successfully received response at: %s\n", time.Now())
			httpProxy.Response <- response
		}
		fmt.Printf("Backend %s removed from pool\n", addr)
	}()

	fmt.Printf("Backend started %s\n", addr)
}

// forwardRequest forwards a request to the appropriate backend server
func (pp *ProxyPool) ForwardRequest(req *http.Request) *http.Response {
	fmt.Printf("[reverse proxy server] received request at: %s\n", time.Now())
	if backend, ok := pp.getServer(); ok {
		requestId := strconv.Itoa(pp.proxyId)
		pp.proxyId++
		req.Header.Set("X-Request-ID", requestId)
		backend.Request <- req
		select {
		case resp := <-backend.Response:
			responseID := resp.Header.Get("X-Request-ID")
			if requestId != responseID {
				pp.proxiedMistmatch++
				fmt.Printf("Error: Request ID mismatch: got=%s, want=%s\n", responseID, requestId)
			}

			// If we want to handle 1 request at a time, close these channels to shut down the go routines
			if pp.singleRequest {
				close(backend.Request)
				close(backend.Response)
			}
			return resp
		case <-time.After(5 * time.Second):
			return &http.Response{StatusCode: http.StatusGatewayTimeout}
		}
	}

	return &http.Response{StatusCode: http.StatusInternalServerError} // Return an error response
}

func (pp *ProxyPool) getServer() (*HttpProxy, bool) {
	var address string
	if pp.backendCount == 0 || len(pp.availableServer) < pp.backendCount {
		// if we don't have a backend server bound, or we're below the cap; add a new one
		// find available port
		// Note: you can let the system decide using :0, but if you ListenAndServer, you can't tell
		//   what port was assigned. We create our own Listener and use the address that was returned.
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return nil, false
		}
		defer listener.Close()

		address = "localhost:" + strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
		pp.addBackend(address)
	} else {
		// we're at max backends, we want to get one at random
		// TODO: Grab one with some sort of logic, not at random, this isn't great :)
		randKey := rand.IntN(len(pp.availableServer))
		iter := 0
		for key := range pp.availableServer {
			if iter == randKey {
				address = key
				break
			}
			iter++
		}
	}
	pp.proxyCount[address]++
	return pp.availableServer[address], true
}

func (pp *ProxyPool) Stop() {
	fmt.Printf("SHUTTING DOWN\n")
	fmt.Printf("Server\t\tCount\n")
	for server, count := range pp.proxyCount {
		fmt.Printf("%s\t%d\n", server, count)
	}
	fmt.Printf("Mismatch - %d\n", pp.proxiedMistmatch)
}

func (pp *ProxyPool) healthCheck(addr string) bool {
	healthCheckURL := "http://" + addr
	client := http.Client{Timeout: 1 * time.Second}
	retries := 5
	for i := 0; i < retries; i++ {
		resp, err := client.Get(healthCheckURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			return true
		}
		fmt.Printf("Waiting for backend %s...\n", addr)
		time.Sleep(time.Millisecond * 100)
	}
	return false
}
