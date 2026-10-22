package repository

import (
	"backend/internal/models"
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrLearningNotFound = errors.New("learning journey not found")
var ErrLearningConflict = errors.New("your learning page changed; refresh it and try again")

type LearningRepository struct{ db *sql.DB }

func NewLearningRepository(db *sql.DB) *LearningRepository { return &LearningRepository{db: db} }

func (r *LearningRepository) ListFarmingTypes() ([]models.FarmingType, error) {
	rows, err := r.db.Query(`SELECT id,name,slug,COALESCE(description,''),created_at FROM farming_types ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.FarmingType{}
	for rows.Next() {
		var item models.FarmingType
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Description, &item.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *LearningRepository) GetUserFarmingTypes(userID int) ([]models.UserFarmingType, error) {
	rows, err := r.db.Query(`SELECT u.id,u.user_id,u.farming_type_id,u.status,u.source,u.confirmed_at,u.created_at,u.updated_at,
        f.id,f.name,f.slug,COALESCE(f.description,''),f.created_at FROM user_farming_types u JOIN farming_types f ON f.id=u.farming_type_id
        WHERE u.user_id=$1 ORDER BY f.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.UserFarmingType{}
	for rows.Next() {
		var item models.UserFarmingType
		var farming models.FarmingType
		if err := rows.Scan(&item.ID, &item.UserID, &item.FarmingTypeID, &item.Status, &item.Source, &item.ConfirmedAt, &item.CreatedAt, &item.UpdatedAt,
			&farming.ID, &farming.Name, &farming.Slug, &farming.Description, &farming.CreatedAt); err != nil {
			return nil, err
		}
		item.FarmingType = &farming
		result = append(result, item)
	}
	return result, rows.Err()
}

const journeySelect = `SELECT j.id,j.user_id,j.farming_type_id,j.user_farming_type_id,j.status,j.experience_level,COALESCE(j.goal,''),
    j.current_stage,j.readiness_score,j.started_at,j.last_activity_at,j.completed_at,j.confirmed_at,j.created_at,j.updated_at,
    f.id,f.name,f.slug,COALESCE(f.description,''),f.created_at FROM learning_journeys j JOIN farming_types f ON f.id=j.farming_type_id `

func scanJourney(row interface{ Scan(...any) error }) (*models.LearningJourney, error) {
	var j models.LearningJourney
	var f models.FarmingType
	err := row.Scan(&j.ID, &j.UserID, &j.FarmingTypeID, &j.UserFarmingTypeID, &j.Status, &j.ExperienceLevel, &j.Goal, &j.CurrentStage, &j.ReadinessScore,
		&j.StartedAt, &j.LastActivityAt, &j.CompletedAt, &j.ConfirmedAt, &j.CreatedAt, &j.UpdatedAt, &f.ID, &f.Name, &f.Slug, &f.Description, &f.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrLearningNotFound
	}
	j.FarmingType = &f
	return &j, err
}

func (r *LearningRepository) JourneyForUser(userID, id int) (*models.LearningJourney, error) {
	return scanJourney(r.db.QueryRow(journeySelect+` WHERE j.id=$1 AND j.user_id=$2`, id, userID))
}

func (r *LearningRepository) ListJourneys(userID int) ([]models.LearningJourney, error) {
	rows, err := r.db.Query(journeySelect+` WHERE j.user_id=$1 ORDER BY j.last_activity_at DESC,j.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.LearningJourney{}
	for rows.Next() {
		item, err := scanJourney(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *item)
	}
	return result, rows.Err()
}

func (r *LearningRepository) StartJourney(userID, farmingID int, experience, goal string) (*models.LearningJourney, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var id int
	if err := tx.QueryRow(`SELECT id FROM users WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, userID).Scan(&id); err != nil {
		return nil, err
	}
	err = tx.QueryRow(`SELECT id FROM learning_journeys WHERE user_id=$1 AND farming_type_id=$2 AND status IN ('active','paused')`, userID, farmingID).Scan(&id)
	if err == nil {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return r.JourneyForUser(userID, id)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	var interestID int
	err = tx.QueryRow(`INSERT INTO user_farming_types (user_id,farming_type_id,status,source)
        VALUES ($1,$2,'learning','learning_journey') ON CONFLICT (user_id,farming_type_id) DO UPDATE
        SET status=CASE WHEN user_farming_types.status='confirmed' THEN user_farming_types.status ELSE 'learning' END,
        updated_at=CURRENT_TIMESTAMP RETURNING id`, userID, farmingID).Scan(&interestID)
	if err != nil {
		return nil, err
	}
	err = tx.QueryRow(`INSERT INTO learning_journeys (user_id,farming_type_id,user_farming_type_id,experience_level,goal,current_stage)
        VALUES ($1,$2,$3,$4,NULLIF($5,''),'learning') RETURNING id`, userID, farmingID, interestID, experience, goal).Scan(&id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.JourneyForUser(userID, id)
}

func (r *LearningRepository) GetLearningMessages(journeyID int) ([]models.LearningMessage, error) {
	rows, err := r.db.Query(`SELECT id,journey_id,sender_type,message_type,content,created_at FROM learning_messages WHERE journey_id=$1 ORDER BY id`, journeyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.LearningMessage{}
	for rows.Next() {
		var m models.LearningMessage
		if err := rows.Scan(&m.ID, &m.JourneyID, &m.SenderType, &m.MessageType, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// SaveExchange records both sides only after a provider returns an answer.
// It detects simultaneous questions so replies cannot be saved out of order.
func (r *LearningRepository) SaveExchange(ctx context.Context, userID, journeyID, lastID int, question, answer string) ([]models.LearningMessage, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := lockActiveJourney(ctx, tx, userID, journeyID); err != nil {
		return nil, err
	}
	var latest int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(id),0) FROM learning_messages WHERE journey_id=$1`, journeyID).Scan(&latest); err != nil {
		return nil, err
	}
	if latest != lastID {
		return nil, ErrLearningConflict
	}
	result := []models.LearningMessage{{JourneyID: journeyID, SenderType: "user", MessageType: "text", Content: question}, {JourneyID: journeyID, SenderType: "ai", MessageType: "guidance", Content: answer}}
	for i := range result {
		m := &result[i]
		if err := tx.QueryRowContext(ctx, `INSERT INTO learning_messages (journey_id,sender_type,message_type,content)
            VALUES ($1,$2,$3,$4) RETURNING id,created_at`, m.JourneyID, m.SenderType, m.MessageType, m.Content).Scan(&m.ID, &m.CreatedAt); err != nil {
			return nil, err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE learning_journeys SET last_activity_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, journeyID); err != nil {
		return nil, err
	}
	return result, tx.Commit()
}

func lockActiveJourney(ctx context.Context, tx *sql.Tx, userID, journeyID int) error {
	var status string
	err := tx.QueryRowContext(ctx, `SELECT status FROM learning_journeys WHERE id=$1 AND user_id=$2 FOR UPDATE`, journeyID, userID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrLearningNotFound
	}
	if err != nil {
		return err
	}
	if status != "active" {
		return errors.New("resume this learning journey before changing it")
	}
	return nil
}

func (r *LearningRepository) GetLearningCheckpoints(journeyID int) ([]models.LearningCheckpoint, error) {
	rows, err := r.db.Query(`SELECT id,journey_id,checkpoint_key,title,COALESCE(description,''),status,progress,COALESCE(notes,''),completed_at,created_at,updated_at
        FROM learning_checkpoints WHERE journey_id=$1 ORDER BY id`, journeyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.LearningCheckpoint{}
	for rows.Next() {
		var c models.LearningCheckpoint
		if err := rows.Scan(&c.ID, &c.JourneyID, &c.CheckpointKey, &c.Title, &c.Description, &c.Status, &c.Progress, &c.Notes, &c.CompletedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *LearningRepository) CreateCheckpoint(userID int, c *models.LearningCheckpoint) error {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := lockActiveJourney(ctx, tx, userID, c.JourneyID); err != nil {
		return err
	}
	err = tx.QueryRowContext(ctx, `INSERT INTO learning_checkpoints (journey_id,checkpoint_key,title,description)
        VALUES ($1,$2,$3,NULLIF($4,'')) RETURNING id,created_at,updated_at`, c.JourneyID, c.CheckpointKey, c.Title, c.Description).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return err
	}
	if err := refreshLearningProgress(ctx, tx, c.JourneyID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *LearningRepository) UpdateCheckpoint(userID int, c *models.LearningCheckpoint) error {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := lockActiveJourney(ctx, tx, userID, c.JourneyID); err != nil {
		return err
	}
	err = tx.QueryRowContext(ctx, `UPDATE learning_checkpoints SET status=$1,progress=$2,notes=NULLIF($3,''),
        completed_at=CASE WHEN $1='completed' THEN COALESCE(completed_at,CURRENT_TIMESTAMP) ELSE NULL END,updated_at=CURRENT_TIMESTAMP
        WHERE id=$4 AND journey_id=$5 RETURNING completed_at,updated_at`, c.Status, c.Progress, c.Notes, c.ID, c.JourneyID).Scan(&c.CompletedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrLearningNotFound
	}
	if err != nil {
		return err
	}
	if err := refreshLearningProgress(ctx, tx, c.JourneyID); err != nil {
		return err
	}
	return tx.Commit()
}

func refreshLearningProgress(ctx context.Context, tx *sql.Tx, journeyID int) error {
	_, err := tx.ExecContext(ctx, `UPDATE learning_journeys SET readiness_score=p.score,
        current_stage=CASE WHEN p.score=100 THEN 'readiness' WHEN p.score>0 THEN 'preparation' ELSE 'learning' END,
        last_activity_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP
        FROM (SELECT COALESCE(FLOOR(AVG(progress)),0)::integer AS score FROM learning_checkpoints WHERE journey_id=$1 AND status<>'not_applicable') p
        WHERE id=$1`, journeyID)
	return err
}

func (r *LearningRepository) ChangeJourneyStatus(userID, journeyID int, status string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var farmingID, interestID int
	var previous string
	err = tx.QueryRow(`SELECT farming_type_id,user_farming_type_id,status FROM learning_journeys WHERE id=$1 AND user_id=$2 FOR UPDATE`, journeyID, userID).Scan(&farmingID, &interestID, &previous)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrLearningNotFound
	}
	if err != nil {
		return err
	}
	if status == "completed" {
		var count, incomplete int
		if err := tx.QueryRow(`SELECT COUNT(*),COUNT(*) FILTER (WHERE status<>'completed' OR progress<>100)
            FROM learning_checkpoints WHERE journey_id=$1 AND status<>'not_applicable'`, journeyID).Scan(&count, &incomplete); err != nil {
			return err
		}
		if count == 0 || incomplete > 0 {
			return errors.New("finish your checklist before completing this journey")
		}
	}
	if status == "active" {
		var other bool
		if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM learning_journeys WHERE user_id=$1 AND farming_type_id=$2 AND id<>$3 AND status IN ('active','paused'))`, userID, farmingID, journeyID).Scan(&other); err != nil {
			return err
		}
		if other {
			return errors.New("another journey for this farming type is already active")
		}
	}
	_, err = tx.Exec(`UPDATE learning_journeys SET status=$1,
        current_stage=CASE WHEN $1='completed' THEN 'confirmed' WHEN current_stage='confirmed' THEN 'readiness' ELSE current_stage END,
        completed_at=CASE WHEN $1='completed' THEN CURRENT_TIMESTAMP ELSE NULL END,
        confirmed_at=CASE WHEN $1='completed' THEN CURRENT_TIMESTAMP ELSE NULL END,
        last_activity_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE id=$2`, status, journeyID)
	if err != nil {
		return err
	}
	if status == "completed" {
		_, err = tx.Exec(`UPDATE user_farming_types SET status='confirmed',source='learning_journey',confirmed_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP
            WHERE id=$1 AND source<>'admin_confirmation'`, interestID)
		if err != nil {
			return fmt.Errorf("save completed learning interest: %w", err)
		}
	}
	return tx.Commit()
}
