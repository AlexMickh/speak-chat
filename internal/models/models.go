package models

import (
	"time"
)

type Chat struct {
	ID              string    `redis:"id"`
	Name            string    `redis:"name"`
	Description     string    `redis:"description"`
	ChatImageUrl    string    `redis:"chat_image_url"`
	ImageExpireTime time.Time `redis:"image_expire_time"`
	ChatOwnerId     string    `redis:"chat_owner_id"`
	ParticipantsId  []string  `redis:"-"`
}

type ChatPreview struct {
	ID              string    `redis:"id"`
	Name            string    `redis:"name"`
	ChatImageUrl    string    `redis:"chat_image_url"`
	ImageExpireTime time.Time `redis:"image_expire_time"`
}

type Avatar struct {
	ID   string
	Data []byte
}
