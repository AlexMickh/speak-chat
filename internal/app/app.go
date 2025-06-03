package app

import (
	"context"
	"log/slog"

	"github.com/AlexMickh/speak-chat/internal/config"
	"github.com/AlexMickh/speak-chat/internal/storage/minio"
	"github.com/AlexMickh/speak-chat/internal/storage/postgres"
	"github.com/AlexMickh/speak-chat/internal/storage/redis"
	minioclient "github.com/AlexMickh/speak-chat/pkg/minio-client"
	postgresclient "github.com/AlexMickh/speak-chat/pkg/postgres-client"
	redisclient "github.com/AlexMickh/speak-chat/pkg/redis-client"
	"github.com/AlexMickh/speak-chat/pkg/sl"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	db *pgxpool.Pool
}

func Register(ctx context.Context, cfg *config.Config) *App {
	const op = "app.Register"

	ctx = sl.GetFromCtx(ctx).With(ctx, slog.String("op", op))

	sl.GetFromCtx(ctx).Info(ctx, "initing postgres")
	pgCfg := postgresclient.NewConfig(
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.Name,
		cfg.DB.MinPools,
		cfg.DB.MaxPools,
		cfg.DB.MigrationsPath,
	)
	db, err := postgresclient.New(ctx, pgCfg)
	if err != nil {
		sl.GetFromCtx(ctx).Fatal(ctx, "failed to init pgx pool", sl.Err(err))
	}

	postgres := postgres.New(db)
	_ = postgres

	sl.GetFromCtx(ctx).Info(ctx, "initing minio")
	minioCfg := minioclient.NewConfig(
		cfg.S3.Endpoint,
		cfg.S3.User,
		cfg.S3.Password,
		cfg.S3.BucketName,
		cfg.S3.IsUseSsl,
	)
	s3, err := minioclient.New(ctx, minioCfg)
	if err != nil {
		sl.GetFromCtx(ctx).Fatal(ctx, "failed to init minio", sl.Err(err))
	}

	minio := minio.New(s3, cfg.S3.BucketName)
	_ = minio

	sl.GetFromCtx(ctx).Info(ctx, "initing redis")
	redisCfg := redisclient.NewConfig(
		cfg.Redis.Addr,
		cfg.Redis.User,
		cfg.Redis.Password,
		cfg.Redis.DB,
	)
	cash, err := redisclient.New(ctx, redisCfg)
	if err != nil {
		sl.GetFromCtx(ctx).Fatal(ctx, "failed to init redis", sl.Err(err))
	}

	redis := redis.New(cash, cfg.Redis.DB, cfg.Redis.Expiration)
	_ = redis

	return &App{
		db: db,
	}
}

func (a *App) GracefulStop(ctx context.Context) {
	const op = "app.GracefulStop"

	ctx = sl.GetFromCtx(ctx).With(ctx, slog.String("op", op))

	sl.GetFromCtx(ctx).Info(ctx, "stopping postgres")
	a.db.Close()
}
