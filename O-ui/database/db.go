package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	db   *sql.DB
	once sync.Once
)

func Init(dbPath string) error {
	var initErr error
	once.Do(func() {
		dir := filepath.Dir(dbPath)
		os.MkdirAll(dir, 0755)

		var err error
		db, err = sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
		if err != nil {
			initErr = err
			return
		}
		db.SetMaxOpenConns(1)
		initErr = migrate()
	})
	return initErr
}

func GetDB() *sql.DB { return db }

func Close() {
	if db != nil {
		db.Close()
	}
}

func migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			role TEXT DEFAULT 'admin',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_login DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS inbounds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tag TEXT UNIQUE NOT NULL,
			protocol TEXT NOT NULL,
			port INTEGER NOT NULL,
			listen TEXT DEFAULT '0.0.0.0',
			settings TEXT DEFAULT '{}',
			stream_settings TEXT DEFAULT '{}',
			enabled INTEGER DEFAULT 1,
			remark TEXT DEFAULT '',
			traffic_up INTEGER DEFAULT 0,
			traffic_down INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS outbounds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tag TEXT UNIQUE NOT NULL,
			protocol TEXT NOT NULL,
			settings TEXT DEFAULT '{}',
			stream_settings TEXT DEFAULT '{}',
			enabled INTEGER DEFAULT 1,
			remark TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS nodes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			address TEXT NOT NULL,
			port INTEGER NOT NULL,
			protocol TEXT NOT NULL,
			uuid TEXT DEFAULT '',
			password TEXT DEFAULT '',
			method TEXT DEFAULT '',
			settings TEXT DEFAULT '{}',
			group_name TEXT DEFAULT 'default',
			enabled INTEGER DEFAULT 1,
			traffic_up INTEGER DEFAULT 0,
			traffic_down INTEGER DEFAULT 0,
			last_check DATETIME,
			latency INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS traffic_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			inbound_tag TEXT,
			node_id INTEGER,
			upload INTEGER DEFAULT 0,
			download INTEGER DEFAULT 0,
			recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	// Insert default admin if not exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if count == 0 {
		// Default password: oiwest (bcrypt hashed)
		db.Exec("INSERT INTO users (username, password, role) VALUES ('admin', '$2a$10$lDtHoMQ2YRE9GH2VZ.ipG.zVnYNPnOtL/u4tK5LYAgnXSF78GAjjW', 'admin')")
	}
	// Insert default settings
	defaults := map[string]string{
		"panel_title":    "O-ui Panel",
		"panel_port":     "54321",
		"core_path":      "./oiwest-core",
		"log_level":      "info",
		"sub_enabled":    "true",
		"sub_path":       "/sub",
		"sub_update":     "24",
	}
	for k, v := range defaults {
		db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES (?, ?)", k, v)
	}
	return nil
}
