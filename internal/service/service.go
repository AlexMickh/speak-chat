package service

import (
	"context"
	"fmt"
	"time"

	"github.com/AlexMickh/speak-chat/internal/models"
	"github.com/google/uuid"
)

type Storage interface {
	SaveChat(
		ctx context.Context,
		id string,
		name string,
		description string,
		chatImageUrl string,
		imageExireTime time.Time,
		chatOwnerId string,
	) error
	GetChat(ctx context.Context, id string) (models.Chat, error)
	AddParticipant(
		ctx context.Context,
		userId string,
		chatId string,
		participantId string,
	) error
	UpdateChatInfo(
		ctx context.Context,
		userId string,
		chatId string,
		name string,
		description string,
		chatImageUrl string,
		imageExireTime time.Time,
	) (models.Chat, error)
	UpdateImageUrl(
		ctx context.Context,
		chatId string,
		chatImageUrl string,
		imageExireTime time.Time,
	) (models.Chat, error)
	DeleteChat(ctx context.Context, userId, chatId string, ch chan error)
}

type Cash interface {
	SaveChat(ctx context.Context, chat models.Chat) error
	GetChat(ctx context.Context, id string) (models.Chat, error)
	UpdateChat(ctx context.Context, chat models.Chat) error
	AddParticipant(ctx context.Context, chatId, participantId string) error
	DeleteChat(ctx context.Context, chatId string) error
}

type S3 interface {
	SaveAvatar(ctx context.Context, avatar *models.Avatar) (string, time.Time, error)
	GetAvatarUrl(ctx context.Context, avatarId string) (string, time.Time, error)
	UpdateAvatar(ctx context.Context, avatar models.Avatar) (string, time.Time, error)
	DeleteAvatar(ctx context.Context, avatarId string) (string, time.Time, error)
}

type Service struct {
	storage Storage
	cash    Cash
	s3      S3
}

func New(storage Storage, cash Cash, s3 S3) *Service {
	return &Service{
		storage: storage,
		cash:    cash,
		s3:      s3,
	}
}

func (s *Service) CreateChat(
	ctx context.Context,
	name string,
	description string,
	avatar []byte,
	chatOwnerId string,
) (string, error) {
	const op = "service.CreateChat"

	id := uuid.NewString()
	var chatImageUrl string
	var expires time.Time
	var err error
	if avatar != nil {
		avatarStruct := &models.Avatar{
			ID:   id,
			Data: avatar,
		}

		chatImageUrl, expires, err = s.s3.SaveAvatar(ctx, avatarStruct)
	} else {
		chatImageUrl, expires, err = s.s3.SaveAvatar(ctx, nil)
	}
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	err = s.storage.SaveChat(
		ctx,
		id,
		name,
		description,
		chatImageUrl,
		expires,
		chatOwnerId,
	)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	chat := models.Chat{
		ID:             id,
		Name:           name,
		Description:    description,
		ChatImageUrl:   chatImageUrl,
		ChatOwnerId:    chatOwnerId,
		ParticipantsId: []string{chatOwnerId},
	}
	err = s.cash.SaveChat(ctx, chat)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (s *Service) GetChat(ctx context.Context, id string) (models.Chat, error) {
	const op = "service.GetChat"

	chat, err := s.cash.GetChat(ctx, id)
	if err == nil {
		if s.isImageExpire(chat.ImageExpireTime) {
			chat.ChatImageUrl, chat.ImageExpireTime, err = s.updateImageUrl(ctx, chat.ID)
			if err != nil {
				return models.Chat{}, fmt.Errorf("%s: %w", op, err)
			}
		}

		return chat, nil
	}

	chat, err = s.storage.GetChat(ctx, id)
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}
	if s.isImageExpire(chat.ImageExpireTime) {
		chat.ChatImageUrl, chat.ImageExpireTime, err = s.updateImageUrl(ctx, chat.ID)
		if err != nil {
			return models.Chat{}, fmt.Errorf("%s: %w", op, err)
		}
	}

	err = s.cash.SaveChat(ctx, chat)
	if err != nil {
		fmt.Println(err)
	}

	return chat, nil
}

func (s *Service) AddParticipant(ctx context.Context, userId, chatId, participantId string) error {
	const op = "service.AddParticipant"

	err := s.storage.AddParticipant(ctx, userId, chatId, participantId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	err = s.cash.AddParticipant(ctx, chatId, participantId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Service) UpdateChatInfo(
	ctx context.Context,
	userId string,
	chatId string,
	name string,
	description string,
	avatar []byte,
) (models.Chat, error) {
	const op = "servive.UpdateChatInfo"

	var url string
	var expires time.Time
	var err error
	if avatar != nil {
		avatarStruct := models.Avatar{
			ID:   chatId,
			Data: avatar,
		}

		url, expires, err = s.s3.UpdateAvatar(ctx, avatarStruct)
		if err != nil {
			return models.Chat{}, nil
		}
	}

	chat, err := s.storage.UpdateChatInfo(
		ctx,
		userId,
		chatId,
		name,
		description,
		url,
		expires,
	)
	if err != nil {
		return models.Chat{}, nil
	}

	err = s.cash.UpdateChat(ctx, chat)
	if err != nil {
		return models.Chat{}, nil
	}

	return chat, nil
}

func (s *Service) DeleteChat(ctx context.Context, userId, chatId string) error {
	const op = "service.DeleteChat"

	ch := make(chan error)
	go s.storage.DeleteChat(ctx, userId, chatId, ch)

	err := s.cash.DeleteChat(ctx, chatId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	select {
	case err = <-ch:
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	return nil
}

func (s *Service) isImageExpire(expireTime time.Time) bool {
	if expireTime.Compare(time.Now()) != 1 {
		return false
	}
	return true
}

func (s *Service) updateImageUrl(ctx context.Context, chatId string) (string, time.Time, error) {
	const op = "service.updateImageUrl"

	url, expireTime, err := s.s3.GetAvatarUrl(ctx, chatId)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%s: %w", op, err)
	}

	_, err = s.storage.UpdateImageUrl(ctx, chatId, url, expireTime)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%s: %w", op, err)
	}

	return url, expireTime, nil
}
