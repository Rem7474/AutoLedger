package handlers

import (
	"net/http"
	"testing"
)

func TestTollInvoiceAPISavesListsAndDeletesAMonth(t *testing.T) {
	a := newCarpoolAPI(t, "tollinvoice")
	h := NewTollInvoiceHandler(a.repo)
	h.SetTimezone("Europe/Paris")
	a.router.Get("/vehicles/{vehicleId}/toll-invoices", h.List)
	a.router.Put("/vehicles/{vehicleId}/toll-invoices/{month}", h.Save)
	a.router.Delete("/vehicles/{vehicleId}/toll-invoices/{month}", h.Delete)

	saved := a.expect(http.StatusOK, "PUT", "/toll-invoices/2026-03", `{"total":"48.20","note":" Avril "}`)
	if saved["month"] != "2026-03" || saved["version"] != float64(1) || saved["note"] != "Avril" {
		t.Fatalf("the month key, a first version and a trimmed note: %+v", saved)
	}
	if saved["period_start"] != "2026-02-28T23:00:00Z" && saved["period_start"] != "2026-03-01T00:00:00+01:00" {
		t.Errorf("the period starts at midnight in the reporting timezone: %v", saved["period_start"])
	}

	list := a.expect(http.StatusOK, "GET", "/toll-invoices", "")
	if invoices, _ := list["invoices"].([]any); len(invoices) != 1 || list["currency"] != "EUR" {
		t.Fatalf("list: %+v", list)
	}

	a.expect(http.StatusConflict, "PUT", "/toll-invoices/2026-03", `{"total":"50","expected_version":7}`)
	a.expect(http.StatusOK, "PUT", "/toll-invoices/2026-03", `{"total":"50","expected_version":1}`)
	a.expect(http.StatusConflict, "DELETE", "/toll-invoices/2026-03?expected_version=1", "")

	a.as(a.viewerID, http.StatusForbidden, "PUT", "/toll-invoices/2026-03", `{"total":"1"}`)
	a.as(a.viewerID, http.StatusOK, "GET", "/toll-invoices", "")
	a.as(a.viewerID, http.StatusForbidden, "DELETE", "/toll-invoices/2026-03", "")

	a.expect(http.StatusNoContent, "DELETE", "/toll-invoices/2026-03?expected_version=2", "")
	a.expect(http.StatusNotFound, "DELETE", "/toll-invoices/2026-03", "")
}

func TestTollInvoiceAPIRefusesWhatCannotBeAnInvoice(t *testing.T) {
	a := newCarpoolAPI(t, "tollinvoicebad")
	h := NewTollInvoiceHandler(a.repo)
	a.router.Put("/vehicles/{vehicleId}/toll-invoices/{month}", h.Save)
	a.router.Delete("/vehicles/{vehicleId}/toll-invoices/{month}", h.Delete)
	for name, c := range map[string][2]string{
		"bad body":       {"2026-03", `{`},
		"bad month":      {"2026-13", `{"total":"1"}`},
		"month as date":  {"2026-03-01", `{"total":"1"}`},
		"future month":   {"2999-01", `{"total":"1"}`},
		"negative total": {"2026-03", `{"total":"-1"}`},
		"huge total":     {"2026-03", `{"total":"99999999999"}`},
	} {
		t.Run(name, func(t *testing.T) { a.expect(http.StatusBadRequest, "PUT", "/toll-invoices/"+c[0], c[1]) })
	}
	a.expect(http.StatusBadRequest, "DELETE", "/toll-invoices/nope", "")
	a.expect(http.StatusBadRequest, "DELETE", "/toll-invoices/2026-03?expected_version=x", "")
	a.expect(http.StatusOK, "PUT", "/toll-invoices/2026-03", `{"total":"0"}`) // no toll that month is a valid invoice
}
