package reqgen

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/Igno21/go-rest-framework/internal/serverutil"
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

func stopProxy(addr string) bool {
	stopProxyURL := "http://" + addr + "/stop"
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(stopProxyURL)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {

			return true
		}
	}

	return false
}

func GenerateRequests(requestCount int, serverAddr string, serverPort string, singleRequest bool, backendCount int) {
	address := serverAddr + ":" + serverPort
	jobs := make(chan string, requestCount)
	results := make(chan *http.Response, requestCount)
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
			serverAddr,
			"-p",
			serverPort,
			"-b",
			strconv.Itoa(backendCount),
		}

		if singleRequest {
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
	healthy := serverutil.HealthCheck(address, 3*time.Second, 1*time.Second, 3)
	if !healthy {
		panic("Proxy Server failed\n")
	}

	// Send requests (replace with your actual URLs)
	for j := 1; j <= requestCount; j++ {
		jobs <- "http://" + address
	}
	close(jobs)

	// Collect results
	wg.Wait()
	sCount := 0
	eCount := 0
	for a := 1; a <= requestCount; a++ {
		resp := <-results

		if code := resp.StatusCode; code == 200 {
			sCount++
		} else {
			eCount++
		}
	}

	stopProxy(address)

	// Calculate and print elapsed time
	elapsed := time.Since(start)
	fmt.Printf("Results:\n")
	fmt.Printf("Took %s\n", elapsed)
	fmt.Printf("Success: %d\tError: %d\n", sCount, eCount)
}
