package storage

import "errors"

var (
	ErrChatAlreadyExists = errors.New("chat already exists")
	ErrChatNotFound      = errors.New("chat with this id does not found")
)
