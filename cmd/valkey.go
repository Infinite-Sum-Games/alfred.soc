package cmd

import (
	"fmt"
	"runtime"

	"github.com/redis/go-redis/v9"
)

func SetupValkey() (*redis.Client, error) {
	// NOTE: Setting up Valkey via RedisClient as there were issues in importing
	// the GLIDE client for valkey. `go mod tidy` did not resolve all dependencies.
	// Additionally, there is the `valkey-go` SDK which could have been used
	// but at the time of writing (24th May, 2025), it did not have support for
	// Streams which was a necessary requirement. If this changes in the future,
	// please make the corresponding upgrades.

	host := EnvVars.ValkeyHost
	port := EnvVars.ValkeyPort
	resp := 3

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: "",
		DB:       0, // default DB
		Protocol: resp,
	})

	fmt.Printf("CPU Count: %d", runtime.NumCPU())

	return rdb, nil
}

func CloseValkey(client *redis.Client) {
	if client != nil {
		client.Close()
	}
}
