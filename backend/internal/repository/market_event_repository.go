package repository

import (
	"backend/internal/models"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ============================================================
// MARKET EVENT REPOSITORY
//
// Responsibility:
// - Record marketplace activity.
// - Track product listing views.
// - Track marketplace searches.
// - Track product price history.
// - Track external market prices.
// - Calculate demand summaries.
// - Calculate marketplace activity trends.
// - Track checkout abandonment.
// - Track LGA relationships.
//
// Current architecture:
//
// products
// users
// organizations
// product_id
// user_id
//
// Products may belong to:
//
// owner_type = "user"
// owner_type = "organization"
// ============================================================

type MarketEventRepository struct {
	db *sql.DB
}

// ============================================================
// CREATE REPOSITORY
// ============================================================

func NewMarketEventRepository(db *sql.DB) *MarketEventRepository {
	return &MarketEventRepository{
		db: db,
	}
}

// ============================================================
// DEMAND SUMMARY
// ============================================================

// DemandSummary calculates marketplace demand statistics for
// a particular product name.
func (r *MarketEventRepository) DemandSummary(
	produceType string,
	since,
	previousSince time.Time,
) (models.ProduceDemandSummary, error) {

	var summary models.ProduceDemandSummary
	var currentTotal int
	var previousTotal int

	produceType = strings.TrimSpace(produceType)

	if produceType == "" {
		return summary, fmt.Errorf("produce type is required")
	}

	summary.ProduceType = produceType

	err := r.db.QueryRow(`
		SELECT
			COALESCE(
				SUM(
					CASE
						WHEN e.event_type = 'listing_view'
							AND e.created_at >= $2
						THEN 1
						ELSE 0
					END
				),
				0
			),

			COALESCE(
				SUM(
					CASE
						WHEN e.event_type = 'cart_item_added'
							AND e.created_at >= $2
						THEN 1
						ELSE 0
					END
				),
				0
			),

			COALESCE(
				SUM(
					CASE
						WHEN e.event_type IN (
							'negotiation_started',
							'negotiation_created'
						)
						AND e.created_at >= $2
						THEN 1
						ELSE 0
					END
				),
				0
			),

			COALESCE(
				SUM(
					CASE
						WHEN e.event_type = 'checkout_completed'
							AND e.created_at >= $2
						THEN 1
						ELSE 0
					END
				),
				0
			),

			COALESCE(
				SUM(
					CASE
						WHEN e.event_type = 'checkout_completed'
							AND e.created_at >= $2
						THEN COALESCE(
							NULLIF(e.metadata->>'quantity', '')::double precision,
							0
						)
						ELSE 0
					END
				),
				0
			),

			COALESCE(
				SUM(
					CASE
						WHEN e.event_type = 'cart_item_added'
							AND e.created_at >= $2
						THEN COALESCE(
							NULLIF(e.metadata->>'quantity', '')::double precision,
							0
						)
						ELSE 0
					END
				),
				0
			),

			(
				SELECT COUNT(*)
				FROM products p
				WHERE LOWER(p.name) = LOWER($1)
				  AND p.status = 'active'
				  AND p.quantity > 0
			)

		FROM market_events e
		JOIN products p
			ON p.id = e.product_id
		WHERE LOWER(p.name) = LOWER($1)
		  AND e.created_at >= $3
	`,
		produceType,
		since,
		previousSince,
	).Scan(
		&summary.Views,
		&summary.CartAdds,
		&summary.Negotiations,
		&summary.Orders,
		&summary.UnitsOrdered,
		&summary.UnitsAdded,
		&summary.CurrentListings,
	)

	if err != nil {
		return summary, fmt.Errorf(
			"calculate demand summary: %w",
			err,
		)
	}

	err = r.db.QueryRow(`
		SELECT
			COUNT(*) FILTER (
				WHERE e.created_at >= $2
			),

			COUNT(*) FILTER (
				WHERE e.created_at >= $3
				  AND e.created_at < $2
			)

		FROM market_events e
		JOIN products p
			ON p.id = e.product_id

		WHERE LOWER(p.name) = LOWER($1)
	`,
		produceType,
		since,
		previousSince,
	).Scan(
		&currentTotal,
		&previousTotal,
	)

	if err != nil {
		return summary, fmt.Errorf(
			"calculate demand trend: %w",
			err,
		)
	}

	summary.TrendPercent = calculateTrendPercent(
		currentTotal,
		previousTotal,
	)

	totalActivity :=
		summary.Views +
			summary.CartAdds +
			summary.Negotiations +
			summary.Orders

	summary.Activity = classifyActivity(totalActivity)

	return summary, nil
}

// ============================================================
// GROUPED DEMAND SUMMARIES
// ============================================================

func (r *MarketEventRepository) GroupedDemandSummaries(
	userID int,
	location string,
	since,
	previous time.Time,
) ([]models.ProduceDemandSummary, error) {

	if userID <= 0 {
		return nil, fmt.Errorf("user id is required")
	}

	rows, err := r.db.Query(`
		SELECT DISTINCT p.name
		FROM products p
		WHERE p.owner_type = 'user'
		  AND p.owner_id = $1
		ORDER BY p.name
	`,
		userID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"load user product names: %w",
			err,
		)
	}

	defer rows.Close()

	var names []string

	for rows.Next() {
		var name string

		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf(
				"scan product name: %w",
				err,
			)
		}

		names = append(names, name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate product names: %w",
			err,
		)
	}

	result := make(
		[]models.ProduceDemandSummary,
		0,
		len(names),
	)

	for _, name := range names {

		summary, err := r.DemandSummary(
			name,
			since,
			previous,
		)

		if err != nil {
			return nil, err
		}

		summary.ComparisonLocation = location

		// ========================================================
		// LOCAL PRICE COMPARISON
		//
		// products.community_id references communities.id.
		// We therefore resolve the supplied community name through
		// the communities table instead of using p.community.
		// ========================================================

		var minimum sql.NullFloat64
		var median sql.NullFloat64
		var maximum sql.NullFloat64
		var localCount int

		err = r.db.QueryRow(`
			SELECT
				MIN(p.price),
				PERCENTILE_CONT(0.5)
					WITHIN GROUP (
						ORDER BY p.price
					),
				MAX(p.price),
				COUNT(*)

			FROM products p

			JOIN communities c
				ON c.id = p.community_id

			WHERE LOWER(p.name) = LOWER($1)
			  AND p.status = 'active'
			  AND p.owner_type = 'user'
			  AND p.owner_id <> $3
			  AND LOWER(c.name) = LOWER($2)
		`,
			name,
			strings.TrimSpace(location),
			userID,
		).Scan(
			&minimum,
			&median,
			&maximum,
			&localCount,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"calculate local price comparison for %q: %w",
				name,
				err,
			)
		}

		if localCount > 0 {
			summary.LocalPriceMin = minimum.Float64
			summary.LocalPriceMedian = median.Float64
			summary.LocalPriceMax = maximum.Float64
			summary.PriceAvailable = true
			summary.LocalListings = localCount
		}

		// ========================================================
		// USER LOCATION
		//
		// users.state is text.
		// products.state_id references states.id.
		//
		// Resolve the user's state name through states.name.
		// ========================================================

		err = r.db.QueryRow(`
			SELECT
				COUNT(*),
				COALESCE(
					PERCENTILE_CONT(0.5)
					WITHIN GROUP (
						ORDER BY p.price
					),
					0
				)

			FROM products p

			JOIN states st
				ON st.id = p.state_id

			WHERE p.status = 'active'
			  AND LOWER(p.name) = LOWER($1)
			  AND p.owner_type = 'user'
			  AND p.owner_id <> $2
			  AND LOWER(st.name) = LOWER(
					(
						SELECT u.state
						FROM users u
						WHERE u.id = $2
					)
			  )
		`,
			name,
			userID,
		).Scan(
			&summary.StateListings,
			&summary.StatePriceMedian,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"calculate state market comparison for %q: %w",
				name,
				err,
			)
		}

		// ========================================================
		// NEARBY LGA LISTINGS
		// ========================================================

		err = r.db.QueryRow(`
			SELECT
				COUNT(*),
				COALESCE(
					PERCENTILE_CONT(0.5)
					WITHIN GROUP (
						ORDER BY p.price
					),
					0
				)

			FROM products p

			JOIN users seller
				ON seller.id = p.owner_id
				AND p.owner_type = 'user'

			WHERE p.status = 'active'
			  AND LOWER(p.name) = LOWER($1)
			  AND p.owner_id <> $2

			  AND LOWER(seller.lga) IN (
					SELECT LOWER(ln.neighbor_lga)
					FROM lga_neighbors ln

					WHERE LOWER(ln.state) = LOWER(
						(
							SELECT u.state
							FROM users u
							WHERE u.id = $2
						)
					)

					AND LOWER(ln.lga) = LOWER(
						(
							SELECT u.lga
							FROM users u
							WHERE u.id = $2
						)
					)
			  )
		`,
			name,
			userID,
		).Scan(
			&summary.NearbyListings,
			&summary.NearbyPriceMedian,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"calculate nearby market comparison for %q: %w",
				name,
				err,
			)
		}

		// ========================================================
		// NATIONAL LISTINGS
		// ========================================================

		err = r.db.QueryRow(`
			SELECT
				COUNT(*),
				COALESCE(
					PERCENTILE_CONT(0.5)
					WITHIN GROUP (
						ORDER BY p.price
					),
					0
				)

			FROM products p

			WHERE p.status = 'active'
			  AND LOWER(p.name) = LOWER($1)

			  AND NOT (
					p.owner_type = 'user'
					AND p.owner_id = $2
			  )
		`,
			name,
			userID,
		).Scan(
			&summary.NationalListings,
			&summary.NationalPriceMedian,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"calculate national market comparison for %q: %w",
				name,
				err,
			)
		}

		// ========================================================
		// COMPARISON SCOPE
		// ========================================================

		switch {
		case summary.LocalListings > 0:
			summary.ComparisonScope = "Local LGA"

		case summary.StateListings > 0:
			summary.ComparisonScope = "State"

		case summary.NationalListings > 0:
			summary.ComparisonScope = "National"

		default:
			summary.ComparisonScope = "No independent listings"
		}

		// ========================================================
		// USER'S CURRENT PRICE
		// ========================================================

		err = r.db.QueryRow(`
			SELECT
				COALESCE(AVG(p.price), 0)

			FROM products p

			WHERE p.owner_type = 'user'
			  AND p.owner_id = $1
			  AND LOWER(p.name) = LOWER($2)
			  AND p.status = 'active'
		`,
			userID,
			name,
		).Scan(&summary.FarmerPrice)

		if err != nil {
			return nil, fmt.Errorf(
				"calculate current user price for %q: %w",
				name,
				err,
			)
		}

		// ========================================================
		// EXTERNAL MARKET PRICE
		// ========================================================

		err = r.db.QueryRow(`
			SELECT
				COUNT(*),
				COALESCE(
					PERCENTILE_CONT(0.5)
					WITHIN GROUP (
						ORDER BY price_per_unit
					),
					0
				)

			FROM external_market_prices

			WHERE LOWER(produce_type) = LOWER($1)
			  AND collected_at >= $2

			  AND (
					state IS NULL

					OR LOWER(state) = LOWER(
						(
							SELECT u.state
							FROM users u
							WHERE u.id = $3
						)
					)
			  )
		`,
			name,
			since,
			userID,
		).Scan(
			&summary.ExternalListings,
			&summary.ExternalPriceMedian,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"calculate external market price for %q: %w",
				name,
				err,
			)
		}

		summary.ExternalPriceAvailable =
			summary.ExternalListings > 0 &&
				summary.ExternalPriceMedian > 0

		// ========================================================
		// PRICE POSITION
		// ========================================================

		if summary.PriceAvailable &&
			summary.LocalPriceMedian > 0 &&
			summary.FarmerPrice > 0 {

			summary.PriceDifferencePercent =
				(summary.FarmerPrice -
					summary.LocalPriceMedian) *
					100 /
					summary.LocalPriceMedian

			switch {
			case summary.PriceDifferencePercent > 5:
				summary.PricePosition = "Above local median"

			case summary.PriceDifferencePercent < -5:
				summary.PricePosition = "Below local median"

			default:
				summary.PricePosition = "Near local median"
			}
		}

		// ========================================================
		// RECOMMENDATION
		// ========================================================

		activityTotal :=
			summary.Views +
				summary.CartAdds +
				summary.Orders +
				summary.Negotiations

		if summary.ComparisonScope == "No independent listings" &&
			activityTotal == 0 {

			summary.Recommendation =
				"No buyer activity yet. Add a clear photo and keep availability current."
		} else {
			summary.Recommendation =
				demandRecommendation(summary)
		}

		result = append(result, summary)
	}

	return result, nil
}

