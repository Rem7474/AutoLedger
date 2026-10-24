package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/teslacost/teslacost/internal/auth"
	"github.com/teslacost/teslacost/internal/models"
)

// A token states what it may do when it is created: there is no default scope, and an unknown one is refused.
func TestCreateTokenRequiresAKnownScope(t *testing.T) {
	repo := authTestRepo(t)
	hash, _ := auth.HashPassword("Correct-Horse-9")
	user, _ := repo.CreateUser(context.Background(), "scope@example.org", hash)
	h := NewAuthHandler(repo, authTestConfig(), nil)
	r := accountRouter(repo, h)
	session, _ := signInToken(t, h, "scope@example.org", "Correct-Horse-9")

	create := func(body string) int {
		return callWithBearer(r, http.MethodPost, "/api/auth/tokens", session, body).Code
	}
	for name, body := range map[string]string{
		"no scope":       `{"name":"dash"}`,
		"unknown scope":  `{"name":"dash","scope":"admin"}`,
		"another casing": `{"name":"dash","scope":"READ"}`,
	} {
		if got := create(body); got != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", name, got)
		}
	}
	if got := create(`{"name":"dash","scope":"read"}`); got != http.StatusCreated {
		t.Fatalf("read token: %d", got)
	}
	if got := create(`{"name":"ha","scope":"ingest"}`); got != http.StatusCreated {
		t.Fatalf("ingest token: %d", got)
	}

	tokens, err := repo.ListAPITokens(context.Background(), user.ID)
	if err != nil || len(tokens) != 2 {
		t.Fatalf("only the two valid requests create a token: %+v, %v", tokens, err)
	}
	scopes := map[string]string{}
	for _, tok := range tokens {
		scopes[tok.Name] = tok.Scope
	}
	if scopes["dash"] != auth.ScopeRead || scopes["ha"] != auth.ScopeIngest {
		t.Errorf("each token keeps the scope it was created with: %v", scopes)
	}
}

// The calls of an access are read from the account page by its owner only.
func TestListTokenCallsAnswersTheOwnerOnly(t *testing.T) {
	repo := authTestRepo(t)
	hash, _ := auth.HashPassword("Correct-Horse-9")
	owner, _ := repo.CreateUser(context.Background(), "calls-owner@example.org", hash)
	_, _ = repo.CreateUser(context.Background(), "calls-other@example.org", hash)
	h := NewAuthHandler(repo, authTestConfig(), nil)
	r := accountRouter(repo, h)
	mine, _ := signInToken(t, h, "calls-owner@example.org", "Correct-Horse-9")
	theirs, _ := signInToken(t, h, "calls-other@example.org", "Correct-Horse-9")

	tok, err := repo.CreateAPIToken(context.Background(), owner.ID, "assistant", auth.ScopeRead, "hash-owner-calls", "al_oc", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordAPITokenCall(context.Background(), tok.ID, models.APITokenCall{Tool: "list_vehicles", Outcome: "ok", DurationMS: 12}); err != nil {
		t.Fatal(err)
	}

	rec := callWithBearer(r, http.MethodGet, "/api/auth/tokens/"+tok.ID+"/calls", mine, "")
	var body struct {
		Calls []models.APITokenCall `json:"calls"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &body) != nil || len(body.Calls) != 1 || body.Calls[0].Tool != "list_vehicles" {
		t.Fatalf("the owner reads the calls: %d %s", rec.Code, rec.Body)
	}
	if got := callWithBearer(r, http.MethodGet, "/api/auth/tokens/"+tok.ID+"/calls", theirs, "").Code; got != http.StatusNotFound {
		t.Errorf("another account: got %d, want 404", got)
	}
}
