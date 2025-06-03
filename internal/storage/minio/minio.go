package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/AlexMickh/speak-chat/internal/models"
	"github.com/minio/minio-go/v7"
)

type Client interface {
	PutObject(
		ctx context.Context,
		bucketName string,
		objectName string,
		reader io.Reader,
		objectSize int64,
		opts minio.PutObjectOptions,
	) (info minio.UploadInfo, err error)
	PresignedGetObject(
		ctx context.Context,
		bucketName string,
		objectName string,
		expires time.Duration,
		reqParams url.Values,
	) (u *url.URL, err error)
	RemoveObject(
		ctx context.Context,
		bucketName string,
		objectName string,
		opts minio.RemoveObjectOptions,
	) error
}

type Minio struct {
	mc         Client
	bucketName string
}

func New(mc Client, bucketName string) *Minio {
	return &Minio{
		mc:         mc,
		bucketName: bucketName,
	}
}

const defaultImage = "avatar.png"

func (m *Minio) SaveAvatar(ctx context.Context, avatar *models.Avatar) (string, error) {
	const op = "storage.minio.SaveAvatar"

	if avatar == nil {
		url, err := m.GetAvatarUrl(ctx, defaultImage)
		if err != nil {
			return "", fmt.Errorf("%s: %w", op, err)
		}

		return url, nil
	}

	reader := bytes.NewReader(avatar.Data)

	_, err := m.mc.PutObject(
		ctx,
		m.bucketName,
		avatar.ID,
		reader,
		int64(len(avatar.Data)),
		minio.PutObjectOptions{ContentType: "image/png"},
	)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	url, err := m.GetAvatarUrl(ctx, avatar.ID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return url, nil
}

func (m *Minio) GetAvatarUrl(ctx context.Context, avatarId string) (string, error) {
	const op = "storage.minio.GetAvatar"

	url, err := m.mc.PresignedGetObject(ctx, m.bucketName, avatarId, 5*24*time.Hour, nil)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return url.String(), nil
}

func (m *Minio) DeleteImage(ctx context.Context, avatarId string) error {
	const op = "storage.minio.DeleteImage"

	err := m.mc.RemoveObject(ctx, m.bucketName, avatarId, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