// ============================================================
// USER DEMAND SUMMARY
// ============================================================

func (r *MarketEventRepository) UserDemandSummary(
	userID int,
	location string,
	since,
	previousSince time.Time,
) (models.ProduceDemandSummary, error) {

	var summary models.ProduceDemandSummary
	var current int
	var previous int

	if userID <= 0 {
		return summary, fmt.Errorf("user id is required")
	}

	err := r.db.QueryRow(`
		SELECT
			COUNT(*) FILTER (
				WHERE e.created_at >= $2
			),

			COUNT(*) FILTER (
				WHERE e.created_at >= $3
				  AND e.created_at < $2
			),

			COUNT(*) FILTER (
				WHERE e.event_type = 'listing_view'
				  AND e.created_at >= $2
			),

			COUNT(*) FILTER (
				WHERE e.event_type = 'cart_item_added'
				  AND e.created_at >= $2
			),

			COUNT(*) FILTER (
				WHERE e.event_type IN (
					'negotiation_started',
					'negotiation_created'
				)
				AND e.created_at >= $2
			),

			COUNT(*) FILTER (
				WHERE e.event_type = 'checkout_completed'
				  AND e.created_at >= $2
			),

			COALESCE(
				SUM(
					CASE
						WHEN e.event_type = 'checkout_completed'
							AND e.created_at >= $2
						THEN COALESCE(
							NULLIF(e.metadata->>'quantity', '')::double precision,
							0
						)
						ELSE 0
					END
				),
				0
			),

			COUNT(DISTINCT p.id)

		FROM market_events e

		JOIN products p
			ON p.id = e.product_id

		WHERE p.owner_type = 'user'
		  AND p.owner_id = $1
	`,
		userID,
		since,
		previousSince,
	).Scan(
		&current,
		&previous,
		&summary.Views,
		&summary.CartAdds,
		&summary.Negotiations,
		&summary.Orders,
		&summary.UnitsOrdered,
		&summary.CurrentListings,
	)

	if err != nil {
		return summary, fmt.Errorf(
			"calculate user demand summary: %w",
			err,
		)
	}

	// ========================================================
	// MARKETPLACE SEARCHES
	// ========================================================

	err = r.db.QueryRow(`
		SELECT COUNT(*)
		FROM marketplace_searches
		WHERE created_at >= $1
	`,
		since,
	).Scan(&summary.Searches)

	if err != nil {
		return summary, fmt.Errorf(
			"calculate marketplace searches: %w",
			err,
		)
	}

	if strings.TrimSpace(location) != "" {
		err = r.db.QueryRow(`
			SELECT COUNT(*)
			FROM marketplace_searches
			WHERE created_at >= $1
			  AND LOWER(location) = LOWER($2)
		`,
			since,
			location,
		).Scan(&summary.Searches)

		if err != nil {
			return summary, fmt.Errorf(
				"calculate local marketplace searches: %w",
				err,
			)
		}
	}

	// ========================================================
	// LOCAL LISTINGS
	//
	// Resolve community through communities.name.
	// ========================================================

	if strings.TrimSpace(location) != "" {
		err = r.db.QueryRow(`
			SELECT COUNT(*)
			FROM products p
			JOIN communities c
				ON c.id = p.community_id
			WHERE p.status = 'active'
			  AND LOWER(c.name) = LOWER($1)
		`,
			location,
		).Scan(&summary.LocalListings)

		if err != nil {
			return summary, fmt.Errorf(
				"calculate local listings: %w",
				err,
			)
		}
	}

	summary.ComparisonLocation = location

	// ========================================================
	// NEGOTIATIONS
	// ========================================================

	err = r.db.QueryRow(`
		SELECT
			COUNT(*) FILTER (
				WHERE n.status = 'open'
			),

			COUNT(*) FILTER (
				WHERE n.status = 'accepted'
			),

			COUNT(*) FILTER (
				WHERE n.status = 'rejected'
			),

			COALESCE(
				SUM(n.quantity),
				0
			)

		FROM negotiations n

		JOIN products p
			ON p.id = n.product_id

		WHERE p.owner_type = 'user'
		  AND p.owner_id = $1
		  AND n.created_at >= $2
	`,
		userID,
		since,
	).Scan(
		&summary.NegotiationsActive,
		&summary.NegotiationsAccepted,
		&summary.NegotiationsRejected,
		&summary.NegotiatedUnits,
	)

	if err != nil {
		return summary, fmt.Errorf(
			"calculate negotiation statistics: %w",
			err,
		)
	}

	// ========================================================
	// AVERAGE OFFER PRICE
	// ========================================================

	err = r.db.QueryRow(`
		SELECT
			COALESCE(
				AVG(m.offer_price),
				0
			)

		FROM negotiation_messages m

		JOIN negotiations n
			ON n.id = m.negotiation_id

		JOIN products p
			ON p.id = n.product_id

		WHERE p.owner_type = 'user'
		  AND p.owner_id = $1
		  AND m.message_type = 'offer'
		  AND m.offer_price > 0
		  AND m.created_at >= $2
	`,
		userID,
		since,
	).Scan(&summary.AverageOfferPrice)

	if err != nil {
		return summary, fmt.Errorf(
			"calculate average offer price: %w",
			err,
		)
	}

	// ========================================================
	// PRODUCT LIFECYCLE
	// ========================================================

	var initial float64
	var remaining float64

	err = r.db.QueryRow(`
		SELECT
			COALESCE(
				SUM(initial_listed_quantity),
				0
			),

			COALESCE(
				SUM(quantity),
				0
			),

			COALESCE(
				AVG(
					EXTRACT(
						EPOCH FROM (
							first_order_at -
							first_listed_at
						)
					) / 3600
				) FILTER (
					WHERE first_order_at IS NOT NULL
					  AND first_listed_at IS NOT NULL
				),
				0
			),

			COALESCE(
				AVG(
					EXTRACT(
						EPOCH FROM (
							sold_out_at -
							first_listed_at
						)
					) / 3600
				) FILTER (
					WHERE sold_out_at IS NOT NULL
					  AND first_listed_at IS NOT NULL
				),
				0
			)

		FROM products

		WHERE owner_type = 'user'
		  AND owner_id = $1
	`,
		userID,
	).Scan(
		&initial,
		&remaining,
		&summary.TimeToFirstOrderHours,
		&summary.TimeToSellOutHours,
	)

	if err != nil {
		return summary, fmt.Errorf(
			"calculate product lifecycle: %w",
			err,
		)
	}

	summary.SellThroughRate =
		calculateSellThrough(
			initial,
			remaining,
		)

	summary.LifecycleDataAvailable =
		initial > 0

	// ========================================================
	// NEGOTIATION CONVERSION
	// ========================================================

	totalNegotiations :=
		summary.NegotiationsActive +
			summary.NegotiationsAccepted +
			summary.NegotiationsRejected

	if totalNegotiations > 0 {
		summary.NegotiationConversionRate =
			float64(summary.NegotiationsAccepted) *
				100 /
				float64(totalNegotiations)
	}

	// ========================================================
	// DEMAND SCORE
	// ========================================================

	summary.DemandScore =
		calculateDemandScore(
			summary.Views,
			summary.CartAdds,
			summary.Orders,
			totalNegotiations,
		)

	summary.ProduceType = "Your products"

	summary.TrendPercent =
		calculateTrendPercent(
			current,
			previous,
		)

	summary.Activity =
		classifyActivity(current)

	return summary, nil
}

