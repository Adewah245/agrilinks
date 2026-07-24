package database

import(
	"database/sql"
	"os"
	"log"
    _ "github.com/jackc/pgx/v5/stdlib"
)
var DB *sql.DB
