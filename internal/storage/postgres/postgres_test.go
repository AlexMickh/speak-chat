package postgres

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/AlexMickh/speak-chat/internal/models"
	"github.com/AlexMickh/speak-chat/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestStorage_SaveChat(t *testing.T) {
	type fields struct {
		db Postgres
	}
	type args struct {
		ctx            context.Context
		id             string
		name           string
		description    string
		chatImageUrl   string
		imageExireTime time.Time
		chatOwnerId    string
	}

	pool := initStorage()
	defer pool.Close()
	id := uuid.NewString()

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "good case",
			fields: fields{
				db: pool,
			},
			args: args{
				ctx:            context.Background(),
				id:             id,
				name:           "chat",
				description:    "chat",
				chatImageUrl:   "chat",
				imageExireTime: time.Now().Add(24 * time.Hour),
				chatOwnerId:    uuid.NewString(),
			},
			wantErr: nil,
		},
		{
			name: "not unique id",
			fields: fields{
				db: pool,
			},
			args: args{
				ctx:            context.Background(),
				id:             id,
				name:           "chat",
				description:    "chat",
				chatImageUrl:   "chat",
				imageExireTime: time.Now().Add(24 * time.Hour),
				chatOwnerId:    uuid.NewString(),
			},
			wantErr: storage.ErrChatAlreadyExists,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{
				db: tt.fields.db,
			}
			if err := s.SaveChat(
				tt.args.ctx,
				tt.args.id,
				tt.args.name,
				tt.args.description,
				tt.args.chatImageUrl,
				tt.args.imageExireTime,
				tt.args.chatOwnerId,
			); err != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Storage.SaveChat() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
	for _, tt := range tests {
		_, _ = pool.Exec(context.Background(), "DELETE FROM chat.chats WHERE id = $1", tt.args.id)
	}
}

