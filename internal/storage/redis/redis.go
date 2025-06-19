package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/AlexMickh/speak-chat/internal/models"
	"github.com/redis/go-redis/v9"
)

type Client interface {
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Pipeline() redis.Pipeliner
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
	RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	LRange(ctx context.Context, key string, start int64, stop int64) *redis.StringSliceCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
}

type Redis struct {
	rdb Client
	cfg struct {
		db         int
		expiration time.Duration
	}
}

func New(rdb Client, db int, expiration time.Duration) *Redis {
	return &Redis{
		rdb: rdb,
		cfg: struct {
			db         int
			expiration time.Duration
		}{
			db:         db,
			expiration: expiration,
		},
	}
}

func (r *Redis) SaveChat(ctx context.Context, chat models.Chat) error {
	const op = "storage.redis.SaveChat"

	pipeline := r.rdb.Pipeline()

	err := pipeline.HSet(ctx, chat.ID, chat).Err()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	err = pipeline.Expire(ctx, chat.ID, r.cfg.expiration).Err()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	err = r.rdb.RPush(ctx, chat.ID+"&part", chat.ParticipantsId).Err()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	err = pipeline.Expire(ctx, chat.ID+"&part", r.cfg.expiration).Err()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = pipeline.Exec(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Redis) GetChat(ctx context.Context, id string) (models.Chat, error) {
	const op = "storage.redis.GetChat"

	var chat models.Chat
	err := r.rdb.HGetAll(ctx, id).Scan(&chat)
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	err = r.rdb.Expire(ctx, id, r.cfg.expiration).Err()
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	chat.ParticipantsId, err = r.rdb.LRange(ctx, chat.ID+"&part", 0, -1).Result()
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	chat.ID = id

	return chat, nil
}

func (r *Redis) UpdateChat(ctx context.Context, chat models.Chat) error {
	const op = "storage.redis.UpdateChat"

	err := r.SaveChat(ctx, chat)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Redis) AddParticipant(ctx context.Context, chatId, participantId string) error {
	const op = "storage.redis.AddParticipant"

	err := r.rdb.RPush(ctx, chatId+"&part", participantId).Err()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Redis) DeleteChat(ctx context.Context, chatId string) error {
	const op = "storage.redis.DeleteChat"

	err := r.rdb.Del(ctx, chatId).Err()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
