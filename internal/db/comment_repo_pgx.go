package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

type PostgresCommentRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresCommentRepository(pool *pgxpool.Pool) *PostgresCommentRepository {
	return &PostgresCommentRepository{pool: pool}
}

// CreateComment posts a new comment on an incident.
func (r *PostgresCommentRepository) CreateComment(ctx context.Context, incidentID, userID uuid.UUID, content string) (*models.Comment, error) {
	query := `
		INSERT INTO comments (incident_id, user_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, incident_id, user_id, content, created_at;
	`

	var comment models.Comment
	err := r.pool.QueryRow(ctx, query, incidentID, userID, content).Scan(
		&comment.ID,
		&comment.IncidentID,
		&comment.UserID,
		&comment.Content,
		&comment.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	return &comment, nil
}

// ListCommentsByIncident fetches comments for an incident with author details.
func (r *PostgresCommentRepository) ListCommentsByIncident(ctx context.Context, incidentID uuid.UUID, limit, offset int) ([]*models.Comment, error) {
	query := `
		SELECT c.id, c.incident_id, c.user_id, c.parent_id, c.content, c.created_at, u.name
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.incident_id = $1
		ORDER BY c.created_at ASC
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.pool.Query(ctx, query, incidentID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query comments: %w", err)
	}
	defer rows.Close()

	var allComments []*models.Comment
	for rows.Next() {
		var c models.Comment
		err := rows.Scan(
			&c.ID,
			&c.IncidentID,
			&c.UserID,
			&c.ParentID, // Scan nullable parent_id
			&c.Content,
			&c.CreatedAt,
			&c.UserName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment row: %w", err)
		}
		allComments = append(allComments, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Build the nested tree structure
	return buildCommentTree(allComments), nil
}

// Helper function to convert flat slice into a nested tree
func buildCommentTree(flat []*models.Comment) []*models.Comment {
	commentMap := make(map[uuid.UUID]*models.Comment)
	var rootComments []*models.Comment

	// Step 1: Map all comments by ID and initialize empty Replies slice
	for _, c := range flat {
		c.Replies = []*models.Comment{}
		commentMap[c.ID] = c
	}

	// Step 2: Attach children to parents
	for _, c := range flat {
		if c.ParentID == nil {
			// Top-level comment
			rootComments = append(rootComments, c)
		} else {
			// Reply -> Attach to parent if parent exists in map
			if parent, exists := commentMap[*c.ParentID]; exists {
				parent.Replies = append(parent.Replies, c)
			}
		}
	}

	return rootComments
}
