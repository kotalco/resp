package redis

import (
	"context"
	"net"
	"testing"
)

type MockDialer struct {
	DialFunc func(ctx context.Context, address string) (net.Conn, error)
}

func (m *MockDialer) Dial(ctx context.Context, address string) (net.Conn, error) {
	return m.DialFunc(ctx, address)
}

func TestDialer_Dial(t *testing.T) {
	dialer := NewDialer()

	// Test dialing a valid address
	address := "localhost:6379" //assuming a Redis server is running on localhost:6379
	conn, err := dialer.Dial(context.Background(), address)
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
	conn, err = dialer.Dial(context.Background(), invalidAddress)
	if err == nil {
		t.Errorf("Expected an error when dialing an invalid address")
	}
}
