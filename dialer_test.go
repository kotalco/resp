package redis

import (
	"testing"
)

func TestDialer_Dial(t *testing.T) {
	dialer := NewDialer()

	// Test dialing a valid address
	address := "localhost:6379" //assuming a Redis server is running on localhost:6379
	conn, err := dialer.Dial(address)
	if err != nil {
		t.Errorf("Failed to dial a valid address: %s", err)
	}
	if conn == nil {
		t.Errorf("Expected a non-nil connection object")
	}
	// Close the connection
	if conn != nil {
		_ = conn.Close()
	}

	// Test dialing an invalid address
	invalidAddress := "invalid:6379"
	conn, err = dialer.Dial(invalidAddress)
	if err == nil {
		t.Errorf("Expected an error when dialing an invalid address")
	}
}
