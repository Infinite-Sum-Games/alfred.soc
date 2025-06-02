package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/redis/go-redis/v9"
)

// Creating sorted sets and adding a "__dummy__" value which needs to be
// filtered out during calls to Valkey for fetching values.
func SetupValkeySSet(names []string, client *redis.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for _, name := range names {
		// ZAdd with no members does nothing, so add a dummy member with score 0
		_, err := client.ZAdd(ctx, name, redis.Z{Score: 0, Member: "__dummy__"}).Result()
		if err != nil {
			pkg.Log.SetupFail(
				fmt.Sprintf("[CRASH]: Could not create empty sorted set: %s", name),
				err)
			return err
		}
		pkg.Log.SetupInfo(
			fmt.Sprintf("[OK]: Sorted set: %s created successfully.", name),
		)
		cardinality, err := client.ZCard(ctx, name).Result()
		if err != nil {
			pkg.Log.SetupFail(
				fmt.Sprintf("[CRASH]: Could not retrieve cardinality of sorted set: %s", name),
				err)
			return err
		}
		pkg.Log.SetupInfo(fmt.Sprintf("[OK]: cardinality of sorted set %s: %d\n", name, cardinality))
	}
	return nil
}

// Verifying the existance of sorted-sets and returning a map[string]bool of their
// presence or absence as (true | false), variables
func VerifySSet(names []string, client *redis.Client) (map[string]bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	exists := make(map[string]bool)
	for _, name := range names {
		typeRes, err := client.Type(ctx, name).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to check type for %s: %w", name, err)
		}
		exists[name] = (typeRes == "zset")
	}
	return exists, nil
}
