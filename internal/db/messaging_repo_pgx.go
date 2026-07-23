package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

type PostgresMessagingRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresMessagingRepository(pool *pgxpool.Pool) *PostgresMessagingRepository {
	return &PostgresMessagingRepository{pool: pool}
}

// GetOrCreateConversation retrieves an existing chat space between two users or creates a new one.
func (r *PostgresMessagingRepository) GetOrCreateConversation(ctx context.Context, currentUserID, targetUserID uuid.UUID) (*models.Conversation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Set RLS user context for this transaction
	_, err = tx.Exec(ctx, "SELECT set_config('app.current_user_id', $1, true)", currentUserID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to set RLS user context: %w", err)
	}

	// Order user IDs deterministically to respect UNIQUE(user_one_id, user_two_id) constraint
	u1, u2 := currentUserID, targetUserID
	if u1.String() > u2.String() {
		u1, u2 = u2, u1
	}

	query := `
		INSERT INTO conversations (user_one_id, user_two_id)
		VALUES ($1, $2)
		ON CONFLICT (user_one_id, user_two_id) 
		DO UPDATE SET user_one_id = EXCLUDED.user_one_id
		RETURNING id, user_one_id, user_two_id, created_at;
	`

	var conv models.Conversation
	err = tx.QueryRow(ctx, query, u1, u2).Scan(
		&conv.ID,
		&conv.UserOneID,
		&conv.UserTwoID,
		&conv.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create conversation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &conv, nil
}

// SendMessage delivers a new private message within a conversation.
func (r *PostgresMessagingRepository) SendMessage(ctx context.Context, conversationID, senderID uuid.UUID, content string) (*models.Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Set RLS user context for this transaction
	_, err = tx.Exec(ctx, "SELECT set_config('app.current_user_id', $1, true)", senderID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to set RLS user context: %w", err)
	}

	query := `
		INSERT INTO messages (conversation_id, sender_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, conversation_id, sender_id, content, is_read, sent_at;
	`

	var msg models.Message
	err = tx.QueryRow(ctx, query, conversationID, senderID, content).Scan(
		&msg.ID,
		&msg.ConversationID,
		&msg.SenderID,
		&msg.Content,
		&msg.IsRead,
		&msg.SentAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send message (RLS check failed or conv not found): %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &msg, nil
}

// GetMessages retrieves recent messages for a conversation, enforced by RLS.
func (r *PostgresMessagingRepository) GetMessages(ctx context.Context, requestingUserID, conversationID uuid.UUID, limit, offset int) ([]*models.Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Set RLS user context for this transaction
	_, err = tx.Exec(ctx, "SELECT set_config('app.current_user_id', $1, true)", requestingUserID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to set RLS user context: %w", err)
	}

	query := `
		SELECT id, conversation_id, sender_id, content, is_read, sent_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY sent_at ASC
		LIMIT $2 OFFSET $3;
	`

	rows, err := tx.Query(ctx, query, conversationID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		var m models.Message
		err := rows.Scan(
			&m.ID,
			&m.ConversationID,
			&m.SenderID,
			&m.Content,
			&m.IsRead,
			&m.SentAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		messages = append(messages, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return messages, nil
}

// ListConversations fetches all active DM conversations for a user.
func (r *PostgresMessagingRepository) ListConversations(ctx context.Context, userID uuid.UUID) ([]*models.Conversation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "SELECT set_config('app.current_user_id', $1, true)", userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to set RLS user context: %w", err)
	}

	query := `
		SELECT id, user_one_id, user_two_id, created_at
		FROM conversations
		ORDER BY created_at DESC;
	`

	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	var convs []*models.Conversation
	for rows.Next() {
		var c models.Conversation
		err := rows.Scan(&c.ID, &c.UserOneID, &c.UserTwoID, &c.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		convs = append(convs, &c)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return convs, nil
}
