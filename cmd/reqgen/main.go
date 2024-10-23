package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
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

func healthCheck(addr string) bool {
	healthCheckURL := "http://" + addr + "/health"
	client := http.Client{Timeout: 3 * time.Second}
	retries := 10
	for i := 0; i < retries; i++ {
		resp, err := client.Get(healthCheckURL)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		fmt.Printf("Waiting for backend %s...\n", addr)
		time.Sleep(time.Second * 1)
	}
	return false
}

func stopProxy(addr string) bool {
	healthCheckURL := "http://" + addr + "/stop"
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(healthCheckURL)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {

			return true
		}
	}

	return false
}

func main() {
	// set up application flags
	requests := flag.Int("r", 100, "The number of requests to send")
	proxyAddr := flag.String("a", "127.0.0.1", "The port of the reverse proxy")
	proxyPort := flag.String("p", "8080", "The port of the reverse proxy")
	singleRequest := flag.Bool("s", false, "Single request instances of backend (default false)")
	backendCount := flag.Int("b", 0, "Number of backend instances allowed at once; 0 is no limit (default 0)")

	// parse flags
	flag.Parse()

	serverAddr := *proxyAddr + ":" + *proxyPort
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

	// Start Proxy Server
	go func() {
		args := []string{
			"-a",
			*proxyAddr,
			"-p",
			*proxyPort,
			"-b",
			strconv.Itoa(*backendCount),
		}

		if *singleRequest {
			args = append(args, "-s")
		}

		reverseproxy := exec.Command("reverseproxy", args...)
		fmt.Println("Starting reverseproxy with: " + reverseproxy.String())

		// redirect stdout/err
		reverseproxy.Stdout = os.Stdout
		reverseproxy.Stderr = os.Stderr

		err := reverseproxy.Start()
		if err != nil {
			fmt.Printf("Error starting reverseproxy: %s\n", err.Error())
		}
		// Wait for command to start
		reverseproxy.Wait()
	}()

	// wait for proxy to start
	healthy := healthCheck(serverAddr)
	if !healthy {
		panic("Proxy Server failed\n")
	}

	// Send requests (replace with your actual URLs)
	for j := 1; j <= *requests; j++ {
		jobs <- "http://" + serverAddr
	}
	close(jobs)

	// Collect results
	wg.Wait()
	sCount := 0
	eCount := 0
	for a := 1; a <= *requests; a++ {
		resp := <-results

		if code := resp.StatusCode; code == 200 {
			sCount++
		} else {
			eCount++
		}
	}

	stopProxy(serverAddr)

	// Calculate and print elapsed time
	elapsed := time.Since(start)
	fmt.Printf("Results:\n")
	fmt.Printf("Took %s\n", elapsed)
	fmt.Printf("Success: %d\tError: %d\n", sCount, eCount)

}
