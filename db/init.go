package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init(dbPath string) {
	var err error
	DB, err = sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		log.Fatal("failed to open database:", err)
	}

	DB.SetMaxOpenConns(1)

	createTables()
}

func createTables() {
	schema := `
	CREATE TABLE IF NOT EXISTS members (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		nickname TEXT NOT NULL UNIQUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT DEFAULT '',
		currency TEXT NOT NULL DEFAULT 'CNY',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS group_members (
		group_id INTEGER NOT NULL,
		member_id INTEGER NOT NULL,
		joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (group_id, member_id),
		FOREIGN KEY (group_id) REFERENCES groups(id),
		FOREIGN KEY (member_id) REFERENCES members(id)
	);

	CREATE TABLE IF NOT EXISTS expenses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id INTEGER NOT NULL,
		payer_id INTEGER NOT NULL,
		amount REAL NOT NULL,
		currency TEXT NOT NULL DEFAULT 'CNY',
		exchange_rate REAL NOT NULL DEFAULT 1.0,
		amount_in_default REAL NOT NULL,
		description TEXT DEFAULT '',
		split_type TEXT NOT NULL DEFAULT 'equal',
		expense_date DATE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (group_id) REFERENCES groups(id),
		FOREIGN KEY (payer_id) REFERENCES members(id)
	);

	CREATE TABLE IF NOT EXISTS expense_splits (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		expense_id INTEGER NOT NULL,
		member_id INTEGER NOT NULL,
		amount REAL NOT NULL,
		percentage REAL DEFAULT 0,
		FOREIGN KEY (expense_id) REFERENCES expenses(id),
		FOREIGN KEY (member_id) REFERENCES members(id)
	);

	CREATE TABLE IF NOT EXISTS settlements (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id INTEGER NOT NULL,
		payer_id INTEGER NOT NULL,
		payee_id INTEGER NOT NULL,
		amount REAL NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (group_id) REFERENCES groups(id),
		FOREIGN KEY (payer_id) REFERENCES members(id),
		FOREIGN KEY (payee_id) REFERENCES members(id)
	);

	CREATE INDEX IF NOT EXISTS idx_expenses_group ON expenses(group_id);
	CREATE INDEX IF NOT EXISTS idx_splits_expense ON expense_splits(expense_id);
	CREATE INDEX IF NOT EXISTS idx_settlements_group ON settlements(group_id);
	`

	_, err := DB.Exec(schema)
	if err != nil {
		log.Fatal("failed to create tables:", err)
	}
}
