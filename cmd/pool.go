package cmd

import (
	"context"

	"github.com/IAmRiteshKoushik/alfred/pkg"
	"github.com/jackc/pgx/v5/pgxpool"
)

var DBPool *pgxpool.Pool

func InitDB() (*pgxpool.Pool, error) {

	connString := AppConfig.DatabaseURL
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		pkg.Log.SetupFail("Failed to parse database config", err)
		return nil, err
	}

	config.MinConns = 1
	config.MaxConns = 5
	config.MaxConnLifetime = 3600
	config.MaxConnIdleTime = 1800
	config.HealthCheckPeriod = 60
	config.MaxConnLifetimeJitter = 0

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		pkg.Log.SetupFail("Failed to create connection pool", err)
		return nil, err
	}

	// Verify connection
	err = pool.Ping(context.Background())
	if err != nil {
		pkg.Log.SetupFail("Failed to ping database", err)
		return nil, err
	}

	return pool, nil
}
