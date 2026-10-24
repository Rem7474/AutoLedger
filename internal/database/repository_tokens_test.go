package database

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
)

func TestAPITokens(t *testing.T) {
	pool := sourcesTestPool(t)
	ctx := context.Background()
	if err := (&DB{Pool: pool}).Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(pool)

	user, err := repo.CreateUser(ctx, "tokens-owner@example.com", "x")
	if err != nil {
		t.Fatal(err)
	}
	other, err := repo.CreateUser(ctx, "tokens-other@example.com", "x")
	if err != nil {
		t.Fatal(err)
	}

	list, err := repo.ListAPITokens(ctx, user.ID)
	if err != nil || list == nil || len(list) != 0 {
		t.Fatalf("a user without token must get an empty, non-nil list: %+v, %v", list, err)
	}

	live, err := repo.CreateAPIToken(ctx, user.ID, "home assistant", "ingest", "hash-live", "al_live", nil)
	if err != nil || live.ID == "" || live.CreatedAt.IsZero() {
		t.Fatalf("create = %+v, %v", live, err)
	}
	past := time.Now().Add(-time.Hour)
	expired, err := repo.CreateAPIToken(ctx, user.ID, "old", "read", "hash-expired", "al_old", &past)
	if err != nil {
		t.Fatal(err)
	}

	list, err = repo.ListAPITokens(ctx, user.ID)
	if err != nil || len(list) != 2 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if list[0].ID != expired.ID {
		t.Fatalf("tokens must be listed newest first, got %s first", list[0].Name)
	}
	if list[0].Scope != "read" || list[1].Scope != "ingest" {
		t.Fatalf("the scope is stored and listed with the token: %q, %q", list[0].Scope, list[1].Scope)
	}

	// Only a live token validates, and it resolves to its owner.
	id, email, scope, err := repo.ValidateAPIToken(ctx, "hash-live")
	if err != nil || id != user.ID || email != "tokens-owner@example.com" || scope != "ingest" {
		t.Fatalf("validate = %q %q %q %v", id, email, scope, err)
	}
	if _, _, _, err := repo.ValidateAPIToken(ctx, "hash-expired"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired token: err = %v, want ErrNotFound", err)
	}
	if _, _, _, err := repo.ValidateAPIToken(ctx, "hash-unknown"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown token: err = %v, want ErrNotFound", err)
	}

	// Validation records the last use, asynchronously.
	deadline := time.Now().Add(3 * time.Second)
	for {
		list, _ = repo.ListAPITokens(ctx, user.ID)
		var used bool
		for _, tok := range list {
			if tok.ID == live.ID && tok.LastUsedAt != nil {
				used = true
			}
		}
		if used {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("last_used_at was not recorded")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// A token can only be revoked by its owner.
	if err := repo.RevokeAPIToken(ctx, other.ID, live.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoke by another user: err = %v, want ErrNotFound", err)
	}
	if err := repo.RevokeAPIToken(ctx, user.ID, live.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := repo.ValidateAPIToken(ctx, "hash-live"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoked token: err = %v, want ErrNotFound", err)
	}
}

func TestAPITokenCallsKeepTheLatestOnly(t *testing.T) {
	pool := sourcesTestPool(t)
	ctx := context.Background()
	if err := (&DB{Pool: pool}).Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(pool)
	owner, err := repo.CreateUser(ctx, "calls-owner@example.com", "x")
	if err != nil {
		t.Fatal(err)
	}
	other, err := repo.CreateUser(ctx, "calls-other@example.com", "x")
	if err != nil {
		t.Fatal(err)
	}
	tok, err := repo.CreateAPIToken(ctx, owner.ID, "assistant", "read", "hash-calls", "al_calls", nil)
	if err != nil {
		t.Fatal(err)
	}

	if calls, err := repo.ListAPITokenCalls(ctx, owner.ID, tok.ID); err != nil || calls == nil || len(calls) != 0 {
		t.Fatalf("a token without call lists an empty, non-nil slice: %+v, %v", calls, err)
	}
	for i := 0; i < apiTokenCallsKept+5; i++ {
		outcome := "ok"
		if i%2 == 1 {
			outcome = "error"
		}
		if err := repo.RecordAPITokenCall(ctx, tok.ID, models.APITokenCall{Tool: fmt.Sprintf("tool_%d", i), Outcome: outcome, DurationMS: i}); err != nil {
			t.Fatal(err)
		}
	}
	calls, err := repo.ListAPITokenCalls(ctx, owner.ID, tok.ID)
	if err != nil || len(calls) != apiTokenCallsKept {
		t.Fatalf("calls = %d, %v; want %d", len(calls), err, apiTokenCallsKept)
	}
	if last := apiTokenCallsKept + 4; calls[0].Tool != fmt.Sprintf("tool_%d", last) || calls[0].DurationMS != last || calls[0].Outcome != "ok" {
		t.Errorf("the most recent call comes first, with its outcome: %+v", calls[0])
	}
	if calls[len(calls)-1].Tool != "tool_5" {
		t.Errorf("the oldest calls are dropped: %+v", calls[len(calls)-1])
	}

	// A second token keeps its own log.
	second, err := repo.CreateAPIToken(ctx, owner.ID, "other assistant", "read", "hash-calls-2", "al_calls2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordAPITokenCall(ctx, second.ID, models.APITokenCall{Tool: "list_vehicles", Outcome: "ok"}); err != nil {
		t.Fatal(err)
	}
	if calls, _ := repo.ListAPITokenCalls(ctx, owner.ID, tok.ID); len(calls) != apiTokenCallsKept {
		t.Errorf("another token's call does not push this log: %d", len(calls))
	}

	if _, err := repo.ListAPITokenCalls(ctx, other.ID, tok.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("another account: err = %v, want ErrNotFound", err)
	}

	// The calls go with the token.
	if err := repo.RevokeAPIToken(ctx, owner.ID, tok.ID); err != nil {
		t.Fatal(err)
	}
	var left int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM api_token_calls WHERE token_id = $1;`, tok.ID).Scan(&left); err != nil || left != 0 {
		t.Errorf("calls left after revocation: %d, %v", left, err)
	}
}
