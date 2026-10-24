package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/teslacost/teslacost/internal/models"
)

// CreateAPIToken stores a hashed API token.
func (r *Repository) CreateAPIToken(ctx context.Context, userID, name, scope, tokenHash, prefix string, expiresAt *time.Time) (*models.APIToken, error) {
	query := `
		INSERT INTO api_tokens (user_id, name, scope, token_hash, token_prefix, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at;
	`
	tok := &models.APIToken{
		UserID:      userID,
		Name:        name,
		TokenPrefix: prefix,
		Scope:       scope,
		ExpiresAt:   expiresAt,
	}
	err := r.pool.QueryRow(ctx, query, userID, name, scope, tokenHash, prefix, expiresAt).Scan(&tok.ID, &tok.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create api token: %w", err)
	}
	return tok, nil
}

// ListAPITokens returns all tokens for a user.
func (r *Repository) ListAPITokens(ctx context.Context, userID string) ([]models.APIToken, error) {
	query := `
		SELECT id, user_id, name, token_prefix, scope, last_used_at,
		       CASE WHEN oauth_client_id IS NULL THEN expires_at ELSE refresh_expires_at END,
		       created_at, oauth_client_id IS NOT NULL, vehicle_ids::text[]
		FROM api_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC;
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list api tokens: %w", err)
	}
	defer rows.Close()

	var list []models.APIToken
	for rows.Next() {
		var tok models.APIToken
		if err := rows.Scan(
			&tok.ID, &tok.UserID, &tok.Name, &tok.TokenPrefix, &tok.Scope,
			&tok.LastUsedAt, &tok.ExpiresAt, &tok.CreatedAt, &tok.OAuth, &tok.VehicleIDs,
		); err != nil {
			return nil, err
		}
		list = append(list, tok)
	}
	if list == nil {
		list = []models.APIToken{}
	}
	return list, rows.Err()
}

// RevokeAPIToken deletes a token.
func (r *Repository) RevokeAPIToken(ctx context.Context, userID, tokenID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM api_tokens WHERE id = $1 AND user_id = $2;`, tokenID, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke api token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ValidateAPIToken verifies a token hash and returns the associated user's ID and email, and the token's scope.
func (r *Repository) ValidateAPIToken(ctx context.Context, tokenHash string) (userID, email, scope string, err error) {
	g, err := r.ValidateAPITokenGrant(ctx, tokenHash)
	if err != nil {
		return "", "", "", err
	}
	return g.UserID, g.Email, g.Scope, nil
}

// ValidateAPITokenGrant verifies a token hash and returns what the token lets its holder do: the account, the scope
// and the vehicles it is limited to.
func (r *Repository) ValidateAPITokenGrant(ctx context.Context, tokenHash string) (models.TokenGrant, error) {
	query := `
		SELECT t.id, t.user_id, u.email, t.scope, t.vehicle_ids::text[]
		FROM api_tokens t
		JOIN users u ON u.id = t.user_id
		WHERE t.token_hash = $1 AND (t.expires_at IS NULL OR t.expires_at > NOW());
	`
	var g models.TokenGrant
	if err := r.pool.QueryRow(ctx, query, tokenHash).Scan(&g.TokenID, &g.UserID, &g.Email, &g.Scope, &g.VehicleIDs); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.TokenGrant{}, ErrNotFound
		}
		return models.TokenGrant{}, err
	}

	// Update last_used_at asynchronously
	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = r.pool.Exec(updateCtx, `UPDATE api_tokens SET last_used_at = NOW() WHERE id = $1;`, g.TokenID)
	}()

	return g, nil
}

// apiTokenCallsKept is how many calls of a token the log holds; a newer call pushes the oldest out.
const apiTokenCallsKept = 50

// RecordAPITokenCall appends a call to the log of a token and drops the ones beyond the latest apiTokenCallsKept.
func (r *Repository) RecordAPITokenCall(ctx context.Context, tokenID string, c models.APITokenCall) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO api_token_calls (token_id, tool, vehicle_id, outcome, duration_ms)
		VALUES ($1, $2, $3, $4, $5);`, tokenID, c.Tool, c.VehicleID, c.Outcome, c.DurationMS); err != nil {
		return fmt.Errorf("failed to record api token call: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM api_token_calls
		WHERE token_id = $1 AND id NOT IN (SELECT id FROM api_token_calls WHERE token_id = $1 ORDER BY id DESC LIMIT $2);`,
		tokenID, apiTokenCallsKept); err != nil {
		return fmt.Errorf("failed to trim api token calls: %w", err)
	}
	return tx.Commit(ctx)
}

// ListAPITokenCalls returns the logged calls of one of the user's tokens, most recent first. A token of another account
// is not found.
func (r *Repository) ListAPITokenCalls(ctx context.Context, userID, tokenID string) ([]models.APITokenCall, error) {
	var owned bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM api_tokens WHERE id = $1 AND user_id = $2);`, tokenID, userID).Scan(&owned); err != nil {
		return nil, fmt.Errorf("failed to look up api token: %w", err)
	}
	if !owned {
		return nil, ErrNotFound
	}
	rows, err := r.pool.Query(ctx, `
		SELECT called_at, tool, vehicle_id::text, outcome, duration_ms
		FROM api_token_calls WHERE token_id = $1 ORDER BY id DESC;`, tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to list api token calls: %w", err)
	}
	defer rows.Close()
	list := []models.APITokenCall{}
	for rows.Next() {
		var c models.APITokenCall
		if err := rows.Scan(&c.At, &c.Tool, &c.VehicleID, &c.Outcome, &c.DurationMS); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}
