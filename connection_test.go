package resp

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

type MockNetConn struct {
	ReadBuffer  bytes.Buffer
	WriteBuffer bytes.Buffer
	WriteErr    error
	Closed      bool
}

func (mc *MockNetConn) Read(b []byte) (n int, err error) {
	return mc.ReadBuffer.Read(b)
}

func (mc *MockNetConn) Write(b []byte) (n int, err error) {
	if mc.Closed {
		return 0, net.ErrClosed
	}
	if mc.WriteErr != nil {
		return 0, mc.WriteErr
	}
	return mc.WriteBuffer.Write(b)
}

func (mc *MockNetConn) Close() error {
	mc.Closed = true
	return nil
}

func (mc *MockNetConn) LocalAddr() net.Addr                { return nil }
func (mc *MockNetConn) RemoteAddr() net.Addr               { return nil }
func (mc *MockNetConn) SetDeadline(t time.Time) error      { return nil }
func (mc *MockNetConn) SetReadDeadline(t time.Time) error  { return nil }
func (mc *MockNetConn) SetWriteDeadline(t time.Time) error { return nil }

func newMockConnection(readData string, writeData *bytes.Buffer, setWriteDeadline time.Time) *Connection {
	mockConn := &MockNetConn{
		ReadBuffer: *bytes.NewBufferString(readData), // Initialize the buffer with readData
	}

	return &Connection{
		conn: mockConn,
		rw:   bufio.NewReadWriter(bufio.NewReader(&mockConn.ReadBuffer), bufio.NewWriter(&mockConn.WriteBuffer)),
	}
}

func TestConnection_Auth(t *testing.T) {
	t.Run("simulate a successful AUTH command", func(t *testing.T) {
		conn := newMockConnection("+OK\r\n", new(bytes.Buffer), time.Time{})
		err := conn.Auth(context.Background(), "correct-password")
		if err != nil {
			t.Errorf("Auth should succeed, got error: %v", err)
		}
	})

	t.Run("simulate an authentication failure", func(t *testing.T) {
		conn := newMockConnection("-ERR invalid password\r\n", new(bytes.Buffer), time.Time{})
		readContents, _ := conn.rw.Reader.ReadString('\n')
		fmt.Printf("Buffer content before ReadString call: %q\n", readContents)
		err := conn.Auth(context.Background(), "wrong-password")
		if err == nil {
			t.Errorf("Expected an authentication error, got nil")
		}
	})
}

func TestConnection_Send(t *testing.T) {
	t.Run("send data within deadline", func(t *testing.T) {
		commandToSend := "SET key value"
		expectedData := "SET key value\r\n"
		mockConn := newMockConnection("", new(bytes.Buffer), time.Time{})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := mockConn.Send(ctx, commandToSend)
		if err != nil {
			t.Fatalf("Send() error = %v, wantErr %v", err, nil)
		}

		if gotData := mockConn.conn.(*MockNetConn).WriteBuffer.String(); gotData != expectedData {
			t.Errorf("Send() got = %v, want %v", gotData, expectedData)
		}
	})

	t.Run("send data past deadline", func(t *testing.T) {
		commandToSend := "SET key value"
		mockConn := newMockConnection("", new(bytes.Buffer), time.Time{})

		// Set a deadline in the past
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Second))
		defer cancel()

		err := mockConn.Send(ctx, commandToSend)
		if err == nil {
			t.Error("Send() expected error, got none")
		}
	})
	t.Run("send data with canceled context", func(t *testing.T) {
		commandToSend := "SET key value"

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel the context

		mockConn := newMockConnection("", new(bytes.Buffer), time.Time{})
		err := mockConn.Send(ctx, commandToSend)
		if err == nil {
			t.Errorf("Send() expected error, got none")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Send() expected context.Canceled error, got %v", err)
		}
	})

	//todo send data with write error
	//todo send data after connection closed

}

func TestConnection_Receive(t *testing.T) {
	t.Run("receive simple string response", func(t *testing.T) {
		conn := newMockConnection("+OK\r\n", new(bytes.Buffer), time.Time{})
		data, err := conn.Receive(context.Background())
		if err != nil {
			t.Fatalf("Receive() error = %v, wantErr %v", err, nil)
		}
		if data != "OK" {
			t.Errorf("Receive() got = %v, want %v", data, "OK")
		}
	})

	t.Run("receive error response", func(t *testing.T) {
		conn := newMockConnection("-Error message\r\n", new(bytes.Buffer), time.Time{})
		_, err := conn.Receive(context.Background())
		if err == nil || !strings.Contains(err.Error(), "Error message") {
			t.Errorf("Receive() expected error containing 'Error message', got %v", err)
		}
	})

	t.Run("receive bulk string response", func(t *testing.T) {
		conn := newMockConnection("$6\r\nfoobar\r\n", new(bytes.Buffer), time.Time{})
		data, err := conn.Receive(context.Background())
		if err != nil {
			t.Fatalf("Receive() error = %v, wantErr %v", err, nil)
		}
		if data != "foobar" {
			t.Errorf("Receive() got = %v, want %v", data, "foobar")
		}
	})

	t.Run("receive nil bulk string response", func(t *testing.T) {
		conn := newMockConnection("$-1\r\n", new(bytes.Buffer), time.Time{})
		data, err := conn.Receive(context.Background())
		if err != nil {
			t.Fatalf("Receive() error = %v, wantErr %v", err, nil)
		}
		if data != "" {
			t.Errorf("Receive() got = %v, want empty string", data)
		}
	})

	t.Run("receive with canceled context", func(t *testing.T) {
		conn := newMockConnection("+OK\r\n", new(bytes.Buffer), time.Time{})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := conn.Receive(ctx)
		if err == nil {
			t.Error("Receive() expected error, got none")
		}
	})
	//todo receive data after connection closed
}
