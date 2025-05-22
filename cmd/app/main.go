package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlexMickh/speak-chat/internal/app"
	"github.com/AlexMickh/speak-chat/internal/config"
	"github.com/AlexMickh/speak-chat/pkg/sl"
)

func main() {
	cfg := config.MustLoad()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = sl.New(ctx, os.Stdout, cfg.Env)

	sl.GetFromCtx(ctx).Info(ctx, "logger is working", slog.String("env", cfg.Env))

	app := app.Register(ctx, cfg)
	defer app.GracefulStop(ctx)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	close(stop)
	sl.GetFromCtx(ctx).Info(ctx, "server stopped")
}
