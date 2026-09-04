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
// - Records marketplace activity.
// - Tracks product listing views.
// - Tracks marketplace searches.
// - Tracks product price history.
// - Tracks external market prices.
// - Calculates demand summaries.
// - Calculates marketplace activity trends.
// - Tracks checkout abandonment.
// - Tracks LGA relationships.
//
// IMPORTANT:
//
// The new Agro-Shield architecture uses:
//
// OLD:
// crops
// farmers
// crop_id
// farmer_id
//
// NEW:
// products
// users
// product_id
// user_id
//
// Products can belong to:
//
// owner_type = "user"
// owner_type = "organization"
//
// Therefore marketplace analytics must work with products
// instead of crops.
// ============================================================

type MarketEventRepository struct {
	db *sql.DB
}

// ============================================================
// CREATE REPOSITORY
// ============================================================

// NewMarketEventRepository creates a new repository for
// recording and analysing marketplace events.
func NewMarketEventRepository(db *sql.DB) *MarketEventRepository {
	return &MarketEventRepository{
		db: db,
	}
}

// ============================================================
// DEMAND SUMMARY
//
// Calculates marketplace demand for a particular product type.
//
// The summary includes:
//
// - Views
// - Cart additions
// - Negotiations
// - Orders
// - Units ordered
// - Units added to cart
// - Current listings
// - Activity trend
// ============================================================

// DemandSummary calculates demand statistics for a product
// type during the supplied time periods.
func (r *MarketEventRepository) DemandSummary(
	produceType string,
	since,
	previousSince time.Time,
) (models.ProduceDemandSummary, error) {

	var s models.ProduceDemandSummary
	var currentTotal, previousTotal int

	s.ProduceType = produceType

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
							(e.metadata->>'quantity')::double precision,
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
							(e.metadata->>'quantity')::double precision,
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
		&s.Views,
		&s.CartAdds,
		&s.Negotiations,
		&s.Orders,
		&s.UnitsOrdered,
		&s.UnitsAdded,
		&s.CurrentListings,
	)

	if err != nil {
		return s, err
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
		return s, err
	}

	s.TrendPercent = calculateTrendPercent(
		currentTotal,
		previousTotal,
	)

	total := s.Views +
		s.CartAdds +
		s.Negotiations +
		s.Orders

	s.Activity = classifyActivity(total)

	return s, nil
}

// ============================================================
// GROUPED DEMAND SUMMARIES
//
// Retrieves demand summaries for products owned by a user.
//
// The user may own products directly:
//
// owner_type = "user"
//
// Organization-owned products are not included when querying
// a user's personal products.
// ============================================================