// ============================================================
// MIN FLOAT
// ============================================================

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}

	return b
}

// ============================================================
// CLASSIFY ACTIVITY
// ============================================================

func classifyActivity(total int) string {
	switch {
	case total < 3:
		return "Not enough data"

	case total < 10:
		return "Early signal"

	case total < 25:
		return "Moderate activity"

	default:
		return "Strong activity"
	}
}

// ============================================================
// CALCULATE TREND
// ============================================================

func calculateTrendPercent(
	current,
	previous int,
) float64 {

	if previous <= 0 {
		return 0
	}

	return float64(current-previous) *
		100 /
		float64(previous)
}

// ============================================================
// CALCULATE DEMAND SCORE
// ============================================================

func calculateDemandScore(
	views,
	cartAdds,
	orders,
	negotiations int,
) float64 {

	return minFloat(
		float64(views)/20*100,
		100,
	)*0.20 +

		minFloat(
			float64(cartAdds)/10*100,
			100,
		)*0.25 +

		minFloat(
			float64(orders)/5*100,
			100,
		)*0.35 +

		minFloat(
			float64(negotiations)/10*100,
			100,
		)*0.20
}

// ============================================================
// CALCULATE SELL THROUGH
// ============================================================

func calculateSellThrough(
	initial,
	remaining float64,
) float64 {

	if initial <= 0 {
		return 0
	}

	rate :=
		(initial-remaining) *
			100 /
			initial

	if rate < 0 {
		return 0
	}

	if rate > 100 {
		return 100
	}

	return rate
}

