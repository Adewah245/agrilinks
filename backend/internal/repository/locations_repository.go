package repository

import (
	"database/sql"
)

type State struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type LGA struct {
	ID      int    `json:"id"`
	StateID int    `json:"state_id"`
	Name    string `json:"name"`
}

type LocationRepository struct {
	db *sql.DB
}

func NewLocationRepository(db *sql.DB) *LocationRepository {
	return &LocationRepository{
		db: db,
	}
}

// GetStates returns all Nigerian states and the FCT.
func (r *LocationRepository) GetStates() ([]State, error) {
	rows, err := r.db.Query(`
		SELECT id, name
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

		states = append(states, state)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return states, nil
}

// GetLGAsByState returns all LGAs belonging to a state.
func (r *LocationRepository) GetLGAsByState(stateID int) ([]LGA, error) {
	rows, err := r.db.Query(`
		SELECT id, state_id, name
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

		lgas = append(lgas, lga)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return lgas, nil
}
