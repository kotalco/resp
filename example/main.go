package main

import (
	"fmt"
	redis "github.com/kotalco/resp-redis"
	"log"
	"os"
)

func main() {
	// Obtain the redis server address from environment variables or default to localhost.
	redisAddress := os.Getenv("REDIS_ADDRESS")
	if redisAddress == "" {
		redisAddress = "localhost:6379"
	}

	// Initialize a new Redis client.
	client, err := redis.NewRedisClient(redisAddress, 10, "123456")
	if err != nil {
		log.Fatalf("Error connecting to redis: %s", err)
	}
	defer client.Close()

	// Set a key in Redis.
	key := "test_key"
	value := "hello world"
	err = client.Set(key, value)
	if err != nil {
		log.Fatalf("Error setting key: %s", err)
	}
	fmt.Printf("Set %s to %s\n", key, value)

	// Get the value of the key from Redis.
	value, err = client.Get(key)
	if err != nil {
		log.Fatalf("Error getting key: %s", err)
	}
	fmt.Printf("Got 'test_key': %s\n", value)
}