// GroupedDemandSummaries returns demand summaries for the
// products owned by a specific user.
func (r *MarketEventRepository) GroupedDemandSummaries(
	userID int,
	location string,
	since,
	previous time.Time,
) ([]models.ProduceDemandSummary, error) {

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
		return nil, err
	}

	defer rows.Close()

	var names []string

	for rows.Next() {
		var name string

		if err := rows.Scan(&name); err != nil {
			return nil, err
		}

		names = append(names, name)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make(
		[]models.ProduceDemandSummary,
		0,
		len(names),
	)

	for _, name := range names {

		s, err := r.DemandSummary(
			name,
			since,
			previous,
		)

		if err != nil {
			return nil, err
		}

		s.ComparisonLocation = location

		// --------------------------------------------------------
		// LOCAL PRICE COMPARISON
		// --------------------------------------------------------

		var minimum sql.NullFloat64
		var median sql.NullFloat64
		var maximum sql.NullFloat64
		var count int

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
			WHERE LOWER(p.name) = LOWER($1)
			  AND p.status = 'active'
			  AND p.owner_type = 'user'
			  AND p.owner_id <> $3
			  AND LOWER(p.community) = LOWER($2)
		`,
			name,
			location,
			userID,
		).Scan(
			&minimum,
			&median,
			&maximum,
			&count,
		)

		if err != nil {
			return nil, err
		}

		if count > 0 {
			s.LocalPriceMin = minimum.Float64
			s.LocalPriceMedian = median.Float64
			s.LocalPriceMax = maximum.Float64
			s.PriceAvailable = true
		}

		// --------------------------------------------------------
		// STATE LISTINGS
		//
		// Find the user's state through the users table.
		// --------------------------------------------------------

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
			  AND p.owner_type = 'user'
			  AND p.owner_id <> $2
			  AND p.state_id = (
					SELECT u.state
					FROM users u
					WHERE u.id = $2
			  )
		`,
			name,
			userID,
		).Scan(
			&s.StateListings,
			&s.StatePriceMedian,
		)

		if err != nil {
			// The current users schema stores state as TEXT while
			// products.state_id stores a location ID.
			//
			// State-level comparison is therefore handled below
			// using the user's state text when available.
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
				JOIN users u
					ON u.id = p.owner_id
					AND p.owner_type = 'user'
				WHERE p.status = 'active'
				  AND LOWER(p.name) = LOWER($1)
				  AND p.owner_id <> $2
				  AND LOWER(u.state) = LOWER(
						(
							SELECT state
							FROM users
							WHERE id = $2
						)
				  )
			`,
				name,
				userID,
			).Scan(
				&s.StateListings,
				&s.StatePriceMedian,
			)

			if err != nil {
				return nil, err
			}
		}

		// --------------------------------------------------------
		// NEARBY LISTINGS
		// --------------------------------------------------------

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
			JOIN users u
				ON u.id = p.owner_id
				AND p.owner_type = 'user'
			WHERE p.status = 'active'
			  AND LOWER(p.name) = LOWER($1)
			  AND p.owner_id <> $2
			  AND LOWER(u.lga) IN (
					SELECT LOWER(neighbor_lga)
					FROM lga_neighbors
					WHERE LOWER(state) = LOWER(
						(
							SELECT state
							FROM users
							WHERE id = $2
						)
					)
					AND LOWER(lga) = LOWER(
						(
							SELECT lga
							FROM users
							WHERE id = $2
						)
					)
			  )
		`,
			name,
			userID,
		).Scan(
			&s.NearbyListings,
			&s.NearbyPriceMedian,
		)

		if err != nil {
			return nil, err
		}

		// --------------------------------------------------------
		// NATIONAL LISTINGS
		// --------------------------------------------------------

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
			&s.NationalListings,
			&s.NationalPriceMedian,
		)

		if err != nil {
			return nil, err
		}

		s.ComparisonScope = "Local LGA"

		if s.StateListings == 0 &&
			s.NationalListings == 0 {
			s.ComparisonScope = "No independent listings"
		}

		// --------------------------------------------------------
		// USER'S CURRENT PRICE
		// --------------------------------------------------------

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
		).Scan(&s.FarmerPrice)

		if err != nil {
			return nil, err
		}

		// --------------------------------------------------------
		// EXTERNAL MARKET PRICE
		// --------------------------------------------------------

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
							SELECT state
							FROM users
							WHERE id = $3
						)
					)
			  )
		`,
			name,
			since,
			userID,
		).Scan(
			&s.ExternalListings,
			&s.ExternalPriceMedian,
		)

		if err != nil {
			return nil, err
		}

		s.ExternalPriceAvailable =
			s.ExternalListings > 0 &&
				s.ExternalPriceMedian > 0

		// --------------------------------------------------------
		// PRICE POSITION
		// --------------------------------------------------------

		if s.PriceAvailable &&
			s.LocalPriceMedian > 0 &&
			s.FarmerPrice > 0 {

			s.PriceDifferencePercent =
				(s.FarmerPrice - s.LocalPriceMedian) *
					100 /
					s.LocalPriceMedian

			switch {
			case s.PriceDifferencePercent > 5:
				s.PricePosition = "Above local median"

			case s.PriceDifferencePercent < -5:
				s.PricePosition = "Below local median"

			default:
				s.PricePosition = "Near local median"
			}
		}

		// --------------------------------------------------------
		// RECOMMENDATION
		// --------------------------------------------------------

		if s.ComparisonScope == "No independent listings" &&
			s.Views+
				s.CartAdds+
				s.Orders+
				s.Negotiations == 0 {

			s.Recommendation =
				"No buyer activity yet. Add a clear photo and keep availability current."
		} else {
			s.Recommendation = demandRecommendation(s)
		}

		result = append(result, s)
	}

	return result, nil
}

// ============================================================
// USER DEMAND SUMMARY
//
// Provides marketplace demand information for products owned
// by a specific user.
//
// This replaces the old FarmerDemandSummary architecture.
// ============================================================

// UserDemandSummary returns marketplace demand statistics
// for products owned by a specific user.
func (r *MarketEventRepository) UserDemandSummary(
	userID int,
	location string,
	since,
	previousSince time.Time,
) (models.ProduceDemandSummary, error) {

	var s models.ProduceDemandSummary

	var current int
	var previous int

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
							(e.metadata->>'quantity')::double precision,
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
		&s.Views,
		&s.CartAdds,
		&s.Negotiations,
		&s.Orders,
		&s.UnitsOrdered,
		&s.CurrentListings,
	)

	if err != nil {
		return s, err
	}

	// --------------------------------------------------------
	// MARKETPLACE SEARCHES
	// --------------------------------------------------------

	_ = r.db.QueryRow(`
		SELECT COUNT(*)
		FROM marketplace_searches
		WHERE created_at >= $1
	`,
		since,
	).Scan(&s.Searches)

	_ = r.db.QueryRow(`
		SELECT COUNT(*)
		FROM marketplace_searches
		WHERE created_at >= $1
		  AND LOWER(location) = LOWER($2)
	`,
		since,
		location,
	).Scan(&s.Searches)

	// --------------------------------------------------------
	// LOCAL LISTINGS
	// --------------------------------------------------------

	_ = r.db.QueryRow(`
		SELECT COUNT(*)
		FROM products p
		WHERE p.status = 'active'
		  AND LOWER(p.community) = LOWER($1)
	`,
		location,
	).Scan(&s.LocalListings)

	s.ComparisonLocation = location

	// --------------------------------------------------------
	// NEGOTIATIONS
	// --------------------------------------------------------

	_ = r.db.QueryRow(`
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
		&s.NegotiationsActive,
		&s.NegotiationsAccepted,
		&s.NegotiationsRejected,
		&s.NegotiatedUnits,
	)

	// --------------------------------------------------------
	// AVERAGE OFFER PRICE
	// --------------------------------------------------------

	_ = r.db.QueryRow(`
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
	).Scan(&s.AverageOfferPrice)

	// --------------------------------------------------------
	// PRODUCT LIFECYCLE
	// --------------------------------------------------------

	var initial float64
	var remaining float64

	_ = r.db.QueryRow(`
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
		&s.TimeToFirstOrderHours,
		&s.TimeToSellOutHours,
	)

	s.SellThroughRate =
		calculateSellThrough(
			initial,
			remaining,
		)

	s.LifecycleDataAvailable =
		initial > 0

	// --------------------------------------------------------
	// NEGOTIATION CONVERSION
	// --------------------------------------------------------

	totalNegotiations :=
		s.NegotiationsActive +
			s.NegotiationsAccepted +
			s.NegotiationsRejected

	if totalNegotiations > 0 {
		s.NegotiationConversionRate =
			float64(s.NegotiationsAccepted) *
				100 /
				float64(totalNegotiations)
	}

	// --------------------------------------------------------
	// DEMAND SCORE
	// --------------------------------------------------------

	s.DemandScore = calculateDemandScore(
		s.Views,
		s.CartAdds,
		s.Orders,
		totalNegotiations,
	)

	s.ProduceType = "Your products"

	s.TrendPercent =
		calculateTrendPercent(
			current,
			previous,
		)

	s.Activity =
		classifyActivity(current)

	return s, nil
}

