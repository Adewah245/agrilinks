// Package testutil creates disposable PostgreSQL databases for integration tests.
package testutil

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"backend/internal/database"
	"github.com/google/uuid"
)

// Database creates a new database using TEST_DATABASE_URL. No existing tables
// are reset. The test role needs CREATEDB; production credentials are not used.
func Database(t *testing.T, migrate bool) (*sql.DB, string) {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if raw == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable PostgreSQL server to run database checks")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatal("TEST_DATABASE_URL must be a PostgreSQL URL")
	}
	admin, err := database.ConnectDB(raw)
	if err != nil {
		t.Fatal(err)
	}
	name := "agrilinks_check_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.Exec(`CREATE DATABASE "` + name + `"`); err != nil {
		admin.Close()
		t.Fatalf("create disposable test database: %v", err)
	}
	u.Path = "/" + name
	db, err := database.ConnectDB(u.String())
	if err != nil {
		admin.Exec(`DROP DATABASE "` + name + `"`)
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(ctx, `DROP DATABASE "`+name+`" WITH (FORCE)`); err != nil {
			t.Errorf("remove disposable test database: %v", err)
		}
		admin.Close()
	})
	if migrate {
		if err := database.RunMigration(db); err != nil {
			t.Fatal(err)
		}
	}
	return db, u.String()
}
