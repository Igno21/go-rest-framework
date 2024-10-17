package proxy

import (
	"fmt"
	"go-rest-framework/modules/restapi"
	"math/rand"
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

type Server struct {
	mu       sync.Mutex
	pool     map[string]*HttpProxy
	count    map[string]int
	id       int
	mismatch int
}

func CreateServer() *Server {
	return &Server{
		pool:  make(map[string]*HttpProxy),
		count: make(map[string]int),
		id:    1,
	}
}

// addBackend adds a backend server to the pool and starts a goroutine to handle its requests
func (s *Server) addBackend(port string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pool[port]; ok {
		fmt.Printf("backend exists")
		return // Backend already exists
	}

	// go routine simulates a backend
	go restapi.StartBackend(port)

	httpProxy := &HttpProxy{
		make(chan *http.Request),
		make(chan *http.Response),
	}

	s.pool[port] = httpProxy

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.pool, port)
			s.mu.Unlock()
		}()

		for request := range httpProxy.Request {
			// Health check the backend server
			healthy := s.healthCheck(port)

			if !healthy {
				fmt.Printf("Restart backend %s", port)
				s.addBackend(port)
				for attempt := 0; attempt < 3 && !healthy; attempt++ {
					healthy = s.healthCheck(port)
					time.Sleep(time.Millisecond * 100)
				}
				if !healthy {
					fmt.Printf("Backend failed %s", port)
					httpProxy.Response <- &http.Response{StatusCode: http.StatusInternalServerError}
					continue
				}
			}

			fmt.Printf("Processing request for %s through %s to %s\n", request.RemoteAddr, request.Host, port)
			originalURL, err := url.Parse(request.URL.String())
			if err != nil {
				fmt.Printf("Error parsing URL: %v\n", err)
				continue
			}

			// Modify the URL to target the origin server
			originalURL.Host = "127.0.0.1:" + port
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

			fmt.Printf("[reverse proxy server] successfully received response at: %s\n\n", time.Now())
			httpProxy.Response <- response
		}
		fmt.Printf("Backend %s removed from pool\n", port)
	}()

	fmt.Printf("Backend started %s\n", port)
}

// forwardRequest forwards a request to the appropriate backend server
func (s *Server) ForwardRequest(req *http.Request) *http.Response {

	if backend, ok := s.getBackend(); ok {
		requestId := strconv.Itoa(s.id)
		s.id++
		req.Header.Set("X-Request-ID", requestId)
		backend.Request <- req
		select {
		case resp := <-backend.Response:
			responseID := resp.Header.Get("X-Request-ID")
			if requestId != responseID {
				s.mismatch++
				fmt.Printf("Error: Request ID mismatch: got=%s, want=%s\n", responseID, requestId)
			}
			return resp
		case <-time.After(5 * time.Second):
			return &http.Response{StatusCode: http.StatusGatewayTimeout}
		}
	}

	return &http.Response{StatusCode: http.StatusInternalServerError} // Return an error response
}

func (s *Server) getBackend() (*HttpProxy, bool) {
	// Allowable ports for backend applications
	originServerPorts := []string{"8081", "8082", "8083", "8084", "8085", "8086", "8087"}

	// For testing we'll use a random server
	port := originServerPorts[rand.Intn(len(originServerPorts))]

	// Add backend to the pool
	if _, ok := s.pool[port]; !ok {
		s.addBackend(port)
	}
	s.count[port]++

	return s.pool[port], true
}

func (s *Server) Shutdown() {
	fmt.Printf("SHUTTING DOWN\n")
	fmt.Printf("PORT\tCOUNT\n")
	for port, count := range s.count {
		fmt.Printf("%s\t%d\n", port, count)
	}
	fmt.Printf("Mismatch - %d\n", s.mismatch)
}

func (s *Server) healthCheck(port string) bool {
	healthCheckURL := fmt.Sprintf("http://127.0.0.1:%s", port)
	client := http.Client{Timeout: 1 * time.Second}
	retries := 5
	for i := 0; i < retries; i++ {
		resp, err := client.Get(healthCheckURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			return true
		}
		fmt.Printf("Waiting for backend %s...\n", port)
		time.Sleep(time.Millisecond * 100)
	}
	return false
}
