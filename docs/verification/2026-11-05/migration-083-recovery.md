# Migration 083 startup recovery

The configured database rejected startup because migration 083 had been edited
after it was applied. Its stored SHA-256 checksum was:

`359aa1538cbad6b210a045aac22bf810adfd7e859d548d8d65da3b0e5ac735df`

The exact original SQL was recovered from the local Go build cache. Its bytes
match that checksum. The source file was restored without altering migration
metadata, resetting the database, or bypassing checksum enforcement.

Migration 084 adds the private-photo lookup index and refreshes photo-access
acknowledgments only for pending, unsubmitted withdrawal reviews. It preserves
existing completed-review timestamps, private photo data, withdrawal records,
MFA sessions and reserved money. It cannot reconstruct timestamps already
cleared when the original migration 083 first ran.

Disposable PostgreSQL tests pass for upgrading from 083 to 084, repeated
migration runs, unchanged reserved money and photos, completed-review history,
renewed pending reviews, withdrawal approval checks and private-photo auditing.
A regression pins migration 083 to its already-applied checksum.

`go run ./cmd/migrate` then completed against the configured database on
2026-11-05 at 16:41:19: 83 SQL files checked, one pending migration applied.
The server can now pass the migration check on its next startup.
