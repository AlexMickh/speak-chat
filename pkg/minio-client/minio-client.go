package minioclient

import (
	"context"
	"fmt"
	"time"

	"github.com/AlexMickh/speak-chat/pkg/utils/retry"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioConfig struct {
	endpoint   string
	user       string
	password   string
	bucketName string
	isUseSsl   bool
}

func NewConfig(
	endpoint string,
	user string,
	password string,
	bucketName string,
	isUseSsl bool,
) *MinioConfig {
	return &MinioConfig{
		endpoint:   endpoint,
		user:       user,
		password:   password,
		bucketName: bucketName,
		isUseSsl:   isUseSsl,
	}
}

func New(ctx context.Context, cfg *MinioConfig) (*minio.Client, error) {
	const op = "minio-client.New"

	var mc *minio.Client

	err := retry.WithDelay(5, 500*time.Millisecond, func() error {
		var err error

		mc, err = minio.New(cfg.endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.user, cfg.password, ""),
			Secure: cfg.isUseSsl,
		})
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		exists, err := mc.BucketExists(ctx, cfg.bucketName)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		if !exists {
			err = mc.MakeBucket(ctx, cfg.bucketName, minio.MakeBucketOptions{})
			if err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return mc, nil
}