// ============================================================
// MIN FLOAT
// ============================================================

// minFloat returns the smaller of two floating-point numbers.
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}

	return b
}

// ============================================================
// CLASSIFY ACTIVITY
// ============================================================

// classifyActivity converts an activity count into a
// human-readable marketplace activity level.
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

// calculateTrendPercent calculates the percentage change
// between the current and previous activity periods.
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

// calculateDemandScore calculates a weighted marketplace
// demand score from 0 to 100.
func calculateDemandScore(
	views,
	cartAdds,
	orders,
	negotiations int,
) float64 {

	return minFloat(
		float64(views)/20*100,
		100,
	)*.20 +

		minFloat(
			float64(cartAdds)/10*100,
			100,
		)*.25 +

		minFloat(
			float64(orders)/5*100,
			100,
		)*.35 +

		minFloat(
			float64(negotiations)/10*100,
			100,
		)*.20
}

// ============================================================
// CALCULATE SELL THROUGH
// ============================================================

// calculateSellThrough calculates the percentage of an
// initially listed quantity that has been sold.
func calculateSellThrough(
	initial,
	remaining float64,
) float64 {

	if initial <= 0 {
		return 0
	}

	rate :=
		(initial - remaining) *
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

// demandRecommendation generates a marketplace recommendation
// based on price and buyer activity.
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
//
// Records one listing view for a product.
//
// A user/session cannot repeatedly generate views for the same
// product inside the deduplication window.
// ============================================================

// RecordListingView records at most one view for a user/session
// and product within the supplied deduplication window.
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
		sessionID,
		fmt.Sprintf(
			"%f seconds",
			window.Seconds(),
		),
	)

	return err
}

