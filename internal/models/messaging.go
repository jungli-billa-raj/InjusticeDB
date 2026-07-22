package models

import (
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID        uuid.UUID `json:"id"`
	UserOneID uuid.UUID `json:"user_one_id"`
	UserTwoID uuid.UUID `json:"user_two_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID             uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	SenderID       uuid.UUID `json:"sender_id"`
	Content        string    `json:"content"`
	IsRead         bool      `json:"is_read"`
	SentAt         time.Time `json:"sent_at"`
}
