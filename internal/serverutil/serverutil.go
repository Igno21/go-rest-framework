package serverutil

import (
	"fmt"
	"net/http"
	"time"
)

func HealthCheck(addr string, timeout time.Duration, retryTime time.Duration, retryCount int) bool {
	healthCheckURL := "http://" + addr + "/health"
	client := http.Client{Timeout: timeout}
	for i := 0; i < retryCount; i++ {
		resp, err := client.Get(healthCheckURL)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		fmt.Printf("Waiting for backend %s...\n", addr)
		time.Sleep(retryTime)
	}
	return false
}
