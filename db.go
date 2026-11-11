package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type RequestLog struct {
	ID        int64
	IPAddress string
	Timestamp time.Time
	Path      string
	Method    string
}

func openDB(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite db: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS request_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip_address TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			path TEXT NOT NULL,
			method TEXT NOT NULL
		);`); err != nil {
		db.Close()
		return nil, fmt.Errorf("create request_logs table: %w", err)
	}

	return db, nil
}

func insertRequestLog(db *sql.DB, ipAddress, path, method string) {
	if db == nil {
		return
	}
	_, err := db.Exec(
		`INSERT INTO request_logs (ip_address, timestamp, path, method) VALUES (?, ?, ?, ?)`,
		ipAddress,
		time.Now().UTC().Format(time.RFC3339),
		path,
		method,
	)
	if err != nil {
		log.Printf("insert request log: %v", err)
	}
}

func countRequestLogs(db *sql.DB) int {
	if db == nil {
		return 0
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM request_logs`).Scan(&count); err != nil {
		log.Printf("count request logs: %v", err)
		return 0
	}
	return count
}