func TestStorage_GetChat(t *testing.T) {
	type fields struct {
		db Postgres
	}
	type args struct {
		ctx context.Context
		id  string
	}

	pool := initStorage()
	defer pool.Close()

	chat := models.Chat{
		ID:              uuid.NewString(),
		Name:            "chat",
		Description:     "chat",
		ChatImageUrl:    "eflrkdnhl",
		ImageExpireTime: time.Time{},
		ChatOwnerId:     uuid.NewString(),
		ParticipantsId:  []string{uuid.NewString(), uuid.NewString(), uuid.NewString()},
	}

	_, err := pool.Exec(
		context.Background(),
		`INSERT INTO chat.chats
		 (id, name, description, owner_id, chat_image_url, participants_id, image_expire_time)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		chat.ID, chat.Name, chat.Description, chat.ChatOwnerId, chat.ChatImageUrl, chat.ParticipantsId, chat.ImageExpireTime,
	)
	if err != nil {
		fmt.Printf("err: %v", err)
		return
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    models.Chat
		wantErr error
	}{
		{
			name: "good case",
			fields: fields{
				db: pool,
			},
			args: args{
				ctx: context.Background(),
				id:  chat.ID,
			},
			want:    chat,
			wantErr: nil,
		},
		{
			name: "id does not exists",
			fields: fields{
				db: pool,
			},
			args: args{
				ctx: context.Background(),
				id:  uuid.NewString(),
			},
			want:    models.Chat{},
			wantErr: storage.ErrChatNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{
				db: tt.fields.db,
			}
			got, err := s.GetChat(tt.args.ctx, tt.args.id)
			if err != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Storage.GetChat() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Storage.GetChat() = %#v, want %#v", got, tt.want)
			}
		})
	}
	for _, tt := range tests {
		_, _ = pool.Exec(context.Background(), "DELETE FROM chat.chats WHERE id = $1", tt.args.id)
	}
}

func TestStorage_GetAllUserChats(t *testing.T) {
	type fields struct {
		db Postgres
	}
	type args struct {
		ctx    context.Context
		userId string
	}

	pool := initStorage()
	defer pool.Close()

	userId := uuid.NewString()

	chats := []models.ChatPreview{
		{
			ID:              uuid.NewString(),
			Name:            "chat1",
			ChatImageUrl:    "gsserga",
			ImageExpireTime: time.Time{},
		},
		{
			ID:              uuid.NewString(),
			Name:            "chat2",
			ChatImageUrl:    "",
			ImageExpireTime: time.Time{},
		},
	}
	for _, chat := range chats {
		_, err := pool.Exec(
			context.Background(),
			"INSERT INTO chat.chats (id, name, chat_image_url, image_expire_time, participants_id) VALUES ($1, $2, $3, $4, ARRAY[$5])",
			chat.ID, chat.Name, chat.ChatImageUrl, chat.ImageExpireTime, userId,
		)
		if err != nil {
			fmt.Printf("err: %v", err)
			return
		}
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []models.ChatPreview
		wantErr error
	}{
		{
			name: "good case",
			fields: fields{
				db: pool,
			},
			args: args{
				ctx:    context.Background(),
				userId: userId,
			},
			want:    chats,
			wantErr: storage.ErrChatNotFound,
		},
		{
			name: "id does not exists",
			fields: fields{
				db: pool,
			},
			args: args{
				ctx:    context.Background(),
				userId: uuid.NewString(),
			},
			want:    nil,
			wantErr: storage.ErrChatNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{
				db: tt.fields.db,
			}
			got, err := s.GetAllUserChats(tt.args.ctx, tt.args.userId)
			if err != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Storage.GetAllUserChats() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Storage.GetAllUserChats() = %v, want %v", got, tt.want)
			}
		})
	}
	for i, tt := range tests {
		_, _ = pool.Exec(context.Background(), "DELETE FROM chat.chats WHERE id = $1", tt.want[i].ID)
	}
}

func TestStorage_UpdateChatInfo(t *testing.T) {
	type fields struct {
		db Postgres
	}
	type args struct {
		ctx            context.Context
		userId         string
		chatId         string
		name           string
		description    string
		chatImageUrl   string
		imageExireTime time.Time
	}

	pool := initStorage()
	defer pool.Close()

	chat := models.Chat{
		ID:              uuid.NewString(),
		Name:            "chat",
		Description:     "chat",
		ChatImageUrl:    "eflrkdnhl",
		ImageExpireTime: time.Time{},
		ChatOwnerId:     uuid.NewString(),
		ParticipantsId:  []string{uuid.NewString(), uuid.NewString(), uuid.NewString()},
	}

	sql := `INSERT INTO chat.chats
			(id, name, description, owner_id, chat_image_url, participants_id, image_expire_time)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := pool.Exec(
		context.Background(),
		sql,
		chat.ID,
		chat.Name,
		chat.Description,
		chat.ChatOwnerId,
		chat.ChatImageUrl,
		chat.ParticipantsId,
		chat.ImageExpireTime,
	)
	if err != nil {
		fmt.Printf("err: %v", err)
		return
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    models.Chat
		wantErr error
	}{
		{
			name: "good case for changing name",
			fields: fields{
				db: pool,
			},
			args: args{
				ctx:            context.Background(),
				userId:         chat.ChatOwnerId,
				chatId:         chat.ID,
				name:           "changed",
				description:    "",
				chatImageUrl:   "",
				imageExireTime: time.Time{},
			},
			want: models.Chat{
				ID:              chat.ID,
				Name:            "changed",
				Description:     chat.Description,
				ChatImageUrl:    chat.ChatImageUrl,
				ImageExpireTime: chat.ImageExpireTime,
				ChatOwnerId:     chat.ChatOwnerId,
				ParticipantsId:  chat.ParticipantsId,
			},
			wantErr: nil,
		},
		{
			name: "good case for changing description",
			fields: fields{
				db: pool,
			},
			args: args{
				ctx:            context.Background(),
				userId:         chat.ChatOwnerId,
				chatId:         chat.ID,
				name:           "",
				description:    "changed",
				chatImageUrl:   "",
				imageExireTime: time.Time{},
			},
			want: models.Chat{
				ID:              chat.ID,
				Name:            "changed",
				Description:     "changed",
				ChatImageUrl:    chat.ChatImageUrl,
				ImageExpireTime: chat.ImageExpireTime,
				ChatOwnerId:     chat.ChatOwnerId,
				ParticipantsId:  chat.ParticipantsId,
			},
			wantErr: nil,
		},
		{
			name: "good case for changing chat image url",
			fields: fields{
				db: pool,
			},
			args: args{
				ctx:            context.Background(),
				userId:         chat.ChatOwnerId,
				chatId:         chat.ID,
				name:           "",
				description:    "",
				chatImageUrl:   "changed",
				imageExireTime: time.Time{},
			},
			want: models.Chat{
				ID:              chat.ID,
				Name:            "changed",
				Description:     "changed",
				ChatImageUrl:    "changed",
				ImageExpireTime: time.Time{},
				ChatOwnerId:     chat.ChatOwnerId,
				ParticipantsId:  chat.ParticipantsId,
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{
				db: tt.fields.db,
			}
			got, err := s.UpdateChatInfo(
				tt.args.ctx,
				tt.args.userId,
				tt.args.chatId,
				tt.args.name,
				tt.args.description,
				tt.args.chatImageUrl,
				tt.args.imageExireTime,
			)
			if err != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Storage.UpdateChatInfo() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Storage.UpdateChatInfo() = %#v, want %#v", got, tt.want)
			}
		})
	}
	for _, tt := range tests {
		_, _ = pool.Exec(context.Background(), "DELETE FROM chat.chats WHERE id = $1", tt.args.chatId)
	}
}

func initStorage() *pgxpool.Pool {
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable&pool_max_conns=%d&pool_min_conns=%d",
		"postgres",
		"root",
		"localhost",
		5422,
		"chat",
		3,
		5,
	)

	pool, _ := pgxpool.New(context.Background(), connString)

	return pool
}
