package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"backend/internal/models"
)

// resolveUserLocation accepts IDs from the form or names from imported callers.
// It checks the hierarchy and returns the database's names for display.
func resolveUserLocation(tx *sql.Tx, u *models.User) error {
	if !strings.EqualFold(strings.TrimSpace(u.Country), "Nigeria") {
		return fmt.Errorf("choose a location in Nigeria")
	}
	var stateID, lgaID, communityID int
	err := tx.QueryRow(`SELECT s.id,s.name,l.id,l.name FROM states s JOIN lgas l ON l.state_id=s.id
		WHERE (LOWER(s.name)=LOWER($1) OR s.id::text=$1) AND (LOWER(l.name)=LOWER($2) OR l.id::text=$2)`, u.State, u.LGA).Scan(&stateID, &u.State, &lgaID, &u.LGA)
	if err == sql.ErrNoRows {
		return fmt.Errorf("choose an LGA that belongs to your selected state")
	}
	if err != nil {
		return err
	}
	u.Location = strings.TrimSpace(u.Location)
	if u.Location == "" || len(u.Location) > 200 {
		return fmt.Errorf("enter your community or town name")
	}
	if _, err = tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, fmt.Sprintf("community:%d:%s", lgaID, strings.ToLower(u.Location))); err != nil {
		return err
	}
	err = tx.QueryRow(`SELECT id,name FROM communities WHERE lga_id=$1 AND LOWER(name)=LOWER($2) ORDER BY id LIMIT 1`, lgaID, u.Location).Scan(&communityID, &u.Location)
	if err == sql.ErrNoRows {
		err = tx.QueryRow(`INSERT INTO communities(lga_id,name) VALUES($1,$2) RETURNING id`, lgaID, u.Location).Scan(&communityID)
	}
	if err != nil {
		return err
	}
	u.StateID, u.LGAID, u.CommunityID = &stateID, &lgaID, &communityID
	u.Country = "Nigeria"
	return nil
}

func saveUserFarmingTypes(tx *sql.Tx, userID int, types []string) error {
	seen := map[string]bool{}
	for _, value := range types {
		value = strings.ToLower(strings.TrimSpace(value))
		if seen[value] {
			continue
		}
		seen[value] = true
		var typeID int

		err := tx.QueryRow(
			`SELECT id FROM farming_types WHERE slug=$1`,
			value,
		).Scan(&typeID)

		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("choose a farming activity from the list")
			}

			return fmt.Errorf("look up farming activity %q: %w", value, err)
		}
		if _, err := tx.Exec(`INSERT INTO user_farming_types(user_id,farming_type_id) VALUES($1,$2) ON CONFLICT(user_id,farming_type_id) DO NOTHING`, userID, typeID); err != nil {
			return err
		}
	}
	return nil
}
