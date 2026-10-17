package handlers

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/teslacost/teslacost/internal/models"
)

func TestWebhookURLIsRestrictedToEditors(t *testing.T) {
	a := newReminderAPI(t, "webhook-reader")
	url := "https://notifications.example/api/webhook/private-capability"
	a.want(a.do(a.owner, http.MethodPut, "/webhook", WebhookPayload{URL: url, Type: "GENERIC", Enabled: true}), http.StatusOK, "")
	reader := a.do(a.viewer, http.MethodGet, "/webhook", nil)
	a.want(reader, http.StatusOK, "")
	if strings.Contains(reader.Body.String(), "private-capability") {
		t.Fatal("reader received the webhook credential")
	}
	body := decodeObj(t, reader.Body.Bytes())
	if body["enabled"] != true || body["type"] != "GENERIC" || body["url"] != "" {
		t.Fatalf("reader status: %v", body)
	}
	for _, role := range []models.VehicleRole{models.RoleEditor, models.RoleOwner} {
		if err := a.repo.UpdateVehicleMemberRole(context.Background(), a.vid, a.viewer, role); err != nil {
			t.Fatal(err)
		}
		response := a.do(a.viewer, http.MethodGet, "/webhook", nil)
		a.want(response, http.StatusOK, "")
		if decodeObj(t, response.Body.Bytes())["url"] != url {
			t.Fatalf("%s must be able to edit the existing URL", role)
		}
	}
	stored, err := a.repo.GetVehicleWebhook(context.Background(), a.vid)
	if err != nil || stored.URL != url {
		t.Fatalf("redaction changed stored URL: %+v, %v", stored, err)
	}
}
