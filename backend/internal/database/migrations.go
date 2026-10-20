package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// ============================================================
// MIGRATION DIRECTORY
// ============================================================
//
// All Agro-Shield database migrations live here.
//
// Example:
//
// backend/migrations/
//     001_users.sql
//     002_farmer_type.sql
//     003_users_verification.sql
//     ...
//
// ============================================================

const migrationsDir = "migrations"

// ============================================================
// 1. RUN DATABASE MIGRATIONS
// ============================================================
//
// Flow:
//
//     RunMigration()
//           ↓
//     Optional development reset
//           ↓
//     Create schema_migrations
//           ↓
//     Find .sql files
//           ↓
//     Sort migrations
//           ↓
//     Check already-applied migrations
//           ↓
//     Apply new migrations
//           ↓
//     Record successful migration
//
// ============================================================

func RunMigration(db *sql.DB) error {

	log.Println("🔄 Starting database migrations...")

	// ----------------------------------------------------------
	// Step 1:
	// Optional database reset
	// ----------------------------------------------------------
	//
	// This is ONLY for rebuilding the development database.
	//
	// It runs only when:
	//
	//     RESET_DATABASE=true
	//
	// is explicitly set.
	//
	// This prevents accidental database destruction during
	// normal application startup.
	// ----------------------------------------------------------

	if os.Getenv("RESET_DATABASE") == "true" {

		log.Println("⚠️ RESET_DATABASE=true")
		log.Println("🧹 Resetting PostgreSQL public schema...")

		if err := resetDatabase(db); err != nil {
			return err
		}

		log.Println("✅ PostgreSQL public schema reset successfully")
	}

	// ----------------------------------------------------------
	// Step 2:
	// Create migration tracking table
	// ----------------------------------------------------------

	if err := createMigrationTable(db); err != nil {
		return err
	}

	// ----------------------------------------------------------
	// Step 3:
	// Find migration files
	// ----------------------------------------------------------

	migrationFiles, err := getMigrationFiles()
	if err != nil {
		return err
	}

	if len(migrationFiles) == 0 {
		log.Println("⚠️ No migration files found")
		return nil
	}

	log.Printf(
		"📦 Found %d migration files",
		len(migrationFiles),
	)

	// ----------------------------------------------------------
	// Step 4:
	// Process migrations one by one
	// ----------------------------------------------------------

	for _, filename := range migrationFiles {

		// ------------------------------------------------------
		// Extract migration version
		// ------------------------------------------------------
		//
		// Example:
		//
		// 009_negotiation.sql
		//
		// becomes:
		//
		// 9
		// ------------------------------------------------------

		version, err := migrationVersion(filename)
		if err != nil {
			return err
		}

		// ------------------------------------------------------
		// Check whether migration was already applied
		// ------------------------------------------------------

		applied, err := migrationApplied(db, version)
		if err != nil {
			return err
		}

		if applied {
			log.Printf(
				"⏭️ Migration already applied: %s",
				filename,
			)

			continue
		}

		// ------------------------------------------------------
		// Apply migration
		// ------------------------------------------------------

		if err := applyMigration(
			db,
			version,
			filename,
		); err != nil {
			return err
		}
	}

	log.Println("✅ All database migrations completed successfully")

	return nil
}

// ============================================================
// 2. RESET DEVELOPMENT DATABASE
// ============================================================
//
// This completely removes the public schema and recreates it.
//
// IMPORTANT:
//
// This function is NEVER called automatically.
//
// It only runs when:
//
//     RESET_DATABASE=true
//
// is explicitly provided.
//
// This is useful while we are rebuilding the Agro-Shield
// database schema from scratch.
//
// ============================================================

func resetDatabase(db *sql.DB) error {

	const query = `
		DROP SCHEMA IF EXISTS public CASCADE;

		CREATE SCHEMA public;
	`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf(
			"failed to reset PostgreSQL public schema: %w",
			err,
		)
	}

	return nil
}

// ============================================================
// 3. CREATE MIGRATION TRACKING TABLE
// ============================================================
//
// PostgreSQL uses this table to remember which migrations
// have already been executed.
//
// Example:
//
//     version | filename
//     --------+--------------------------
//       1     | 001_users.sql
//       2     | 002_farmer_type.sql
//       3     | 003_users_verification.sql
//
// ============================================================

func createMigrationTable(db *sql.DB) error {

	const query = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id INTEGER GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,

			version INTEGER NOT NULL UNIQUE,

			filename TEXT NOT NULL UNIQUE,

			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf(
			"failed to create schema_migrations table: %w",
			err,
		)
	}

	return nil
}