// ============================================================
// RECORD MARKET EVENT
// ============================================================

// Record stores a marketplace event in the database.
func (r *MarketEventRepository) Record(
	event *models.MarketEvent,
) error {

	if event == nil {
		return fmt.Errorf("market event is nil")
	}

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
		VALUES ($1, $2, $3, $4, $5)
	`,
		event.EventType,
		event.ProductID,
		event.UserID,
		event.SessionID,
		metadata,
	)

	return err
}

// ============================================================
// RECORD MARKETPLACE SEARCH
// ============================================================

// RecordSearch records a marketplace search performed by
// a user or anonymous session.
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

	if strings.TrimSpace(normalized) == "" {
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
		strings.ToLower(
			strings.TrimSpace(normalized),
		),
		strings.TrimSpace(location),
		userID,
		sessionID,
	)

	return err
}

// ============================================================
// RECORD PRODUCT PRICE
// ============================================================

// RecordPrice records the price history of a product.
func (r *MarketEventRepository) RecordPrice(
	productID int,
	price float64,
	unit,
	source string,
) error {

	if productID <= 0 ||
		price <= 0 ||
		strings.TrimSpace(unit) == "" {

		return fmt.Errorf(
			"invalid price history record",
		)
	}

	if strings.TrimSpace(source) == "" {
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

	return err
}

// ============================================================
// RECORD EXTERNAL PRICE
// ============================================================

// RecordExternalPrice stores a verified external market-price
// observation together with its provenance.
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
		unit,
		provider,
		market,
		strings.TrimSpace(lga),
		strings.TrimSpace(state),
		collectedAt,
		strings.TrimSpace(sourceURL),
	)

	return err
}

// ============================================================
// UPSERT LGA NEIGHBOR
// ============================================================

// UpsertLGANeighbor stores a directional LGA relationship.
// When the relationship is mutual, the caller should store
// both directions.
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
		neighbor == "" ||
		strings.EqualFold(lga, neighbor) {

		return fmt.Errorf(
			"invalid LGA neighbor relationship",
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

	return err
}

// ============================================================
// VALIDATE EXTERNAL PRICE
// ============================================================

// validateExternalPrice validates an external market-price
// observation before it is stored.
func validateExternalPrice(
	produce string,
	price float64,
	unit,
	provider,
	market string,
	collectedAt time.Time,
) error {

	if strings.TrimSpace(produce) == "" ||
		price <= 0 ||
		strings.TrimSpace(unit) == "" ||
		strings.TrimSpace(provider) == "" ||
		strings.TrimSpace(market) == "" ||
		collectedAt.IsZero() {

		return fmt.Errorf(
			"incomplete external price observation",
		)
	}

	return nil
}

// ============================================================
// RECONCILE CHECKOUT ABANDONMENTS
//
// Finds checkout sessions that started but were not completed
// within the configured timeout.
//
// The operation is safe to run repeatedly because it checks
// whether an abandonment event already exists.
// ============================================================

// ReconcileCheckoutAbandonments records abandonment events for
// checkout starts that remained incomplete beyond the timeout.
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

	return err
}
