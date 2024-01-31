package redis

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type IConnection interface {
	Auth(password string) error
	Send(command string) error
	Receive() (string, error)
	Close() error
}
type Connection struct {
	conn net.Conn
	rw   *bufio.ReadWriter
}

func NewRedisConnection(dialer IDialer, address string, auth string) (IConnection, error) {
	conn, err := dialer.Dial(address)
	if err != nil {
		return nil, err
	}

	rc := &Connection{
		conn: conn,
		rw:   bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
	}

	if auth != "" {
		// Authenticate with Redis using the AUTH command
		if err := rc.Auth(auth); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}

	return rc, nil
}

func (rc *Connection) Auth(password string) error {
	if err := rc.Send(fmt.Sprintf("AUTH %s", password)); err != nil {
		return err
	}
	reply, err := rc.Receive()
	if err != nil {
		return err
	}
	if reply != "+OK" {
		return errors.New("authentication failed")
	}
	return nil
}

func (rc *Connection) Send(command string) error {
	_, err := rc.rw.WriteString(command + "\r\n")
	if err != nil {
		return err
	}
	return rc.rw.Flush()
}

func (rc *Connection) Receive() (string, error) {
	line, err := rc.rw.ReadString('\n')
	if err != nil {
		return "", err
	}
	if line[0] == '-' { // if the response contains - then it's a simple errors
		return "", fmt.Errorf(strings.TrimSuffix(line[1:], "\r\n"))
	}
	//Assume the reply is a bulk string ,array serialization ain't supported in this client
	if line[0] == '$' {
		length, _ := strconv.Atoi(strings.TrimSuffix(line[1:], "\r\n")) //trim the CRLF from our response
		if length == -1 {
			// This is a nil reply
			return "", nil
		}
		buf := make([]byte, length+2) // +2 for the CRLF (\r\n)
		_, err = rc.rw.Read(buf)
		if err != nil {
			return "", err
		}
		return string(buf[:length]), nil
	}
	return strings.TrimSuffix(line, "\r\n"), nil
}

func (rc *Connection) Close() error {
	return rc.conn.Close()
}