// ============================================================
// DEMAND RECOMMENDATION
// ============================================================

func demandRecommendation(
	s models.ProduceDemandSummary,
) string {

	if s.PriceDifferencePercent > 15 &&
		s.Orders == 0 {

		return fmt.Sprintf(
			"Your price is %.0f%% above the local median. Consider reviewing it.",
			s.PriceDifferencePercent,
		)
	}

	if s.Views+
		s.CartAdds+
		s.Orders+
		s.Negotiations == 0 {

		return "No recent activity. Review the listing price, photo, and availability."
	}

	if s.CartAdds > 0 &&
		s.Orders == 0 {

		return "Buyers are showing intent but not completing orders. Review price and delivery terms."
	}

	if s.Orders > 0 {
		return "Demand is converting into orders. Keep quantity and availability current."
	}

	return "Monitor activity and keep listing details current."
}

// ============================================================
// RECORD LISTING VIEW
// ============================================================

func (r *MarketEventRepository) RecordListingView(
	productID int,
	userID *int,
	sessionID string,
	window time.Duration,
) error {

	if productID <= 0 {
		return fmt.Errorf("product id is required")
	}

	if window <= 0 {
		window = 30 * time.Minute
	}

	_, err := r.db.Exec(`
		INSERT INTO market_events (
			event_type,
			product_id,
			user_id,
			session_id,
			metadata
		)

		SELECT
			'listing_view',
			$1,
			$2,
			NULLIF($3, ''),
			'{}'::jsonb

		WHERE NOT EXISTS (
			SELECT 1
			FROM market_events

			WHERE event_type = 'listing_view'
			  AND product_id = $1
			  AND created_at >=
					CURRENT_TIMESTAMP -
					$4::interval

			  AND (
					(
						user_id IS NOT NULL
						AND user_id = $2
					)

					OR

					(
						session_id IS NOT NULL
						AND session_id =
							NULLIF($3, '')
					)
			  )
		)
	`,
		productID,
		userID,
		strings.TrimSpace(sessionID),
		fmt.Sprintf(
			"%f seconds",
			window.Seconds(),
		),
	)

	if err != nil {
		return fmt.Errorf(
			"record listing view: %w",
			err,
		)
	}

	return nil
}

