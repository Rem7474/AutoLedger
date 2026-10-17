package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/teslacost/teslacost/internal/models"
)

// Expense document uploads and the legacy DB-to-filesystem storage migration.
// SaveExpenseDocument stores a new uploaded document record in PostgreSQL.
// The binary data is stored on the filesystem volume; only the storage_path is persisted here.
func (r *Repository) SaveExpenseDocument(ctx context.Context, doc *models.ExpenseDocument) error {
	query := `
		INSERT INTO expense_documents (
			user_id, vehicle_id, filename, mime_type, file_size, storage_path, description
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at;
	`
	return r.pool.QueryRow(ctx, query,
		doc.UserID, doc.VehicleID, doc.Filename, doc.MimeType, doc.FileSize, doc.StoragePath, doc.Description,
	).Scan(&doc.ID, &doc.CreatedAt, &doc.UpdatedAt)
}

// GetExpenseDocumentByID retrieves an expense document metadata, its storage path, and legacy binary data if present.
// Access is verified against the current vehicle membership.
func (r *Repository) GetExpenseDocumentByID(ctx context.Context, id, vehicleID, userID string) (*models.ExpenseDocument, error) {
	query := `
		SELECT d.id, d.user_id, d.vehicle_id, d.filename, d.mime_type, d.file_size, d.storage_path, d.data, d.description, d.created_at, d.updated_at
		FROM expense_documents d
		JOIN vehicles v ON v.id = d.vehicle_id
		WHERE d.id::text = $1 AND d.vehicle_id::text = $2 AND EXISTS (
			SELECT 1 FROM vehicle_members vm WHERE vm.vehicle_id = v.id AND vm.user_id::text = $3
		);
	`
	var doc models.ExpenseDocument
	err := r.pool.QueryRow(ctx, query, id, vehicleID, userID).Scan(
		&doc.ID, &doc.UserID, &doc.VehicleID, &doc.Filename, &doc.MimeType, &doc.FileSize, &doc.StoragePath, &doc.Data, &doc.Description, &doc.CreatedAt, &doc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &doc, nil
}

// ListExpenseDocuments lists document headers for a vehicle, including how many expenses link to each document.
func (r *Repository) ListExpenseDocuments(ctx context.Context, vehicleID, userID string) ([]models.ExpenseDocumentHeader, error) {
	query := `
		SELECT
			d.id, d.vehicle_id, d.filename, d.mime_type, d.file_size, d.description,
			(
				(SELECT COUNT(*) FROM drive_expenses de WHERE de.document_id = d.id) +
				(SELECT COUNT(*) FROM maintenance_expenses me WHERE me.document_id = d.id) +
				(SELECT COUNT(*) FROM charge_logs cl WHERE cl.document_id = d.id)
			) AS linked_expenses_count,
			d.created_at
		FROM expense_documents d
		JOIN vehicles v ON v.id = d.vehicle_id
		WHERE d.vehicle_id::text = $1 AND EXISTS (
			SELECT 1 FROM vehicle_members vm WHERE vm.vehicle_id = v.id AND vm.user_id::text = $2
		)
		ORDER BY d.created_at DESC;
	`
	rows, err := r.pool.Query(ctx, query, vehicleID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.ExpenseDocumentHeader
	for rows.Next() {
		var h models.ExpenseDocumentHeader
		if err := rows.Scan(
			&h.ID, &h.VehicleID, &h.Filename, &h.MimeType, &h.FileSize, &h.Description,
			&h.LinkedExpensesCount, &h.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, rows.Err()
}

// DeleteExpenseDocument deletes an expense document. Linked expenses have their document_id set to NULL automatically.
func (r *Repository) DeleteExpenseDocument(ctx context.Context, id, vehicleID, userID string) error {
	query := `
		DELETE FROM expense_documents d
		USING vehicles v
		WHERE d.vehicle_id = v.id AND d.id::text = $1 AND d.vehicle_id::text = $2 AND EXISTS (
			SELECT 1 FROM vehicle_members vm WHERE vm.vehicle_id = v.id AND vm.user_id::text = $3 AND vm.role IN ('OWNER', 'EDITOR')
		);
	`
	cmdTag, err := r.pool.Exec(ctx, query, id, vehicleID, userID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateDocumentStoragePath sets the storage_path for a document after the file has been written to the volume.
func (r *Repository) UpdateDocumentStoragePath(ctx context.Context, docID, storagePath string) error {
	query := `UPDATE expense_documents SET storage_path = $1 WHERE id::text = $2;`
	_, err := r.pool.Exec(ctx, query, storagePath, docID)
	return err
}

// LegacyDocumentRecord represents an unmigrated document record still containing binary data in PostgreSQL.
type LegacyDocumentRecord struct {
	ID        string
	VehicleID string
	Data      []byte
}

// GetUnmigratedDocuments retrieves all documents that have raw binary data in PostgreSQL but no storage_path.
func (r *Repository) GetUnmigratedDocuments(ctx context.Context) ([]LegacyDocumentRecord, error) {
	query := `SELECT id::text, vehicle_id::text, data FROM expense_documents WHERE storage_path IS NULL AND data IS NOT NULL;`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []LegacyDocumentRecord
	for rows.Next() {
		var d LegacyDocumentRecord
		if err := rows.Scan(&d.ID, &d.VehicleID, &d.Data); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

// MigrateLegacyDocuments migrates all unmigrated documents from PostgreSQL BYTEA column to volume storage.
// It writes each file using the saveFile callback, then updates the storage_path in PostgreSQL.
func (r *Repository) MigrateLegacyDocuments(ctx context.Context, saveFile func(vehicleID, docID string, data []byte) (string, error)) (int, error) {
	docs, err := r.GetUnmigratedDocuments(ctx)
	if err != nil {
		return 0, err
	}
	migrated := 0
	for _, doc := range docs {
		if len(doc.Data) == 0 {
			continue
		}
		storagePath, err := saveFile(doc.VehicleID, doc.ID, doc.Data)
		if err != nil {
			return migrated, fmt.Errorf("failed to save document %s to storage: %w", doc.ID, err)
		}
		if err := r.UpdateDocumentStoragePath(ctx, doc.ID, storagePath); err != nil {
			return migrated, fmt.Errorf("failed to update storage path for document %s: %w", doc.ID, err)
		}
		migrated++
	}
	return migrated, nil
}

// ============================================================================
// Maintenance Reminders & Webhooks
// ============================================================================
