package repository

import (
	"database/sql"
)

// ============================================================
// STATE
//
// Represents a Nigerian state or the Federal Capital Territory.
// ============================================================

type State struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ============================================================
// LGA
//
// Represents a Local Government Area belonging to a state.
// ============================================================

type LGA struct {
	ID      int    `json:"id"`
	StateID int    `json:"state_id"`
	Name    string `json:"name"`
}

// ============================================================
// LOCATION REPOSITORY
//
// Responsibility:
// - Retrieves Nigerian states and the FCT.
// - Retrieves LGAs belonging to a specific state.
//
// This repository works with:
//
// states
// lgas
//
// These locations are used across Agro-Shield for:
//
// - User profiles
// - Product listings
// - Organizations
// - Marketplace searches
// - Location-based agricultural services
// ============================================================

type LocationRepository struct {
	db *sql.DB
}

// ============================================================
// CREATE REPOSITORY
// ============================================================

// NewLocationRepository creates a new repository for managing
// Nigerian states and Local Government Areas.
func NewLocationRepository(db *sql.DB) *LocationRepository {
	return &LocationRepository{
		db: db,
	}
}

// ============================================================
// GET STATES
//
// Returns all Nigerian states and the Federal Capital Territory.
//
// Results are ordered alphabetically by name.
// ============================================================

// GetStates returns all Nigerian states and the FCT.
func (r *LocationRepository) GetStates() ([]State, error) {
	rows, err := r.db.Query(`
		SELECT
			id,
			name
		FROM states
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	states := make([]State, 0)

	for rows.Next() {
		var state State

		if err := rows.Scan(
			&state.ID,
			&state.Name,
		); err != nil {
			return nil, err
		}

		states = append(
			states,
			state,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return states, nil
}

// ============================================================
// GET LGAs BY STATE
//
// Returns all Local Government Areas belonging to a state.
//
// Example:
//
// State:
// Benue
//
// LGAs:
// Otukpo
// Makurdi
// Gboko
// Ogbadibo
// etc.
// ============================================================

// GetLGAsByState returns all LGAs belonging to a state.
func (r *LocationRepository) GetLGAsByState(
	stateID int,
) ([]LGA, error) {

	rows, err := r.db.Query(`
		SELECT
			id,
			state_id,
			name
		FROM lgas
		WHERE state_id = $1
		ORDER BY name ASC
	`, stateID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	lgas := make([]LGA, 0)

	for rows.Next() {
		var lga LGA

		if err := rows.Scan(
			&lga.ID,
			&lga.StateID,
			&lga.Name,
		); err != nil {
			return nil, err
		}

		lgas = append(
			lgas,
			lga,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return lgas, nil
}