// ============================================================
// RECORD MARKET EVENT
// ============================================================

func (r *MarketEventRepository) Record(
	event *models.MarketEvent,
) error {

	if event == nil {
		return fmt.Errorf("market event is nil")
	}

	event.EventType =
		strings.TrimSpace(event.EventType)

	if event.EventType == "" {
		return fmt.Errorf("event type is required")
	}

	if event.ProductID != nil &&
		*event.ProductID <= 0 {

		return fmt.Errorf("invalid product id")
	}

	if event.UserID != nil &&
		*event.UserID <= 0 {

		return fmt.Errorf("invalid user id")
	}

	event.SessionID =
		strings.TrimSpace(event.SessionID)

	metadata :=
		strings.TrimSpace(event.Metadata)

	if metadata == "" {
		metadata = "{}"
	}

	_, err := r.db.Exec(`
		INSERT INTO market_events (
			event_type,
			product_id,
			user_id,
			session_id,
			metadata
		)
		VALUES (
			$1,
			$2,
			$3,
			NULLIF($4, ''),
			$5::jsonb
		)
	`,
		event.EventType,
		event.ProductID,
		event.UserID,
		event.SessionID,
		metadata,
	)

	if err != nil {
		return fmt.Errorf(
			"record market event: %w",
			err,
		)
	}

	return nil
}

