package postgresclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AlexMickh/speak-chat/pkg/utils/retry"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgConfig struct {
	host           string
	port           int
	username       string
	password       string
	database       string
	minPools       int
	maxPools       int
	migrationsPath string
}

func NewConfig(
	username string,
	password string,
	host string,
	port int,
	database string,
	minPools int,
	maxPools int,
	migrationsPath string,
) *PgConfig {
	return &PgConfig{
		host:           host,
		port:           port,
		username:       username,
		password:       password,
		database:       database,
		minPools:       minPools,
		maxPools:       maxPools,
		migrationsPath: migrationsPath,
	}
}

func New(ctx context.Context, cfg *PgConfig) (*pgxpool.Pool, error) {
	const op = "postgres-client.New"

	var pool *pgxpool.Pool

	err := retry.WithDelay(5, 500*time.Millisecond, func() error {
		var err error

		connString := fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=disable&pool_max_conns=%d&pool_min_conns=%d",
			cfg.username,
			cfg.password,
			cfg.host,
			cfg.port,
			cfg.database,
			cfg.maxPools,
			cfg.minPools,
		)

		pool, err = pgxpool.New(ctx, connString)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		err = pool.Ping(ctx)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		m, err := migrate.New(
			"file://"+cfg.migrationsPath,
			strings.Split(connString, "&")[0],
		)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("%s: %w", op, err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return pool, nil
}
