package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AlexMickh/speak-chat/internal/models"
	"github.com/AlexMickh/speak-chat/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Postgres interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Storage struct {
	db Postgres
}

func New(db Postgres) *Storage {
	return &Storage{
		db: db,
	}
}

// TODO: add pgx errors

func (s *Storage) SaveChat(
	ctx context.Context,
	id string,
	name string,
	description string,
	chatImageUrl string,
	imageExireTime time.Time,
	chatOwnerId string,
) error {
	const op = "storage.postgres.SaveChat"

	sql := `INSERT INTO chat.chats
			(id, name, description, owner_id, chat_image_url, participants_id, image_expire_time)
			VALUES ($1, $2, $3, $4, $5, ARRAY[$6], $7)`
	_, err := s.db.Exec(ctx, sql, id, name, description, chatOwnerId, chatImageUrl, chatOwnerId, imageExireTime)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return fmt.Errorf("%s: %w", op, storage.ErrChatAlreadyExists)
			}
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) GetChat(ctx context.Context, id string) (models.Chat, error) {
	const op = "storage.postgres.GetChat"

	var chat models.Chat
	sqlStr := `SELECT id, name, description, chat_image_url, owner_id, participants_id, image_expire_time
			FROM chat.chats
			WHERE id = $1`
	err := s.db.QueryRow(ctx, sqlStr, id).Scan(
		&chat.ID,
		&chat.Name,
		&chat.Description,
		&chat.ChatImageUrl,
		&chat.ChatOwnerId,
		&chat.ParticipantsId,
		&chat.ImageExpireTime,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Chat{}, fmt.Errorf("%s: %w", op, storage.ErrChatNotFound)
		}
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	return chat, nil
}

func (s *Storage) GetAllUserChats(ctx context.Context, userId string) ([]models.ChatPreview, error) {
	const op = "storage.postgres.GetAllUserChats"

	sql := `SELECT id, name, chat_image_url, image_expire_time
			FROM chat.chats
			WHERE $1 = ANY(participants_id)`
	rows, err := s.db.Query(ctx, sql, userId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var chats []models.ChatPreview
	for rows.Next() {
		var chat models.ChatPreview

		err = rows.Scan(&chat.ID, &chat.Name, &chat.ChatImageUrl, &chat.ImageExpireTime)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		chats = append(chats, chat)
	}

	if len(chats) == 0 {
		return nil, fmt.Errorf("%s: %w", op, storage.ErrChatNotFound)
	}

	return chats, nil
}

func (s *Storage) UpdateImageUrl(
	ctx context.Context,
	chatId string,
	chatImageUrl string,
	imageExireTime time.Time,
) (models.Chat, error) {
	const op = "storage.postgres.UpdateImageUrl"

	var chat models.Chat
	sql := `UPDATE chat.chats 
			SET chat_image_url = $1, image_expire_time = $2
			WHERE id = $3
			RETURNING id, name, description, chat_image_url, chat_owner_id, participants_id, image_expire_time`
	err := s.db.QueryRow(ctx, sql, chatImageUrl, imageExireTime, chatId).Scan(
		&chat.ID,
		&chat.Name,
		&chat.Description,
		&chat.ChatImageUrl,
		&chat.ChatOwnerId,
		&chat.ParticipantsId,
		&chat.ImageExpireTime,
	)
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	return chat, nil
}

func (s *Storage) UpdateChatInfo(
	ctx context.Context,
	userId string,
	chatId string,
	name string,
	description string,
	chatImageUrl string,
	imageExireTime time.Time,
) (models.Chat, error) {
	const op = "storage.postgres.UpdateChatInfo"

	var sb strings.Builder

	_, err := sb.WriteString("UPDATE chat.chats ")
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	counter := 1
	args := make([]any, 0, 6)

	if name != "" {
		_, err = sb.WriteString(fmt.Sprintf("SET name = $%d", counter))
		if err != nil {
			return models.Chat{}, fmt.Errorf("%s: %w", op, err)
		}

		counter++
		args = append(args, name)
	}
	if description != "" {
		if counter == 1 {
			_, err = sb.WriteString(fmt.Sprintf("SET description = $%d", counter))
			if err != nil {
				return models.Chat{}, fmt.Errorf("%s: %w", op, err)
			}
		} else {
			_, err = sb.WriteString(fmt.Sprintf(", description = $%d", counter))
			if err != nil {
				return models.Chat{}, fmt.Errorf("%s: %w", op, err)
			}
		}
		counter++
		args = append(args, description)
	}
	if chatImageUrl != "" {
		if counter == 1 {
			_, err = sb.WriteString(
				fmt.Sprintf("SET chat_image_url = $%d, image_expire_time = $%d", counter, counter+1),
			)
			if err != nil {
				return models.Chat{}, fmt.Errorf("%s: %w", op, err)
			}
		} else {
			_, err = sb.WriteString(
				fmt.Sprintf(", chat_image_url = $%d, image_expire_time = $%d", counter, counter+1),
			)
			if err != nil {
				return models.Chat{}, fmt.Errorf("%s: %w", op, err)
			}
		}
		counter += 2
		args = append(args, chatImageUrl)
		args = append(args, imageExireTime)
	}

	_, err = sb.WriteString(
		fmt.Sprintf(` WHERE id = $%d AND owner_id = $%d 
					 RETURNING id, name, description, chat_image_url, owner_id, participants_id, image_expire_time`,
			counter, counter+1),
	)
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	args = append(args, chatId)
	args = append(args, userId)

	var chat models.Chat
	err = s.db.QueryRow(ctx, sb.String(), args...).Scan(
		&chat.ID,
		&chat.Name,
		&chat.Description,
		&chat.ChatImageUrl,
		&chat.ChatOwnerId,
		&chat.ParticipantsId,
		&chat.ImageExpireTime,
	)
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	return chat, nil
}

func (s *Storage) AddParticipant(
	ctx context.Context,
	userId string,
	chatId string,
	participantId string,
) error {
	const op = "storage.postgres.AddParticipant"

	sql := `UPDATE chat.chats
			SET participants_id = array_append(participants_id, $1)
			WHERE id = $2 AND owner_id = $3`
	_, err := s.db.Exec(ctx, sql, participantId, chatId, userId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) DeleteChat(ctx context.Context, userId, chatId string, ch chan error) {
	const op = "storage.postgres.DeleteChat"

	sql := "DELETE FROM chat.chats WHERE id = $1 AND owner_id = $2"
	_, err := s.db.Exec(ctx, sql, chatId, userId)
	if err != nil {
		ch <- fmt.Errorf("%s: %w", op, err)
	}

	ch <- nil
}
