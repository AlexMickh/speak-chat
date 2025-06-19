package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlexMickh/speak-chat/internal/app"
	"github.com/AlexMickh/speak-chat/internal/config"
	"github.com/AlexMickh/speak-chat/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	cfg := config.MustLoad()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = logger.New(ctx, []string{"stdout", "logs.log"}, cfg.Env)

	logger.GetFromCtx(ctx).Info(ctx, "logger is working", zap.String("env", cfg.Env))

	app := app.Register(ctx, cfg)
	defer app.GracefulStop(ctx)

	app.Run(ctx)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	close(stop)
	logger.GetFromCtx(ctx).Info(ctx, "server stopped")
}
