package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func SetupValkeyStreams(names []string, client *redis.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for _, name := range names {
		// XGroupCreateMkStream will create the stream if it does not exist,
		// but requires a group name. Instead, add a dummy entry to ensure
		// creation.
		_, err := client.XAdd(ctx, &redis.XAddArgs{
			Stream: name,
			Values: map[string]any{"__dummy__": "1"},
		}).Result()
		if err != nil {
			return fmt.Errorf("failed to create stream %s: %w", name, err)
		}
	}
	return nil
}

// VerifyStreams checks if each stream exists (even if empty) by using the TYPE command.
func VerifyStreams(names []string, client *redis.Client) (map[string]bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	exists := make(map[string]bool)
	for _, name := range names {
		typeRes, err := client.Type(ctx, name).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to check type for %s: %w", name, err)
		}
		exists[name] = (typeRes == "stream")
	}
	return exists, nil
}