// ============================================================
// RECORD MARKETPLACE SEARCH
// ============================================================

func (r *MarketEventRepository) RecordSearch(
	term,
	normalized,
	location string,
	userID *int,
	sessionID string,
) error {

	term = strings.TrimSpace(term)

	if term == "" {
		return nil
	}

	normalized = strings.TrimSpace(normalized)

	if normalized == "" {
		normalized = term
	}

	_, err := r.db.Exec(`
		INSERT INTO marketplace_searches (
			search_term,
			normalized_produce,
			location,
			user_id,
			session_id
		)
		VALUES (
			$1,
			NULLIF($2, ''),
			NULLIF($3, ''),
			$4,
			NULLIF($5, '')
		)
	`,
		term,
		strings.ToLower(normalized),
		strings.TrimSpace(location),
		userID,
		strings.TrimSpace(sessionID),
	)

	if err != nil {
		return fmt.Errorf(
			"record marketplace search: %w",
			err,
		)
	}

	return nil
}

// ============================================================
// RECORD PRODUCT PRICE
// ============================================================

func (r *MarketEventRepository) RecordPrice(
	productID int,
	price float64,
	unit,
	source string,
) error {

	if productID <= 0 {
		return fmt.Errorf("product id is required")
	}

	if price <= 0 {
		return fmt.Errorf("price must be greater than zero")
	}

	unit = strings.TrimSpace(unit)

	if unit == "" {
		return fmt.Errorf("unit is required")
	}

	source = strings.TrimSpace(source)

	if source == "" {
		source = "Agro-Shield"
	}

	_, err := r.db.Exec(`
		INSERT INTO produce_price_history (
			product_id,
			price_per_unit,
			unit,
			source
		)
		VALUES ($1, $2, $3, $4)
	`,
		productID,
		price,
		unit,
		source,
	)

	if err != nil {
		return fmt.Errorf(
			"record product price: %w",
			err,
		)
	}

	return nil
}

