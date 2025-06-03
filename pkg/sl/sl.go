package sl

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type key string

var (
	Key       = key("logger")
	RequestID = "request_id"
)

type Logger struct {
	log *slog.Logger
}

func New(ctx context.Context, w io.Writer, env string) context.Context {
	var log *slog.Logger

	switch env {
	case "local":
		log = slog.New(
			slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case "dev":
		log = slog.New(
			slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case "prod":
		log = slog.New(
			slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		log = slog.New(
			slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return context.WithValue(ctx, Key, &Logger{log: log})
}

func GetFromCtx(ctx context.Context) *Logger {
	return ctx.Value(Key).(*Logger)
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...any) {
	if ctx.Value(RequestID) != nil {
		fields = append(fields, slog.String(RequestID, ctx.Value(RequestID).(string)))
	}
	l.log.Info(msg, fields...)
}

func (l *Logger) Error(ctx context.Context, msg string, fields ...any) {
	if ctx.Value(RequestID) != nil {
		fields = append(fields, slog.String(RequestID, ctx.Value(RequestID).(string)))
	}
	l.log.Error(msg, fields...)
}

func (l *Logger) Fatal(ctx context.Context, msg string, fields ...any) {
	if ctx.Value(RequestID) != nil {
		fields = append(fields, slog.String(RequestID, ctx.Value(RequestID).(string)))
	}
	l.log.Info(msg, fields...)
	os.Exit(1)
}

func (l *Logger) With(ctx context.Context, fields ...any) context.Context {
	if ctx.Value(RequestID) != nil {
		fields = append(fields, slog.String(RequestID, ctx.Value(RequestID).(string)))
	}
	return context.WithValue(ctx, Key, &Logger{log: l.log.With(fields...)})
}

func Err(err error) slog.Attr {
	return slog.Attr{
		Key:   "error",
		Value: slog.StringValue(err.Error()),
	}
}

func Interceptor(ctx context.Context) grpc.UnaryServerInterceptor {
	return func(lCtx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		log := GetFromCtx(ctx)
		lCtx = context.WithValue(lCtx, Key, log)

		md, ok := metadata.FromIncomingContext(lCtx)
		if ok {
			guid, ok := md[RequestID]
			if ok {
				GetFromCtx(lCtx).Error(ctx, "No request id")
				//nolint:staticcheck
				ctx = context.WithValue(ctx, RequestID, guid)
			}
		}

		GetFromCtx(lCtx).Info(lCtx, "request",
			slog.String("method", info.FullMethod),
			slog.Time("request time", time.Now()),
		)

		return handler(lCtx, req)
	}
}
