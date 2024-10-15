package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

func worker(id int, jobs <-chan string, results chan<- *http.Response, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range jobs {
		// Make the request and process the response
		resp, err := http.Get(j) // Use the URL from the jobs channel
		if err != nil {
			fmt.Printf("Worker %d: Error sending request: %v\n", id, err)
			results <- nil
			continue
		}
		defer resp.Body.Close() // Close the response body

		body, err := io.ReadAll(resp.Body) // Read the response body
		if err != nil {
			fmt.Printf("Worker %d: Error reading body: %v\n", id, err)
			results <- nil
			continue
		}

		// Create a new response with the read body
		newResp := &http.Response{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Header:     resp.Header,
			Body:       io.NopCloser(bytes.NewBuffer(body)), // Create a new ReadCloser from the body
		}

		results <- newResp // Send the new response with the body
	}
}

func main() {
	jobs := make(chan string, 100)
	results := make(chan *http.Response, 100)
	var wg sync.WaitGroup

	// Start workers
	for w := 1; w <= 3; w++ {
		wg.Add(1)
		go worker(w, jobs, results, &wg)
	}

	// Start timer
	start := time.Now()

	// Send requests (replace with your actual URLs)
	for j := 1; j <= 100; j++ {
		jobs <- "http://127.0.0.1:8080"
	}
	close(jobs)

	// Collect results
	wg.Wait()
	for a := 1; a <= 100; a++ {
		resp := <-results
		if resp != nil {
			body, _ := io.ReadAll(resp.Body) // Read the response body

			fmt.Printf("Response %d: Status code: %d\t%s\n", a, resp.StatusCode, body)
		} else {
			fmt.Printf("Response %d: Error\n", a)
		}
	}

	// Calculate and print elapsed time
	elapsed := time.Since(start)
	fmt.Printf("Took %s\n", elapsed)
}
