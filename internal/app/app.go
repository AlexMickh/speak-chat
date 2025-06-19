package app

import (
	"context"
	"fmt"
	"net"

	"github.com/AlexMickh/speak-chat/internal/config"
	authclient "github.com/AlexMickh/speak-chat/internal/grpc/clients/auth"
	"github.com/AlexMickh/speak-chat/internal/grpc/server"
	"github.com/AlexMickh/speak-chat/internal/service"
	"github.com/AlexMickh/speak-chat/internal/storage/minio"
	"github.com/AlexMickh/speak-chat/internal/storage/postgres"
	"github.com/AlexMickh/speak-chat/internal/storage/redis"
	"github.com/AlexMickh/speak-chat/pkg/logger"
	minioclient "github.com/AlexMickh/speak-chat/pkg/minio-client"
	postgresclient "github.com/AlexMickh/speak-chat/pkg/postgres-client"
	redisclient "github.com/AlexMickh/speak-chat/pkg/redis-client"
	"github.com/AlexMickh/speak-protos/pkg/api/chat"
	"github.com/jackc/pgx/v5/pgxpool"
	redislib "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type App struct {
	cfg        *config.Config
	db         *pgxpool.Pool
	cash       *redislib.Client
	server     *grpc.Server
	authClient *authclient.AuthClient
}

func Register(ctx context.Context, cfg *config.Config) *App {
	const op = "app.Register"

	ctx = logger.GetFromCtx(ctx).With(ctx, zap.String("op", op))

	logger.GetFromCtx(ctx).Info(ctx, "initing postgres")
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
		logger.GetFromCtx(ctx).Fatal(ctx, "failed to init pgx pool", zap.Error(err))
	}

	postgres := postgres.New(db)

	logger.GetFromCtx(ctx).Info(ctx, "initing minio")
	minioCfg := minioclient.NewConfig(
		cfg.S3.Endpoint,
		cfg.S3.User,
		cfg.S3.Password,
		cfg.S3.BucketName,
		cfg.S3.IsUseSsl,
	)
	s3, err := minioclient.New(ctx, minioCfg)
	if err != nil {
		logger.GetFromCtx(ctx).Fatal(ctx, "failed to init minio", zap.Error(err))
	}

	minio := minio.New(s3, cfg.S3.BucketName, cfg.S3.Expires)

	logger.GetFromCtx(ctx).Info(ctx, "initing redis")
	redisCfg := redisclient.NewConfig(
		cfg.Redis.Addr,
		cfg.Redis.User,
		cfg.Redis.Password,
		cfg.Redis.DB,
	)
	cash, err := redisclient.New(ctx, redisCfg)
	if err != nil {
		logger.GetFromCtx(ctx).Fatal(ctx, "failed to init redis", zap.Error(err))
	}

	redis := redis.New(cash, cfg.Redis.DB, cfg.Redis.Expiration)

	logger.GetFromCtx(ctx).Info(ctx, "initing serice layer")
	service := service.New(postgres, redis, minio)

	logger.GetFromCtx(ctx).Info(ctx, "initing auth client")
	authClient, err := authclient.New(cfg.AuthServiceAddr)
	if err != nil {
		logger.GetFromCtx(ctx).Fatal(ctx, "failed to init auth client", zap.Error(err))
	}

	logger.GetFromCtx(ctx).Info(ctx, "initing server")
	srv := server.New(service, authClient)
	server := grpc.NewServer(grpc.UnaryInterceptor(logger.Interceptor(ctx)))
	chat.RegisterChatServer(server, srv)

	return &App{
		cfg:        cfg,
		db:         db,
		cash:       cash,
		server:     server,
		authClient: authClient,
	}
}

func (a *App) Run(ctx context.Context) {
	const op = "app.Run"

	logger.GetFromCtx(ctx).Info(ctx, "starting app")

	ctx = logger.GetFromCtx(ctx).With(ctx, zap.String("op", op))

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", a.cfg.Port))
	if err != nil {
		logger.GetFromCtx(ctx).Fatal(ctx, "failed to listen", zap.Error(err))
	}

	go func() {
		if err := a.server.Serve(lis); err != nil {
			logger.GetFromCtx(ctx).Fatal(ctx, "failed to listen", zap.Error(err))
		}
	}()

	logger.GetFromCtx(ctx).Info(ctx, "server started", zap.Int("port", a.cfg.Port))
}

func (a *App) GracefulStop(ctx context.Context) {
	const op = "app.GracefulStop"

	ctx = logger.GetFromCtx(ctx).With(ctx, zap.String("op", op))

	logger.GetFromCtx(ctx).Info(ctx, "stopping postgres")
	a.db.Close()

	logger.GetFromCtx(ctx).Info(ctx, "stopping redis")
	err := a.cash.Close()
	if err != nil {
		logger.GetFromCtx(ctx).Fatal(ctx, "failed to stop redis")
	}

	logger.GetFromCtx(ctx).Info(ctx, "stopping auth client")
	a.authClient.Close()

	logger.GetFromCtx(ctx).Info(ctx, "stopping server")
	a.server.GracefulStop()
}
