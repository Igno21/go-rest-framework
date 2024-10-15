package main

import (
	"bytes"
	"fmt"
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
func (s *Server) addBackend(serverUrl url.URL) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pool[serverUrl.Host]; ok {
		return // Backend already exists
	}

	ch := make(chan *RequestResponseWrapper)
	s.pool[serverUrl.Host] = ch

	go func() {
		defer func() {
			s.mu.Lock()
			fmt.Printf("Server handled %d requests", s.count[serverUrl.Host])
			delete(s.pool, serverUrl.Host)
			s.mu.Unlock()
		}()

		for wrapper := range ch {

			fmt.Printf("Processing request for %s: %s\n", serverUrl.Host, wrapper.Request.URL)

			// Modify the request to target the origin server
			wrapper.Request.URL.Host = serverUrl.Host
			wrapper.Request.URL.Scheme = serverUrl.Scheme
			wrapper.Request.Host = serverUrl.Host
			wrapper.Request.RequestURI = "" // Reset RequestURI

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
			newBody := serverUrl.Host + "->" + string(body)
			t := &http.Response{
				Status:        "200 OK",
				StatusCode:    200,
				Proto:         "HTTP/1.1",
				ProtoMajor:    1,
				ProtoMinor:    1,
				Body:          io.NopCloser(bytes.NewBufferString(newBody)),
				ContentLength: int64(len(body)),
				Request:       wrapper.Request,
				Header:        make(http.Header, 0),
			}

			// Send the modified response back to the reverse proxy
			wrapper.Response <- t
		}
		fmt.Printf("Backend %s removed from pool\n", serverUrl.Host)
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
		return <-wrapper.Response
	}
	s.count[serverUrl]++

	fmt.Printf("Backend %s not found\n", serverUrl)
	return nil // Or return an error response
}

func main() {
	server := createServer()
	server.count = make(map[string]int)

	originServerURL, err := url.Parse("http://127.0.0.1:8081")
	if err != nil {
		log.Fatal("invalid origin server URL")
	}

	server.addBackend(*originServerURL)

	// Create the reverse proxy handler
	reverseProxy := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Printf("[reverse proxy server] received request at: %s\n", time.Now())

		// Forward the request to the backend server
		response := server.forwardRequest(originServerURL.Host, req)
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
