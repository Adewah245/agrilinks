package repository

import (
	"context"
	"database/sql"

	"backend/internal/models"
)

func (r *LearningRepository) ListRooms(ctx context.Context, userID, farmingID int) ([]models.LearningRoom, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT c.id,r.farming_type_id,f.name,c.title,COALESCE(r.description,''),
		(SELECT COUNT(*) FROM conversation_members m WHERE m.conversation_id=c.id AND m.left_at IS NULL),
		EXISTS(SELECT 1 FROM conversation_members m WHERE m.conversation_id=c.id AND m.user_id=$1 AND m.left_at IS NULL)
		FROM learning_rooms r JOIN conversations c ON c.id=r.conversation_id JOIN farming_types f ON f.id=r.farming_type_id
		WHERE ($2=0 OR r.farming_type_id=$2) ORDER BY r.created_at DESC,c.id DESC LIMIT 200`, userID, farmingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.LearningRoom{}
	for rows.Next() {
		var room models.LearningRoom
		if err := rows.Scan(&room.ConversationID, &room.FarmingTypeID, &room.FarmingType, &room.Title, &room.Description, &room.MemberCount, &room.Joined); err != nil {
			return nil, err
		}
		result = append(result, room)
	}
	return result, rows.Err()
}

func (r *LearningRepository) CreateRoom(ctx context.Context, userID, farmingID int, title, description string) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var id int
	if err := tx.QueryRowContext(ctx, `INSERT INTO conversations (conversation_type,title,created_by) VALUES ('group',$1,$2) RETURNING id`, title, userID).Scan(&id); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO conversation_members (conversation_id,user_id,member_role) VALUES ($1,$2,'owner')`, id, userID); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO learning_rooms (conversation_id,farming_type_id,description) VALUES ($1,$2,NULLIF($3,''))`, id, farmingID, description); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (r *LearningRepository) JoinRoom(ctx context.Context, userID, roomID int) error {
	var id int
	err := r.db.QueryRowContext(ctx, `INSERT INTO conversation_members (conversation_id,user_id)
		SELECT conversation_id,$1 FROM learning_rooms WHERE conversation_id=$2
		ON CONFLICT (conversation_id,user_id) DO UPDATE SET left_at=NULL
		RETURNING conversation_id`, userID, roomID).Scan(&id)
	if err == sql.ErrNoRows {
		return ErrLearningNotFound
	}
	return err
}