// ============================================================
// RECORD EXTERNAL PRICE
// ============================================================

func (r *MarketEventRepository) RecordExternalPrice(
	produce string,
	price float64,
	unit,
	provider,
	market,
	lga,
	state,
	sourceURL string,
	collectedAt time.Time,
) error {

	if err := validateExternalPrice(
		produce,
		price,
		unit,
		provider,
		market,
		collectedAt,
	); err != nil {
		return err
	}

	_, err := r.db.Exec(`
		INSERT INTO external_market_prices (
			produce_type,
			price_per_unit,
			unit,
			provider,
			market_name,
			lga,
			state,
			collected_at,
			source_url
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			NULLIF($6, ''),
			NULLIF($7, ''),
			$8,
			NULLIF($9, '')
		)
	`,
		strings.TrimSpace(produce),
		price,
		strings.TrimSpace(unit),
		strings.TrimSpace(provider),
		strings.TrimSpace(market),
		strings.TrimSpace(lga),
		strings.TrimSpace(state),
		collectedAt,
		strings.TrimSpace(sourceURL),
	)

	if err != nil {
		return fmt.Errorf(
			"record external market price: %w",
			err,
		)
	}

	return nil
}

// ============================================================
// UPSERT LGA NEIGHBOR
// ============================================================

func (r *MarketEventRepository) UpsertLGANeighbor(
	state,
	lga,
	neighbor string,
) error {

	state = strings.TrimSpace(state)
	lga = strings.TrimSpace(lga)
	neighbor = strings.TrimSpace(neighbor)

	if state == "" ||
		lga == "" ||
		neighbor == "" {

		return fmt.Errorf(
			"invalid LGA neighbor relationship",
		)
	}

	if strings.EqualFold(lga, neighbor) {
		return fmt.Errorf(
			"LGA cannot be its own neighbor",
		)
	}

	_, err := r.db.Exec(`
		INSERT INTO lga_neighbors (
			state,
			lga,
			neighbor_lga
		)
		VALUES ($1, $2, $3)

		ON CONFLICT (
			state,
			lga,
			neighbor_lga
		)
		DO NOTHING
	`,
		state,
		lga,
		neighbor,
	)

	if err != nil {
		return fmt.Errorf(
			"upsert LGA neighbor: %w",
			err,
		)
	}

	return nil
}

