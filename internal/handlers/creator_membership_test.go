package handlers

import (
	"net/http"
	"testing"
)

func TestCreatorAccessDoesNotReturnAfterLeaving(t *testing.T) {
	a := newMemberAPI(t, "creator-revocation")
	a.as(a.ownerID, http.StatusCreated, "POST", "/members", `{"email":"`+a.guest+`","role":"OWNER"}`)
	a.as(a.guestID, http.StatusOK, "PUT", "/members/"+a.ownerID, `{"role":"VIEWER"}`)
	a.as(a.ownerID, http.StatusOK, "DELETE", "/members/"+a.ownerID, "")
	a.as(a.ownerID, http.StatusNotFound, "GET", "/members", "")
	a.as(a.ownerID, http.StatusNotFound, "POST", "/members", `{"email":"`+a.guest+`","role":"OWNER"}`)
	a.as(a.guestID, http.StatusOK, "GET", "/members", "")
}
