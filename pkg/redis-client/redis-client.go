package redisclient

import (
	"context"
	"fmt"
	"time"

	"github.com/AlexMickh/speak-chat/pkg/utils/retry"
	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	addr     string
	user     string
	password string
	db       int
}

func NewConfig(
	addr string,
	user string,
	password string,
	db int,
) *RedisConfig {
	return &RedisConfig{
		addr:     addr,
		user:     user,
		password: password,
		db:       db,
	}
}

func New(ctx context.Context, cfg *RedisConfig) (*redis.Client, error) {
	const op = "redis-client.New"

	var rdb *redis.Client

	err := retry.WithDelay(5, 500*time.Millisecond, func() error {
		rdb = redis.NewClient(&redis.Options{
			Addr:     cfg.addr,
			Password: cfg.password,
			DB:       cfg.db,
		})

		err := rdb.Ping(ctx).Err()
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return rdb, nil
}
