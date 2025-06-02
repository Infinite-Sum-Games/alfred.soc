package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// SetupValkeyHSet creates a hash set for each name if it does not exist.
func SetupValkeyHSet(names []string, client *redis.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for _, name := range names {
		// HSetNX sets a field only if it does not exist. We'll use a dummy field to ensure creation.
		_, err := client.HSetNX(ctx, name, "__dummy__", "1").Result()
		if err != nil {
			return fmt.Errorf("failed to create hash set %s: %w", name, err)
		}
	}
	return nil
}

// VerifyHSet checks if each hash set exists (even if empty) by using the TYPE command.
func VerifyHSet(names []string, client *redis.Client) (map[string]bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	exists := make(map[string]bool)
	for _, name := range names {
		typeRes, err := client.Type(ctx, name).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to check type for %s: %w", name, err)
		}
		exists[name] = (typeRes == "hash")
	}
	return exists, nil
}
