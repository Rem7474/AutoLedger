package models

import (
	"time"

	"github.com/teslacost/teslacost/internal/money"
)

// TirePosition represents wheel locations or storage status.
type TirePosition string

const (
	TirePosFL       TirePosition = "FL"
	TirePosFR       TirePosition = "FR"
	TirePosRL       TirePosition = "RL"
	TirePosRR       TirePosition = "RR"
	TirePosStorage  TirePosition = "STORAGE"
	TirePosDisposed TirePosition = "DISPOSED"
)

// TireSeason represents the tire category.
type TireSeason string

const (
	TireSeasonSummer    TireSeason = "SUMMER"
	TireSeasonWinter    TireSeason = "WINTER"
	TireSeasonAllSeason TireSeason = "ALL_SEASON"
)

// Tire represents an individual tire or set entry.
type Tire struct {
	ID                    string       `json:"id"`
	VehicleID             *string      `json:"vehicle_id,omitempty"`
	Brand                 string       `json:"brand"`
	Model                 string       `json:"model"`
	Dimension             string       `json:"dimension"`
	Season                TireSeason   `json:"season"`
	PurchaseDate          time.Time    `json:"purchase_date"`
	PurchasePrice         money.Cents  `json:"purchase_price"`
	CurrentPosition       TirePosition `json:"current_position"`
	InitialDepthMm        float64      `json:"initial_depth_mm"`
	MinLegalDepthMm       float64      `json:"min_legal_depth_mm"`
	DotCode               *string      `json:"dot_code,omitempty"`
	IsArchived            bool         `json:"is_archived"`
	MountedOdometer       *float64     `json:"mounted_odometer,omitempty"`
	InitialDistanceKm     float64      `json:"initial_distance_km"`
	AccumulatedDistanceKm float64      `json:"accumulated_distance_km"`
	EstimatedLifespanKm   int          `json:"estimated_lifespan_km"`
	CreatedAt             time.Time    `json:"created_at"`
	UpdatedAt             time.Time    `json:"updated_at"`
	Version               int64        `json:"version"`
	ExpectedVersion       *int64       `json:"-"`
}

// TireMountSession logs a specific period where a tire was mounted on a vehicle wheel.
type TireMountSession struct {
	ID                 string       `json:"id"`
	TireID             string       `json:"tire_id"`
	VehicleID          string       `json:"vehicle_id"`
	Position           TirePosition `json:"position"`
	MountedDate        time.Time    `json:"mounted_date"`
	MountedOdometer    float64      `json:"mounted_odometer"`
	DismountedDate     *time.Time   `json:"dismounted_date,omitempty"`
	DismountedOdometer *float64     `json:"dismounted_odometer,omitempty"`
	DistanceKm         float64      `json:"distance_km"`
	Notes              *string      `json:"notes,omitempty"`
	CreatedAt          time.Time    `json:"created_at"`
	UpdatedAt          time.Time    `json:"updated_at"`
	Version            int64        `json:"version"`
	ExpectedVersion    *int64       `json:"-"`
}

// TireLog records a tread depth measurement.
type TireLog struct {
	ID              string    `json:"id"`
	TireID          string    `json:"tire_id"`
	Date            time.Time `json:"date"`
	Odometer        float64   `json:"odometer"`
	DepthMm         float64   `json:"depth_mm"`
	Notes           *string   `json:"notes,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	Version         int64     `json:"version"`
	ExpectedVersion *int64    `json:"-"`
}

// TireRotation stores wheel swap events across the vehicle.
type TireRotation struct {
	ID          string         `json:"id"`
	VehicleID   string         `json:"vehicle_id"`
	Date        time.Time      `json:"date"`
	Odometer    float64        `json:"odometer"`
	MappingJSON map[string]any `json:"mapping_json"` // e.g. {"FL": "<uuid>", "FR": "<uuid>"}
	Notes       *string        `json:"notes,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}