// ============================================================
// 4. FIND MIGRATION FILES
// ============================================================
//
// Reads:
//
// backend/migrations/
//
// and returns only .sql files.
//
// ============================================================

func getMigrationFiles() ([]string, error) {

	entries, err := os.ReadDir(migrationsDir)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to read migrations directory %q: %w",
			migrationsDir,
			err,
		)
	}

	var migrationFiles []string

	for _, entry := range entries {

		// Ignore directories.
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		// Only process SQL files.
		if !strings.HasSuffix(filename, ".sql") {
			continue
		}

		migrationFiles = append(
			migrationFiles,
			filename,
		)
	}

	// Sort files.
	//
	// Because migrations are numbered:
	//
	// 001
	// 002
	// 003
	//
	// alphabetical sorting gives us the correct order.

	sort.Strings(migrationFiles)

	return migrationFiles, nil
}

// ============================================================
// 5. GET MIGRATION VERSION
// ============================================================
//
// Example:
//
//     009_negotiation.sql
//
// returns:
//
//     9
//
// ============================================================

func migrationVersion(filename string) (int, error) {

	parts := strings.SplitN(
		filename,
		"_",
		2,
	)

	if len(parts) != 2 {
		return 0, fmt.Errorf(
			"invalid migration filename %q: expected format 001_name.sql",
			filename,
		)
	}

	version, err := strconv.Atoi(parts[0])

	if err != nil {
		return 0, fmt.Errorf(
			"invalid migration version in %q: %w",
			filename,
			err,
		)
	}

	return version, nil
}

// ============================================================
// 6. CHECK WHETHER MIGRATION WAS ALREADY APPLIED
// ============================================================

func migrationApplied(
	db *sql.DB,
	version int,
) (bool, error) {

	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM schema_migrations
			WHERE version = $1
		);
	`

	var exists bool

	if err := db.QueryRow(
		query,
		version,
	).Scan(&exists); err != nil {

		return false, fmt.Errorf(
			"failed to check migration version %d: %w",
			version,
			err,
		)
	}

	return exists, nil
}

// ============================================================
// 7. APPLY ONE MIGRATION
// ============================================================
//
// Important:
//
// The migration itself and the record saying
// "this migration was successful" happen in ONE transaction.
//
// If anything fails:
//
//     migration → ROLLBACK
//
// and the migration is NOT recorded as completed.
//
// ============================================================

func applyMigration(
	db *sql.DB,
	version int,
	filename string,
) error {

	// ----------------------------------------------------------
	// Build migration file path
	// ----------------------------------------------------------

	path := filepath.Join(
		migrationsDir,
		filename,
	)

	// ----------------------------------------------------------
	// Read SQL file
	// ----------------------------------------------------------

	sqlBytes, err := os.ReadFile(path)

	if err != nil {
		return fmt.Errorf(
			"failed to read migration %s: %w",
			filename,
			err,
		)
	}

	migrationSQL := strings.TrimSpace(
		string(sqlBytes),
	)

	if migrationSQL == "" {
		return fmt.Errorf(
			"migration %s is empty",
			filename,
		)
	}

	log.Printf(
		"▶️ Applying migration: %s",
		filename,
	)

	// ----------------------------------------------------------
	// Begin transaction
	// ----------------------------------------------------------

	tx, err := db.Begin()

	if err != nil {
		return fmt.Errorf(
			"failed to begin migration %s: %w",
			filename,
			err,
		)
	}

	// ----------------------------------------------------------
	// Rollback automatically if Commit() has not succeeded.
	// ----------------------------------------------------------

	defer tx.Rollback()

	// ----------------------------------------------------------
	// Execute migration SQL
	// ----------------------------------------------------------

	if _, err := tx.Exec(migrationSQL); err != nil {

		return fmt.Errorf(
			"migration %s failed: %w",
			filename,
			err,
		)
	}

	// ----------------------------------------------------------
	// Record successful migration
	// ----------------------------------------------------------

	const recordMigration = `
		INSERT INTO schema_migrations (
			version,
			filename
		)
		VALUES ($1, $2);
	`

	if _, err := tx.Exec(
		recordMigration,
		version,
		filename,
	); err != nil {

		return fmt.Errorf(
			"failed to record migration %s: %w",
			filename,
			err,
		)
	}

	// ----------------------------------------------------------
	// Commit transaction
	// ----------------------------------------------------------

	if err := tx.Commit(); err != nil {

		return fmt.Errorf(
			"failed to commit migration %s: %w",
			filename,
			err,
		)
	}

	log.Printf(
		"✅ Migration applied: %s",
		filename,
	)

	return nil
}
