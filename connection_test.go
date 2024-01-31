package redis

import (
	"bufio"
	"bytes"
	"net"
	"strings"
	"testing"
)

type mockConn struct {
	net.Conn
	readBuffer  *bytes.Buffer
	writeBuffer *bytes.Buffer
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	return m.readBuffer.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	n, err = m.writeBuffer.Write(b)
	return n, err
}

func (m *mockConn) Close() error {
	return nil
}

// MockDialer is a mock type that satisfies the IDialer interface
type MockDialer struct {
	DialFunc func(address string) (net.Conn, error)
}

func (m *MockDialer) Dial(address string) (net.Conn, error) {
	return m.DialFunc(address)
}

func newMockConnection(readData string, writeData *bytes.Buffer) *Connection {
	mockConn := &mockConn{
		readBuffer:  bytes.NewBufferString(readData),
		writeBuffer: writeData,
	}
	return &Connection{
		conn: mockConn,
		rw:   bufio.NewReadWriter(bufio.NewReader(mockConn), bufio.NewWriter(mockConn)),
	}
}

func TestNewRedisConnection(t *testing.T) {
	t.Run("successful connection with authentication", func(t *testing.T) {
		mockDialer := &MockDialer{
			DialFunc: func(address string) (net.Conn, error) { // Simulate successful connection
				return &mockConn{
					readBuffer:  bytes.NewBufferString("+OK\r\n"),
					writeBuffer: new(bytes.Buffer),
				}, nil
			},
		}
		_, err := NewRedisConnection(mockDialer, "localhost:6379", "correctpassword")
		if err != nil {
			t.Errorf("Failed to create new Redis connection with error: %v", err)
		}
	})

	t.Run("failed authentication", func(t *testing.T) {
		mockDialer := &MockDialer{
			DialFunc: func(address string) (net.Conn, error) { // Simulate successful connection
				return &mockConn{
					readBuffer:  bytes.NewBufferString("-error message\r\n"),
					writeBuffer: new(bytes.Buffer),
				}, nil
			},
		}
		_, err := NewRedisConnection(mockDialer, "localhost:6379", "wrongpass")
		if err == nil {
			t.Errorf("Expected an authentication error, got nil")
		}
	})
}

func TestConnection_Auth(t *testing.T) {
	t.Run("simulate a successful AUTH command", func(t *testing.T) {
		conn := newMockConnection("+OK\r\n", new(bytes.Buffer))
		err := conn.Auth("correct-password")
		if err != nil {
			t.Errorf("Auth should succeed, got error: %v", err)
		}
	})

	t.Run("simulate an authentication failure", func(t *testing.T) {
		conn := newMockConnection("-ERR invalid password\r\n", new(bytes.Buffer))
		err := conn.Auth("wrong-password")
		if err == nil {
			t.Errorf("Expected an authentication error, got nil")
		}
	})
}

func TestConnection_Send(t *testing.T) {
	writeBuffer := new(bytes.Buffer)
	conn := newMockConnection("", writeBuffer)
	t.Run("simulate a successful send command", func(t *testing.T) {
		// Test sending a command
		err := conn.Send("PING")
		if err != nil {
			t.Errorf("Failed to send command with error: %v", err)
		}
		if writeBuffer.String() != "PING\r\n" {
			t.Errorf("Send wrote the wrong data: %v", writeBuffer.String())
		}
	})
}

func TestConnection_Receive(t *testing.T) {
	t.Run("simulate a valid response", func(t *testing.T) {
		conn := newMockConnection("PONG\r\n", new(bytes.Buffer))
		resp, err := conn.Receive()
		if err != nil {
			t.Errorf("Receive should succeed, got error: %v", err)
		}
		if resp != "PONG" {
			t.Errorf("Expected PONG, got: %v", resp)
		}
	})

	t.Run("Simulate a Redis error response", func(t *testing.T) {
		conn := newMockConnection("-some error\r\n", new(bytes.Buffer))
		_, err := conn.Receive()
		if err == nil || !strings.Contains(err.Error(), "some error") {
			t.Errorf("Expected an error containing 'some error', got: %v", err)
		}
	})

	t.Run("Simulate a bulk string response", func(t *testing.T) {
		conn := newMockConnection("$5\r\nhello\r\n", new(bytes.Buffer))
		resp, err := conn.Receive()
		if err != nil {
			t.Errorf("Receive should succeed, got error: %v", err)
		}
		if resp != "hello" {
			t.Errorf("Expected 'hello', got: %v", resp)
		}
	})

}

func TestConnection_Close(t *testing.T) {
	writeBuffer := new(bytes.Buffer)
	conn := newMockConnection("", writeBuffer)

	err := conn.Close()
	if err != nil {
		t.Errorf("Failed to close connection with error: %v", err)
	}
}
