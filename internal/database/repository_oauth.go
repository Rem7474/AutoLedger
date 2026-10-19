package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/teslacost/teslacost/internal/models"
)

// ErrTooManyOAuthClients is returned when the instance holds as many registered clients as it accepts.
var ErrTooManyOAuthClients = errors.New("too many registered OAuth clients")

// ErrOAuthRefreshReused is returned when a refresh token that was already renewed is presented again: the grant is
// revoked, since either the client or whoever copied the token is using a stale one.
var ErrOAuthRefreshReused = errors.New("OAuth refresh token reused")

// CreateOAuthClient stores a registered client, unless the instance already holds maxClients.
func (r *Repository) CreateOAuthClient(ctx context.Context, c *models.OAuthClient, maxClients int) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO oauth_clients (id, name, redirect_uris)
		SELECT $1, $2, $3 WHERE (SELECT COUNT(*) FROM oauth_clients) < $4;`,
		c.ID, c.Name, c.RedirectURIs, maxClients)
	if err != nil {
		return fmt.Errorf("failed to create oauth client: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTooManyOAuthClients
	}
	return nil
}

// GetOAuthClient returns a registered client.
func (r *Repository) GetOAuthClient(ctx context.Context, id string) (*models.OAuthClient, error) {
	var c models.OAuthClient
	err := r.pool.QueryRow(ctx, `SELECT id, name, redirect_uris, created_at FROM oauth_clients WHERE id = $1;`, id).
		Scan(&c.ID, &c.Name, &c.RedirectURIs, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// CreateOAuthCode stores the hash of an authorization code the user approved.
func (r *Repository) CreateOAuthCode(ctx context.Context, codeHash string, c models.OAuthCode, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO oauth_codes (code_hash, client_id, user_id, redirect_uri, code_challenge, scope, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7);`,
		codeHash, c.ClientID, c.UserID, c.RedirectURI, c.CodeChallenge, c.Scope, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to store oauth code: %w", err)
	}
	return nil
}

// ConsumeOAuthCode removes and returns an unexpired authorization code: a code works once, whatever happens next.
func (r *Repository) ConsumeOAuthCode(ctx context.Context, codeHash string) (*models.OAuthCode, error) {
	var c models.OAuthCode
	err := r.pool.QueryRow(ctx, `
		DELETE FROM oauth_codes WHERE code_hash = $1 AND expires_at > NOW()
		RETURNING client_id, user_id, redirect_uri, code_challenge, scope;`, codeHash).
		Scan(&c.ClientID, &c.UserID, &c.RedirectURI, &c.CodeChallenge, &c.Scope)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// OAuthGrant is what a token exchange stores: the hashes of the access and refresh tokens and their lifetimes.
type OAuthGrant struct {
	UserID, ClientID, Name, Scope string
	AccessHash, AccessPrefix      string
	AccessExpiresAt               time.Time
	RefreshHash                   string
	RefreshExpiresAt              time.Time
}

// IssueOAuthGrant stores a new grant. A client holds one grant per user: authorizing again replaces the earlier one.
func (r *Repository) IssueOAuthGrant(ctx context.Context, g OAuthGrant) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM api_tokens WHERE user_id = $1 AND oauth_client_id = $2;`, g.UserID, g.ClientID); err != nil {
		return fmt.Errorf("failed to replace oauth grant: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO api_tokens (user_id, name, scope, token_hash, token_prefix, expires_at, oauth_client_id, refresh_hash, refresh_expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);`,
		g.UserID, g.Name, g.Scope, g.AccessHash, g.AccessPrefix, g.AccessExpiresAt, g.ClientID, g.RefreshHash, g.RefreshExpiresAt); err != nil {
		return fmt.Errorf("failed to store oauth grant: %w", err)
	}
	return tx.Commit(ctx)
}

// RotateOAuthGrant replaces the access and refresh tokens of the grant the presented refresh token belongs to, and
// returns the grant's scope. ErrNotFound means no live grant of the client holds that token; ErrOAuthRefreshReused
// means it held it before the last renewal, and the grant is revoked.
func (r *Repository) RotateOAuthGrant(ctx context.Context, clientID, presentedRefreshHash string, next OAuthGrant) (scope string, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id, currentHash string
	var refreshExpires time.Time
	err = tx.QueryRow(ctx, `
		SELECT id, refresh_hash, refresh_expires_at, scope FROM api_tokens
		WHERE oauth_client_id = $1 AND (refresh_hash = $2 OR prev_refresh_hash = $2)
		FOR UPDATE;`, clientID, presentedRefreshHash).Scan(&id, &currentHash, &refreshExpires, &scope)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if currentHash != presentedRefreshHash {
		if _, err := tx.Exec(ctx, `DELETE FROM api_tokens WHERE id = $1;`, id); err != nil {
			return "", err
		}
		if err := tx.Commit(ctx); err != nil {
			return "", err
		}
		return "", ErrOAuthRefreshReused
	}
	if !refreshExpires.After(time.Now()) {
		return "", ErrNotFound
	}
	if _, err := tx.Exec(ctx, `
		UPDATE api_tokens SET token_hash = $2, token_prefix = $3, expires_at = $4,
		       prev_refresh_hash = refresh_hash, refresh_hash = $5, refresh_expires_at = $6
		WHERE id = $1;`, id, next.AccessHash, next.AccessPrefix, next.AccessExpiresAt, next.RefreshHash, next.RefreshExpiresAt); err != nil {
		return "", fmt.Errorf("failed to rotate oauth grant: %w", err)
	}
	return scope, tx.Commit(ctx)
}

// RevokeOAuthGrant ends the grant of the client that holds the token, an access or a refresh token. Revoking a token
// that is unknown is not an error.
func (r *Repository) RevokeOAuthGrant(ctx context.Context, clientID, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM api_tokens WHERE oauth_client_id = $1 AND (token_hash = $2 OR refresh_hash = $2);`, clientID, tokenHash)
	return err
}

// CleanupOAuth removes what can no longer be used: expired codes, grants whose refresh token expired, and clients
// that registered more than thirty days ago and hold no grant.
func (r *Repository) CleanupOAuth(ctx context.Context) (removed int64, err error) {
	for _, q := range []string{
		`DELETE FROM oauth_codes WHERE expires_at < NOW();`,
		`DELETE FROM api_tokens WHERE oauth_client_id IS NOT NULL AND refresh_expires_at < NOW();`,
		`DELETE FROM oauth_clients c WHERE c.created_at < NOW() - INTERVAL '30 days'
		   AND NOT EXISTS (SELECT 1 FROM api_tokens t WHERE t.oauth_client_id = c.id)
		   AND NOT EXISTS (SELECT 1 FROM oauth_codes o WHERE o.client_id = c.id);`,
	} {
		tag, err := r.pool.Exec(ctx, q)
		if err != nil {
			return removed, err
		}
		removed += tag.RowsAffected()
	}
	return removed, nil
}