// ============================================================
// VALIDATE EXTERNAL PRICE
// ============================================================

func validateExternalPrice(
	produce string,
	price float64,
	unit,
	provider,
	market string,
	collectedAt time.Time,
) error {

	if strings.TrimSpace(produce) == "" {
		return fmt.Errorf(
			"produce type is required",
		)
	}

	if price <= 0 {
		return fmt.Errorf(
			"external price must be greater than zero",
		)
	}

	if strings.TrimSpace(unit) == "" {
		return fmt.Errorf(
			"price unit is required",
		)
	}

	if strings.TrimSpace(provider) == "" {
		return fmt.Errorf(
			"price provider is required",
		)
	}

	if strings.TrimSpace(market) == "" {
		return fmt.Errorf(
			"market name is required",
		)
	}

	if collectedAt.IsZero() {
		return fmt.Errorf(
			"collection time is required",
		)
	}

	return nil
}

// ============================================================
// RECONCILE CHECKOUT ABANDONMENTS
// ============================================================

func (r *MarketEventRepository) ReconcileCheckoutAbandonments(
	timeout time.Duration,
) error {

	if timeout <= 0 {
		timeout = 24 * time.Hour
	}

	_, err := r.db.Exec(`
		INSERT INTO market_events (
			event_type,
			user_id,
			session_id,
			metadata,
			created_at
		)

		SELECT
			'checkout_abandoned',
			s.user_id,
			s.session_id,

			jsonb_build_object(
				'checkout_started_event_id',
				s.id
			),

			CURRENT_TIMESTAMP

		FROM market_events s

		WHERE s.event_type = 'checkout_started'

		  AND s.created_at <
				CURRENT_TIMESTAMP -
				$1::interval

		  AND NOT EXISTS (
				SELECT 1
				FROM market_events c

				WHERE c.event_type =
					'checkout_completed'

				  AND c.created_at >=
						s.created_at

				  AND (
						(
							s.user_id IS NOT NULL
							AND c.user_id =
								s.user_id
						)

						OR

						(
							s.session_id IS NOT NULL
							AND c.session_id =
								s.session_id
						)
				  )
		  )

		  AND NOT EXISTS (
				SELECT 1
				FROM market_events a

				WHERE a.event_type =
					'checkout_abandoned'

				  AND a.metadata->>
						'checkout_started_event_id'
						= s.id::text
		  )
	`,
		fmt.Sprintf(
			"%f seconds",
			timeout.Seconds(),
		))

	if err != nil {
		return fmt.Errorf(
			"reconcile checkout abandonments: %w",
			err,
		)
	}

	return nil
}