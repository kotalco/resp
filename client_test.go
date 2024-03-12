package resp

import (
	"context"
	"testing"
)

// Mock objects and helpers
var (
	AuthFunc    func(password string) error
	PingFunc    func(ctx context.Context) error
	SendFunc    func(command string) error
	ReceiveFunc func() (string, error)
	CloseFunc   func() error
)

type mockConnection struct {
	// fields to simulate the Redis connection state
}

func (m *mockConnection) Ping(ctx context.Context) error {
	return PingFunc(ctx)
}

func (m *mockConnection) Auth(ctx context.Context, password string) error {
	return AuthFunc(password)
}

func (m *mockConnection) Send(ctx context.Context, command string) error {
	return SendFunc(command)
}

func (m *mockConnection) Receive(ctx context.Context) (string, error) {
	return ReceiveFunc()
}

func (m *mockConnection) Close() error {
	return CloseFunc()
}

// Helper function to create a Client with a mock dialer and connection
func newMockClient(poolSize int, auth string) *Client {
	client := &Client{
		address: "localhost:6379",
		auth:    auth,
		dialer:  &MockDialer{},
	}

	client.conn = &mockConnection{}

	return client
}

// TestDo tests sending a command to the Redis server
func TestClient_Do(t *testing.T) {
	SendFunc = func(command string) error {
		return nil
	}
	ReceiveFunc = func() (string, error) {
		return "OK", nil
	}
	client := newMockClient(2, "password")
	response, err := client.Do(context.Background(), "PING")
	if err != nil {
		t.Errorf("Do returned error: %s", err)
	}
	if response != "OK" {
		t.Errorf("Do did not return +OK, got: %s", response)
	}
}

func TestClient_Set(t *testing.T) {
	SendFunc = func(command string) error {
		return nil
	}
	ReceiveFunc = func() (string, error) {
		return "OK", nil
	}
	client := newMockClient(2, "password")
	err := client.Set(context.Background(), "key", "value")
	if err != nil {
		t.Errorf("Set returned error: %s", err)
	}
}

func TestClient_Incr(t *testing.T) {
	SendFunc = func(command string) error {
		return nil
	}
	ReceiveFunc = func() (string, error) {
		return ":1\r\n", nil
	}
	client := newMockClient(2, "password")
	resp, err := client.Incr(context.Background(), "key")
	if err != nil {
		t.Errorf("Incr returned error: %s", err)
	}
	if resp != 1 {
		t.Errorf("Do did not return valid reponse, got: %d", resp)
	}
}

func TestClient_Expire(t *testing.T) {
	SendFunc = func(command string) error {
		return nil
	}
	ReceiveFunc = func() (string, error) {
		return ":1", nil
	}
	client := newMockClient(2, "password")
	success, err := client.Expire(context.Background(), "key", 1)
	if err != nil {
		t.Errorf("Expire returned error: %s", err)
	}
	if !success {
		t.Errorf("invalid expire reponse")
	}

}
func TestClient_SetWithTTL(t *testing.T) {
	SendFunc = func(command string) error {
		return nil
	}
	ReceiveFunc = func() (string, error) {
		return "OK", nil
	}
	client := newMockClient(2, "password")
	err := client.SetWithTTL(context.Background(), "key", "value", 1)
	if err != nil {
		t.Errorf("Set returned error: %s", err)
	}
}

func TestClient_Get(t *testing.T) {
	SendFunc = func(command string) error {
		return nil
	}
	ReceiveFunc = func() (string, error) {
		return "value", nil
	}
	client := newMockClient(2, "password")
	resp, err := client.Get(context.Background(), "key")
	if err != nil {
		t.Errorf("Get returned error: %s", err)
	}
	if resp != "value" {
		t.Errorf("invalid Get reponse")
	}
}

func TestClient_Delete(t *testing.T) {
	SendFunc = func(command string) error {
		return nil
	}
	ReceiveFunc = func() (string, error) {
		return ":1", nil
	}
	client := newMockClient(2, "password")
	err := client.Delete(context.Background(), "key")
	if err != nil {
		t.Errorf("Delete returned error: %s", err)
	}

}

func TestClose(t *testing.T) {
	CloseFunc = func() error {
		return nil
	}
	client := newMockClient(2, "password")
	err := client.Close()
	if err != nil {
		t.Errorf("Close did not close the channel")
	}
}
