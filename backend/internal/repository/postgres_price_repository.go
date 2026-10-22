package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"backend/internal/models"
)

type PostgresPriceRepository struct{ db *sql.DB }

func NewPostgresPriceRepository(db *sql.DB) *PostgresPriceRepository {
	return &PostgresPriceRepository{db: db}
}

func (r *PostgresPriceRepository) SaveReport(ctx context.Context, report *models.PriceReport) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO market_price_series (commodity_id, market_id, currency, unit)
		VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`, report.CommodityID, report.MarketID, report.Currency, report.Unit); err != nil {
		return err
	}
	var currency, unit string
	if err := tx.QueryRowContext(ctx, `SELECT currency,unit FROM market_price_series WHERE commodity_id=$1 AND market_id=$2`, report.CommodityID, report.MarketID).Scan(&currency, &unit); err != nil {
		return err
	}
	if currency != report.Currency || unit != report.Unit {
		return fmt.Errorf("%w: this market and product use %s per %s", models.ErrInvalidInput, currency, unit)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO price_reports
		(id,commodity_id,market_id,submitter_id,price,currency,unit,status,flag_reason,submitted_at,reviewed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, report.ID, report.CommodityID, report.MarketID,
		report.SubmitterID, report.Price, report.Currency, report.Unit, report.Status, report.FlagReason, report.SubmittedAt, report.ReviewedAt)
	if err != nil {
		return err
	}
	return tx.Commit()
}

const priceReportColumns = `id::text,commodity_id,market_id,submitter_id::text,price,currency,unit,status,flag_reason,submitted_at,reviewed_at`

func scanPriceReport(row interface{ Scan(...any) error }) (*models.PriceReport, error) {
	var report models.PriceReport
	err := row.Scan(&report.ID, &report.CommodityID, &report.MarketID, &report.SubmitterID, &report.Price,
		&report.Currency, &report.Unit, &report.Status, &report.FlagReason, &report.SubmittedAt, &report.ReviewedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, models.ErrNotFound
	}
	return &report, err
}

func (r *PostgresPriceRepository) GetReport(ctx context.Context, id string) (*models.PriceReport, error) {
	return scanPriceReport(r.db.QueryRowContext(ctx, `SELECT `+priceReportColumns+` FROM price_reports WHERE id=$1`, id))
}

func (r *PostgresPriceRepository) UpdateReportStatus(ctx context.Context, id string, status models.ReportStatus, reason string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE price_reports SET status=$2,flag_reason=$3,reviewed_at=CURRENT_TIMESTAMP
		WHERE id=$1 AND status IN ('pending','flagged')`, id, status, reason)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%w: report was already reviewed or is missing", models.ErrInvalidInput)
	}
	return nil
}

func (r *PostgresPriceRepository) RecentAcceptedPrices(ctx context.Context, commodity, market string, since time.Time) ([]models.PriceReport, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+priceReportColumns+` FROM price_reports
		WHERE commodity_id=$1 AND market_id=$2 AND status='accepted' AND submitted_at >= $3 ORDER BY submitted_at,id`, commodity, market, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	reports := []models.PriceReport{}
	for rows.Next() {
		report, err := scanPriceReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, *report)
	}
	return reports, rows.Err()
}

func (r *PostgresPriceRepository) CurrentAggregate(ctx context.Context, commodity, market string) (*models.PriceAggregate, error) {
	var result models.PriceAggregate
	err := r.db.QueryRowContext(ctx, `SELECT commodity_id,market_id,MAX(submitted_at),AVG(price),MIN(price),MAX(price),COUNT(*),currency,unit
		FROM price_reports WHERE commodity_id=$1 AND market_id=$2 AND status='accepted'
		AND submitted_at >= CURRENT_TIMESTAMP - INTERVAL '24 hours'
		GROUP BY commodity_id,market_id,currency,unit`, commodity, market).Scan(&result.CommodityID, &result.MarketID, &result.Date,
		&result.AvgPrice, &result.MinPrice, &result.MaxPrice, &result.SampleSize, &result.Currency, &result.Unit)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, models.ErrNotFound
	}
	return &result, err
}

func (r *PostgresPriceRepository) SaveAggregate(ctx context.Context, value *models.PriceAggregate) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO price_aggregates (commodity_id,market_id,calculated_at,avg_price,min_price,max_price,sample_size)
		VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (commodity_id,market_id) DO UPDATE
		SET calculated_at=EXCLUDED.calculated_at,avg_price=EXCLUDED.avg_price,min_price=EXCLUDED.min_price,max_price=EXCLUDED.max_price,sample_size=EXCLUDED.sample_size`,
		value.CommodityID, value.MarketID, value.Date, value.AvgPrice, value.MinPrice, value.MaxPrice, value.SampleSize)
	return err
}

type PostgresSubmitterRepository struct{ db *sql.DB }

func NewPostgresSubmitterRepository(db *sql.DB) *PostgresSubmitterRepository {
	return &PostgresSubmitterRepository{db: db}
}

func (r *PostgresSubmitterRepository) Get(ctx context.Context, id string) (*models.Submitter, error) {
	userID, err := strconv.Atoi(id)
	if err != nil || userID <= 0 {
		return nil, models.ErrInvalidInput
	}
	var result models.Submitter
	// The user record establishes identity. Reporting history establishes counts;
	// a missing moderation profile does not make the user verified.
	err = r.db.QueryRowContext(ctx, `SELECT u.id::text,COALESCE(s.role,CASE WHEN u.is_admin THEN 'admin' ELSE 'farmer' END),
		COALESCE(s.verified,FALSE),COALESCE(s.trust_score,0.5),
		(SELECT COUNT(*) FROM price_reports p WHERE p.submitter_id=u.id),
		(SELECT COUNT(*) FROM price_reports p WHERE p.submitter_id=u.id AND p.status='accepted')
		FROM users u LEFT JOIN price_submitters s ON s.user_id=u.id WHERE u.id=$1 AND u.deleted_at IS NULL`, userID).
		Scan(&result.ID, &result.Role, &result.Verified, &result.TrustScore, &result.TotalReports, &result.AcceptedReports)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, models.ErrNotFound
	}
	return &result, err
}

func (r *PostgresSubmitterRepository) Save(ctx context.Context, value *models.Submitter) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO price_submitters (user_id,role,verified,trust_score,total_reports,accepted_reports)
		VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (user_id) DO UPDATE SET role=EXCLUDED.role,
		verified=EXCLUDED.verified,trust_score=EXCLUDED.trust_score,total_reports=EXCLUDED.total_reports,accepted_reports=EXCLUDED.accepted_reports`,
		value.ID, value.Role, value.Verified, value.TrustScore, value.TotalReports, value.AcceptedReports)
	return err
}
