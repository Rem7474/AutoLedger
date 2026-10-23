package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/teslacost/teslacost/internal/auth"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
)

// A revoked session can neither read nor change the sharing of a vehicle: giving another account access would outlive
// the access token that did it.
func TestRevokedSessionCannotChangeMembership(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	hash, err := auth.HashPassword("Correct-Horse-9")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := repo.CreateUser(ctx, "membership-owner@example.com", hash)
	if err != nil {
		t.Fatal(err)
	}
	guest, err := repo.CreateUser(ctx, "membership-guest@example.com", hash)
	if err != nil {
		t.Fatal(err)
	}
	vehicle := &models.Vehicle{Currency: "EUR", UserID: owner.ID, Name: "Shared", TeslaMateAuthType: models.AuthModeNone, CurrentOdometer: 1000}
	if err := repo.CreateVehicle(ctx, vehicle); err != nil {
		t.Fatal(err)
	}
	h := NewAuthHandler(repo, authTestConfig(), nil)
	r := accountRouter(repo, h)
	members := NewVehicleMemberHandler(repo)
	guarded := r.With(middleware.AuthenticateJWT(authTestConfig().JWTSecret), middleware.RequireActiveSession(repo))
	guarded.Get("/vehicles/{id}/members", members.ListMembers)
	guarded.Post("/vehicles/{id}/members", members.AddMember)
	guarded.Put("/vehicles/{id}/members/{memberId}", members.UpdateMemberRole)
	guarded.Delete("/vehicles/{id}/members/{memberId}", members.RemoveMember)

	token, _ := signInToken(t, h, owner.Email, "Correct-Horse-9")
	path := "/vehicles/" + vehicle.ID + "/members"
	if response := callWithBearer(r, http.MethodPost, path, token, `{"email":"`+guest.Email+`","role":"VIEWER"}`); response.Code != http.StatusCreated {
		t.Fatal(response.Body.String())
	}
	if response := callWithBearer(r, http.MethodPost, "/api/auth/logout-all", token, ""); response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	for _, request := range []struct{ method, path, body string }{
		{http.MethodGet, path, ""},
		{http.MethodPost, path, `{"email":"` + guest.Email + `","role":"OWNER"}`},
		{http.MethodPut, path + "/" + guest.ID, `{"role":"OWNER"}`},
		{http.MethodDelete, path + "/" + guest.ID, ""},
	} {
		if response := callWithBearer(r, request.method, request.path, token, request.body); response.Code != http.StatusUnauthorized {
			t.Fatalf("%s with a revoked session: %d %s", request.method, response.Code, response.Body.String())
		}
	}
	if role, err := repo.GetVehicleMemberRole(ctx, vehicle.ID, guest.ID); err != nil || role != models.RoleViewer {
		t.Fatalf("membership changed: %s, %v", role, err)
	}
}
