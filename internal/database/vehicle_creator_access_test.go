package database

import (
	"testing"

	"github.com/teslacost/teslacost/internal/models"
)

func TestCreatorAccessFollowsCurrentMembership(t *testing.T) {
	repo, ctx, creator, other, vehicle := vehicleRepoFixture(t, "creator-access")
	if _, err := repo.AddVehicleMember(ctx, vehicle.ID, other.Email, models.RoleOwner, nil); err != nil {
		t.Fatal(err)
	}
	doc := &models.ExpenseDocument{UserID: creator.ID, VehicleID: vehicle.ID, Filename: "receipt.pdf", MimeType: "application/pdf", FileSize: 1}
	if err := repo.SaveExpenseDocument(ctx, doc); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateVehicleMemberRole(ctx, vehicle.ID, creator.ID, models.RoleViewer); err != nil {
		t.Fatal(err)
	}
	price := 0.25
	requireNotFound(t, repo.UpdateVehicleEstimatedEnergy(ctx, vehicle.ID, creator.ID, nil, &price))
	requireNotFound(t, repo.DeleteVehicle(ctx, vehicle.ID, creator.ID))
	requireNotFound(t, repo.DeleteExpenseDocument(ctx, doc.ID, vehicle.ID, creator.ID))
	if _, err := repo.GetExpenseDocumentByID(ctx, doc.ID, vehicle.ID, creator.ID); err != nil {
		t.Fatalf("a viewer can still read: %v", err)
	}
	if err := repo.RemoveVehicleMember(ctx, vehicle.ID, creator.ID); err != nil {
		t.Fatal(err)
	}
	_, err := repo.GetVehicleByID(ctx, vehicle.ID, creator.ID)
	requireNotFound(t, err)
	_, err = repo.GetVehicleMemberRole(ctx, vehicle.ID, creator.ID)
	requireNotFound(t, err)
	vehicles, err := repo.ListVehiclesByUserID(ctx, creator.ID)
	if err != nil || len(vehicles) != 0 {
		t.Fatalf("removed creator's vehicles = %v, %v", vehicles, err)
	}
	_, err = repo.GetExpenseDocumentByID(ctx, doc.ID, vehicle.ID, creator.ID)
	requireNotFound(t, err)
	docs, err := repo.ListExpenseDocuments(ctx, vehicle.ID, creator.ID)
	if err != nil || len(docs) != 0 {
		t.Fatalf("removed creator's documents = %v, %v", docs, err)
	}
	requireNotFound(t, repo.DeleteVehicle(ctx, vehicle.ID, creator.ID))
	if got, err := repo.GetVehicleByID(ctx, vehicle.ID, other.ID); err != nil || got.Role != models.RoleOwner {
		t.Fatalf("remaining owner = %+v, %v", got, err)
	}
}
