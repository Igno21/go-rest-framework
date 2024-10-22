package main

import (
	"flag"
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

		results <- resp // Send the new response with the body
	}
}

func main() {
	// set up application flags
	requests := flag.Int("r", 100, "The number of requests to send")
	proxyAddr := flag.String("a", "http://127.0.0.1:8080", "The port of the reverse proxy")

	// parse flags
	flag.Parse()

	jobs := make(chan string, *requests)
	results := make(chan *http.Response, *requests)
	var wg sync.WaitGroup

	// Start workers
	for w := 1; w <= 1; w++ {
		wg.Add(1)
		go worker(w, jobs, results, &wg)
	}

	// Start timer
	start := time.Now()

	// Send requests (replace with your actual URLs)
	for j := 1; j <= *requests; j++ {
		jobs <- *proxyAddr
	}
	close(jobs)

	// Collect results
	// wg.Wait()
	sCount := 0
	eCount := 0
	for a := 1; a <= *requests; a++ {
		resp := <-results
		if resp != nil {
			body, _ := io.ReadAll(resp.Body) // Read the response body

			fmt.Printf("Response %d: Status code: %d\t%s\n", a, resp.StatusCode, body)
		} else {
			fmt.Printf("Response %d: Error\n", a)
		}

		if code := resp.StatusCode; code == 200 {
			sCount++
		} else {
			eCount++
		}
	}

	// Calculate and print elapsed time
	elapsed := time.Since(start)
	fmt.Printf("Took %s\n", elapsed)
	fmt.Printf("Success: %d\tError: %d", sCount, eCount)
}
