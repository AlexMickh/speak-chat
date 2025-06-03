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
	HGet(ctx context.Context, key string, field string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Pipeline() redis.Pipeliner
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
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

func (r *Redis) SaveMessages(ctx context.Context, chatId string, messages []models.Message) error {
	const op = "storage.redis.SaveMessage"

	pipeline := r.rdb.Pipeline()
	for i, message := range messages {
		key := genKey(i, chatId)

		err := pipeline.HSet(ctx, key, message).Err()
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		err = pipeline.Expire(ctx, key, r.cfg.expiration).Err()
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	_, err := pipeline.Exec(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Redis) SaveOneMessage(ctx context.Context, chatId string, message models.Message) error {
	const op = "storage.redis.SaveOneMessage"

	pipeline := r.rdb.Pipeline()

	err := pipeline.Del(ctx, genKey(9, chatId)).Err()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	for i := range 9 {
		err = pipeline.Copy(ctx, genKey(i, chatId), genKey(i+1, chatId), r.cfg.db, true).Err()
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	err = pipeline.HSet(ctx, genKey(9, chatId), message).Err()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = pipeline.Exec(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Redis) GetMessages(ctx context.Context, chatId string) ([]models.Message, error) {
	const op = "storage.redis.GetMessages"

	// TODO: add pipeline
	var cursor uint64
	var keys []string
	var err error
	for {
		keys, _, err = r.rdb.Scan(ctx, cursor, "*&"+chatId, 10).Result()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		if cursor == 0 {
			break
		}
	}

	// sort.Slice(keys, func(i, j int) bool {
	// 	el1, _ := strconv.Atoi(strings.Split(keys[i], "&")[0])
	// 	el2, _ := strconv.Atoi(strings.Split(keys[j], "&")[0])
	// 	return el1 < el2
	// })

	pipeline := r.rdb.Pipeline()
	messages := make([]models.Message, len(keys))
	for i := range keys {
		var message models.Message

		err = pipeline.HGetAll(ctx, genKey(i, chatId)).Scan(&message)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		messages[i] = message
	}

	_, err = pipeline.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return messages, nil
}

func genKey(index int, chatId string) string {
	return fmt.Sprint(index) + "&" + chatId
}
