package models

import "time"

type Chat struct {
	ID             string
	Name           string
	Description    string
	ChatImageUrl   string
	ChatOwnerId    string
	ParticipantsId []string
}

type Message struct {
	ID        string
	SenderId  string
	Message   string
	CreatedAt time.Time
}

type Avatar struct {
	ID   string
	Data []byte
}
