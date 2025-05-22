package app

import (
	"context"
	"log/slog"

	"github.com/AlexMickh/speak-chat/internal/config"
	postgresclient "github.com/AlexMickh/speak-chat/pkg/postgres-client"
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
