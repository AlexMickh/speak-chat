package postgres

import (
	"context"
	"fmt"

	"github.com/AlexMickh/speak-chat/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Postgres interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

type Storage struct {
	db Postgres
}

func New(db Postgres) *Storage {
	return &Storage{
		db: db,
	}
}

func (s *Storage) SaveChat(
	ctx context.Context,
	name string,
	description string,
	chatImageUrl string,
	chatOwnerId string,
) (string, error) {
	const op = "storage.postgres.SaveChat"

	var chatId string
	sql := `INSERT INTO chat.chats
			(name, description, owner_id, chat_image, participants_id)
			VALUES ($1, $2, $3, $4, ARRAY[$5])
			RETURNING id`
	err := s.db.QueryRow(ctx, sql, name, description, chatOwnerId, chatImageUrl, chatOwnerId).Scan(&chatId)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return chatId, nil
}

func (s *Storage) GetChat(ctx context.Context, id string) (models.Chat, error) {
	const op = "storage.postgres.GetChat"

	var chat models.Chat
	sql := `SELECT id, name, description, chat_image_url, chat_owner_id, participants_id
			FROM chat.chats
			WHERE id = $1`
	err := s.db.QueryRow(ctx, sql, id).Scan(
		&chat.ID,
		&chat.Name,
		&chat.Description,
		&chat.ChatImageUrl,
		&chat.ChatOwnerId,
		&chat.ParticipantsId,
	)
	if err != nil {
		return models.Chat{}, fmt.Errorf("%s: %w", op, err)
	}

	return chat, nil
}

func (s *Storage) AddParticipant(ctx context.Context, chatId, participantsId string) error {
	const op = "storage.postgres.AddParticipant"

	sql := `UPDATE chat.chats
			SET participants_id = array_append(participants_id, $1)
			WHERE id = $2`
	_, err := s.db.Exec(ctx, sql, participantsId, chatId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) SaveMessage(
	ctx context.Context,
	senderId string,
	chatId string,
	message string,
) error {
	const op = "storage.postgres.SaveMessage"

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			_ = tx.Commit(ctx)
		}
	}()

	var messageId string
	sql := `INSERT INTO chat.messages
			(message, sender_id)
			VALUES ($1, $2)
			RETURNING id`
	err = tx.QueryRow(ctx, sql, message, senderId).Scan(&messageId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	sql = `UPDATE chat.chats
		   SET messages_id = array_append(messages_id, $1)
		   WHERE id = $2`
	_, err = tx.Exec(ctx, sql, messageId, chatId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) GetMessages(ctx context.Context, chatId string, from int, to int) ([]models.Message, error) {
	const op = "storage.postgres.GetMessages"

	sql := `SELECT id, sender_id, message, created_at
			FROM chat.messages
			WHERE id IN (
				SELECT unnest(messages_id[$1:$2])
				FROM chat.chats
				WHERE id = $3
			)
    		ORDER BY created_at`
	rows, err := s.db.Query(ctx, sql, from, to, chatId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var message models.Message
		err = rows.Scan(
			&message.ID,
			&message.SenderId,
			&message.Message,
			&message.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		messages = append(messages, message)
	}

	return messages, nil
}
