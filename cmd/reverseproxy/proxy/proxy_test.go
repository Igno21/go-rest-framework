package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthCheck(t *testing.T) {
	// Create a test server that responds with a 200 OK status
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Get the address of the test server
	addr := ts.Listener.Addr().String()

	// Test cases
	tests := []struct {
		name     string
		addr     string
		expected bool
	}{
		{"Healthy server", addr, true},
		{"Invalid address", "invalid-address", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := healthCheck(tt.addr, 3*time.Millisecond, 1*time.Millisecond, 3); got != tt.expected {
				// Override the sleep function to avoid delays in the test

				t.Errorf("healthCheck(%s) = %v; want %v", tt.addr, got, tt.expected)
			}
		})
	}
}
