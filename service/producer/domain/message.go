package domain

import (
	"time"

	"cloud.google.com/go/datastore"
)

const MessageKind = "Message"

type Message struct {
	ID        *datastore.Key `datastore:"__key__"`
	Name      string
	Message   string
	CreatedAt time.Time
}

func NewMessage(name string, message string) *Message {
	key := datastore.IDKey(MessageKind, 0, nil)
	return &Message{
		ID:        key,
		Name:      name,
		Message:   message,
		CreatedAt: time.Now(),
	}
}
