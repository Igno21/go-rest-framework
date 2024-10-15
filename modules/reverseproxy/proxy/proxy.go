package proxy

import (
	"bytes"
	"fmt"
	"go-rest-framework/modules/restapi"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
)

type Server struct {
	mu    sync.Mutex
	pool  map[string]chan *RequestResponseWrapper
	count map[string]int
}

type RequestResponseWrapper struct {
	Request  *http.Request
	Response chan *http.Response
}

func NewRRW() *RequestResponseWrapper {
	return &RequestResponseWrapper{}
}

func CreateServer() *Server {
	return &Server{
		pool:  make(map[string]chan *RequestResponseWrapper),
		count: make(map[string]int),
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

	backend := make(chan *RequestResponseWrapper)
	s.pool[port] = backend

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.pool, port)
			s.mu.Unlock()
		}()

		for rrw := range backend {

			fmt.Printf("Processing request for %s to %s\n", rrw.Request.Host, port)
			originalURL, err := url.Parse(rrw.Request.URL.String())
			if err != nil {
				fmt.Printf("Error parsing URL: %v\n", err)
				rrw.Response = nil
				continue
			}
			fmt.Printf("originalURL: %+v\n", originalURL)

			// Modify the URL to target the origin server
			originalURL.Host = "127.0.0.1:" + port
			originalURL.Scheme = "http"
			rrw.Request.URL = originalURL
			rrw.Request.RequestURI = ""

			// Forward the request to the origin server
			response, err := http.DefaultClient.Do(rrw.Request)
			if err != nil {
				// Handle error (e.g., create an error response)
				fmt.Printf("Error forwarding request: %v\n", err)
				rrw.Response <- &http.Response{StatusCode: http.StatusInternalServerError}
				continue
			}

			// Create a new response with the modified body
			body, _ := io.ReadAll(response.Body)
			newBody := port + "->" + string(body)
			newResp := &http.Response{
				Status:        response.Status,
				StatusCode:    response.StatusCode,
				Proto:         response.Proto,
				ProtoMajor:    response.ProtoMajor,
				ProtoMinor:    response.ProtoMinor,
				Body:          io.NopCloser(bytes.NewBufferString(newBody)),
				ContentLength: int64(len(body)),
				Request:       rrw.Request,
				Header:        make(http.Header, 0),
			}
			newResp.Body.Close() // Close response body

			// Send the modified response back to the reverse proxy
			rrw.Response <- newResp
		}
		fmt.Printf("Backend %s removed from pool\n", port)
	}()

	fmt.Printf("Backend started %s\n", port)
}

// forwardRequest forwards a request to the appropriate backend server
func (s *Server) ForwardRequest(rrw *RequestResponseWrapper) *http.Response {

	if backend, ok := s.getBackend(); ok {
		rrw.Response = make(chan *http.Response)
		backend <- rrw

		return <-rrw.Response
	}

	return &http.Response{StatusCode: http.StatusInternalServerError} // Return an error response
}

func (s *Server) getBackend() (chan *RequestResponseWrapper, bool) {
	// Allowable ports for backend applications
	originServerPorts := []string{"8081", "8082", "8083", "8084"}

	// For testing we'll use a random server
	backendPort := originServerPorts[rand.Intn(len(originServerPorts))]

	// Add backend to the pool
	if _, ok := s.pool[backendPort]; !ok {
		s.addBackend(backendPort)
	}

	// count of time's we've used the backend
	s.count[backendPort]++

	return s.pool[backendPort], true
}

func (s *Server) Shutdown() {
	fmt.Printf("SHUTTING DOWN\n")
	fmt.Printf("PORT\tCOUNT\n")
	for port, count := range s.count {
		fmt.Printf("%s\t%d\n", port, count)
	}
}
