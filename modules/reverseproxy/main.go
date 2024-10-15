package main

import (
	"bytes"
	"fmt"
	"go-rest-framework/modules/restapi"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Server struct {
	mu    sync.Mutex
	pool  map[string]chan *RequestResponseWrapper
	count map[string]int
}

type RequestResponseWrapper struct {
	id       int64 // do i want this for sync?
	Request  *http.Request
	Response chan *http.Response
}

// createServer creates a new server with a pool of goroutines to handle requests
func createServer() *Server {
	return &Server{
		pool: make(map[string]chan *RequestResponseWrapper),
	}
}

// addBackend adds a backend server to the pool and starts a goroutine to handle its requests
func (s *Server) addBackend(port string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// go routine simulates a backend
	go restapi.StartBackend("8081")

	if _, ok := s.pool[port]; ok {
		return // Backend already exists
	}

	ch := make(chan *RequestResponseWrapper)
	s.pool[port] = ch

	go func() {
		defer func() {
			s.mu.Lock()
			fmt.Printf("Server handled %d requests", s.count[port])
			delete(s.pool, port)
			s.mu.Unlock()
		}()

		for wrapper := range ch {

			fmt.Printf("Processing request for %s to %s\n", wrapper.Request.Host, port)
			originalURL, err := url.Parse(wrapper.Request.URL.String())
			if err != nil {
				fmt.Printf("Error parsing URL: %v\n", err)
				wrapper.Response <- nil
				continue
			}
			fmt.Printf("originalURL: %+v\n", originalURL)

			// Modify the URL to target the origin server
			originalURL.Host = "127.0.0.1:" + port
			originalURL.Scheme = "http"
			wrapper.Request.URL = originalURL
			wrapper.Request.RequestURI = ""

			// Forward the request to the origin server
			response, err := http.DefaultClient.Do(wrapper.Request)
			if err != nil {
				// Handle error (e.g., create an error response)
				fmt.Printf("Error forwarding request: %v\n", err)
				wrapper.Response <- nil // Or send an error response
				continue
			}

			// Read the original response body (if needed for logging or other purposes)
			// body, err := io.ReadAll(response.Body)
			if err != nil {
				fmt.Printf("Error reading response body: %v\n", err)
				wrapper.Response <- nil
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
				Request:       wrapper.Request,
				Header:        make(http.Header, 0),
			}

			// Send the modified response back to the reverse proxy
			wrapper.Response <- newResp
		}
		fmt.Printf("Backend %s removed from pool\n", port)
	}()
}

// forwardRequest forwards a request to the appropriate backend server
func (s *Server) forwardRequest(serverUrl string, req *http.Request) *http.Response {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ch, ok := s.pool[serverUrl]; ok {
		wrapper := &RequestResponseWrapper{
			Request:  req,
			Response: make(chan *http.Response),
		}
		ch <- wrapper
		s.count[serverUrl]++
		return <-wrapper.Response
	}

	fmt.Printf("Backend %s not found\n", serverUrl)
	return nil // Or return an error response
}

func main() {
	server := createServer()
	server.count = make(map[string]int)

	originServerPorts := "8081"

	server.addBackend(originServerPorts)

	// Create the reverse proxy handler
	reverseProxy := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Printf("[reverse proxy server] received request at: %s\n", time.Now())

		// Forward the request to the backend server
		fmt.Printf("Handling request for %s\n", req.Host)
		response := server.forwardRequest(originServerPorts, req)
		if response == nil {
			http.Error(rw, "Backend server error", http.StatusInternalServerError)
			return
		}

		// Copy the response headers
		for k, v := range response.Header {
			for _, vv := range v {
				rw.Header().Add(k, vv)
			}
		}

		// Copy the response body
		rw.WriteHeader(response.StatusCode)
		io.Copy(rw, response.Body)
	})

	log.Fatal(http.ListenAndServe(":8080", reverseProxy))
}
