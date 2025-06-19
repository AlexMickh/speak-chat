package logger

import (
	"context"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type key string

var (
	Key       = key("logger")
	RequestID = "request_id"
)

type Logger struct {
	log *zap.Logger
}

func New(ctx context.Context, outputPaths []string, env string) context.Context {
	var cfg zap.Config

	switch env {
	case "local":
		cfg = zap.Config{
			Encoding:         "console",
			Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
			OutputPaths:      outputPaths,
			ErrorOutputPaths: []string{"stderr"},
			EncoderConfig: zapcore.EncoderConfig{
				MessageKey: "msg",
				LevelKey:   "level",
				TimeKey:    "ts",
				EncodeTime: zapcore.ISO8601TimeEncoder,
			},
		}
	case "dev":
		cfg = zap.Config{
			Encoding:         "json",
			Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
			OutputPaths:      outputPaths,
			ErrorOutputPaths: []string{"stderr"},
			EncoderConfig:    zap.NewProductionEncoderConfig(),
		}
	case "prod":
		cfg = zap.Config{
			Encoding:         "json",
			Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
			OutputPaths:      outputPaths,
			ErrorOutputPaths: []string{"stderr"},
			EncoderConfig:    zap.NewProductionEncoderConfig(),
		}
	default:
		cfg = zap.Config{
			Encoding:         "json",
			Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
			OutputPaths:      outputPaths,
			ErrorOutputPaths: []string{"stderr"},
			EncoderConfig:    zap.NewProductionEncoderConfig(),
		}
	}

	log, err := cfg.Build()
	if err != nil {
		panic("can't init logger: " + err.Error())
	}

	return context.WithValue(ctx, Key, &Logger{log: log})
}

func GetFromCtx(ctx context.Context) *Logger {
	return ctx.Value(Key).(*Logger)
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	if ctx.Value(RequestID) != nil {
		fields = append(fields, zap.String(RequestID, ctx.Value(RequestID).(string)))
	}
	l.log.Info(msg, fields...)
}

func (l *Logger) Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	if ctx.Value(RequestID) != nil {
		fields = append(fields, zap.String(RequestID, ctx.Value(RequestID).(string)))
	}
	l.log.Fatal(msg, fields...)
}

func (l *Logger) With(ctx context.Context, fields ...zap.Field) context.Context {
	if ctx.Value(RequestID) != nil {
		fields = append(fields, zap.String(RequestID, ctx.Value(RequestID).(string)))
	}
	return context.WithValue(ctx, Key, &Logger{log: l.log.With(fields...)})
}

func (l *Logger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	if ctx.Value(RequestID) != nil {
		fields = append(fields, zap.String(RequestID, ctx.Value(RequestID).(string)))
	}
	l.log.Error(msg, fields...)
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
				ctx = context.WithValue(ctx, RequestID, guid)
			}
		}

		GetFromCtx(lCtx).Info(lCtx, "request",
			zap.String("method", info.FullMethod),
			zap.Time("request time", time.Now()),
		)

		return handler(lCtx, req)
	}
}
