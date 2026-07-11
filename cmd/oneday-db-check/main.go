package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/crimsab/oneday/internal/storage"
)

func main() {
	dbPath := flag.String("db", "", "path to a OneDay SQLite database")
	flag.Parse()
	if *dbPath == "" { log.Fatal("-db is required") }
	db, err := storage.Open(*dbPath)
	if err != nil { log.Fatalf("open/migrate: %v", err) }
	defer db.Close()
	var version int
	if err := db.Conn().QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_version`).Scan(&version); err != nil { log.Fatal(err) }
	var integrity string
	if err := db.Conn().QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil { log.Fatal(err) }
	var foreignKeyViolations int
	rows, err := db.Conn().Query(`PRAGMA foreign_key_check`)
	if err != nil { log.Fatal(err) }
	for rows.Next() { foreignKeyViolations++ }
	if err := rows.Close(); err != nil { log.Fatal(err) }
	if integrity != "ok" || foreignKeyViolations != 0 { log.Fatalf("integrity=%s foreign_key_violations=%d", integrity, foreignKeyViolations) }
	fmt.Printf("schema_version=%d integrity=%s foreign_key_violations=%d\n", version, integrity, foreignKeyViolations)
}
