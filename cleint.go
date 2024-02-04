package redis

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
)

type IClient interface {
	GetConnection() (IConnection, error)
	ReleaseConnection(conn IConnection)
	Do(ctx context.Context, command string) (string, error)
	Set(ctx context.Context, key string, value string) error
	SetWithTTL(ctx context.Context, key string, value string, ttl int) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
	Incr(ctx context.Context, key string) (int, error)
	Expire(ctx context.Context, key string, seconds int) (bool, error)
	Close()
}

type Client struct {
	pool     chan IConnection
	address  string
	poolSize int
	mu       sync.Mutex // protects pool from race condition
	auth     string
	dialer   IDialer
}

func NewRedisClient(address string, poolSize int, auth string) (IClient, error) {
	client := &Client{
		pool:     make(chan IConnection, poolSize),
		address:  address,
		poolSize: poolSize,
		auth:     auth,
		dialer:   NewDialer(),
	}
	// pre-populate the pool with connections , authenticated and ready to be used
	for i := 0; i < poolSize; i++ {
		conn, err := NewRedisConnection(client.dialer, address, auth)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		client.pool <- conn
	}
	if len(client.pool) == 0 {
		return nil, errors.New("can't create redis connection")
	}

	return client, nil
}

func (client *Client) GetConnection() (IConnection, error) {
	// make sure that the access to the client.pool is synchronized among concurrent goroutines, make the operation atomic to prevent race conditions
	client.mu.Lock()
	defer client.mu.Unlock()

	select {
	case conn := <-client.pool:
		return conn, nil
	default:
		// Pool is empty now all connection are being used , create a new connection till some connections get released
		conn, err := NewRedisConnection(client.dialer, client.address, client.auth)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}
}

func (client *Client) ReleaseConnection(conn IConnection) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.pool) >= client.poolSize {
		err := conn.Close()
		if err != nil {
			return
		} //if the pool is full the new conn is closed and discarded
	} else {
		client.pool <- conn //if there is room put into the pool for future use
	}
}

func (client *Client) Do(ctx context.Context, command string) (string, error) {
	conn, err := client.GetConnection()
	if err != nil {
		return "", err
	}
	defer client.ReleaseConnection(conn)

	// Start sending the command in a separate goroutine
	errChan := make(chan error, 1)
	replyChan := make(chan string, 1)
	go func() {
		err := conn.Send(ctx, command)
		if err != nil {
			errChan <- err
			return
		}

		reply, err := conn.Receive(ctx)
		if err != nil {
			errChan <- err
		} else {
			replyChan <- reply
		}
	}()
	// Use select to wait either for the operation to complete or the context to be cancelled
	select {
	case <-ctx.Done():
		return "", ctx.Err() // The context was cancelled
	case err := <-errChan:
		return "", err // The redis operation returned an error
	case reply := <-replyChan:
		return reply, nil // The redis operation was successful
	}

}

func (client *Client) Set(ctx context.Context, key string, value string) error {
	response, err := client.Do(ctx, fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(value), value))
	if err != nil {
		return err
	}
	if response != "OK" {
		return errors.New("unexpected response from server")
	}
	return nil
}

func (client *Client) Incr(ctx context.Context, key string) (int, error) {
	// Construct the Redis INCR command
	command := fmt.Sprintf("*2\r\n$4\r\nINCR\r\n$%d\r\n%s\r\n", len(key), key)

	// Send the command to the Redis server
	response, err := client.Do(ctx, command)
	if err != nil {
		return 0, err
	}

	// Parse the response => should be in the format: ":<number>\r\n" for a successful INCR command
	var newValue int
	if _, err := fmt.Sscanf(response, ":%d\r\n", &newValue); err != nil {
		return 0, errors.New("unexpected response from server")
	}

	// Return the new value
	return newValue, nil
}

func (client *Client) Expire(ctx context.Context, key string, seconds int) (bool, error) {
	// Construct the Redis EXPIRE command
	command := fmt.Sprintf("*3\r\n$6\r\nEXPIRE\r\n$%d\r\n%s\r\n$%d\r\n%d\r\n", len(key), key, len(fmt.Sprintf("%d", seconds)), seconds)

	// Send the command to the Redis server
	response, err := client.Do(ctx, command)
	if err != nil {
		return false, err
	}

	// Parse the response => should be in the format: ":1" for a successful EXPIRE command (if the key exists), or ":0" if it does not.
	//notice that the response was in  ":1\r\n"  format then it was stripped from it's suffix in the do function
	if response == ":1" {
		return true, nil
	} else if response == ":0" {
		return false, nil
	} else {
		return false, errors.New("unexpected response from server")
	}
}

func (client *Client) SetWithTTL(ctx context.Context, key string, value string, ttl int) error {
	response, err := client.Do(ctx, fmt.Sprintf("*5\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n$2\r\nEX\r\n$%d\r\n%d\r\n", len(key), key, len(value), value, len(strconv.Itoa(ttl)), ttl))
	if err != nil {
		return err
	}
	if response != "+OK" {
		return errors.New("unexpected response from server: " + response)
	}
	return nil
}

func (client *Client) Get(ctx context.Context, key string) (string, error) {
	response, err := client.Do(ctx, fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key))
	if err != nil {
		return "", err
	}
	return response, nil
}

func (client *Client) Delete(ctx context.Context, key string) error {
	cmd := fmt.Sprintf("*2\r\n$3\r\nDEL\r\n$%d\r\n%s\r\n", len(key), key)
	response, err := client.Do(ctx, cmd)
	if err != nil {
		return err
	}
	// DEL will return an integer which is the number of keys removed.
	// ":1" for successful deletion of one key.
	// ":0" If the key does not exist
	if response != ":1" && response != ":0" {
		return errors.New("unexpected response from server")
	}

	return nil
}

func (client *Client) Close() {
	close(client.pool)
	for conn := range client.pool {
		_ = conn.Close()
	}
}
