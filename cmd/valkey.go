package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/redis/go-redis/v9"
)

// NOTE: Setting up Valkey via RedisClient as there were issues in importing
// the GLIDE client for valkey. `go mod tidy` did not resolve all dependencies.
// Additionally, there is the `valkey-go` SDK which could have been used
// but at the time of writing (24th May, 2025), it did not have support for
// Streams which was a necessary requirement. If this changes in the future,
// please make the corresponding upgrades.
func SetupValkey() (*redis.Client, error) {
	host := AppConfig.ValkeyHost
	port := AppConfig.ValkeyPort
	resp := 3

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: "",
		DB:       0, // default DB
		Protocol: resp,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pong, err := rdb.Ping(ctx).Result() // health-check
	if err != nil {
		pkg.Log.SetupFail("[FAIL]: Health-check failed for Valkey.", err)
		return nil, err
	}
	pkg.Log.SetupInfo(
		fmt.Sprintf("[PASSED]: Health-check successfuly for Valkey. Response: %s", pong))

	return rdb, nil
}

func CloseValkey(client *redis.Client) {
	if client != nil {
		client.Close()
	}
}
