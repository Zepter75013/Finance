package database

// Schéma consolidé pour la base SQLite locale — traduit à la main du schéma
// MySQL actuel (une seule fois, puisque la base SQLite est indépendante et
// démarre toujours vide : pas besoin de rejouer l'historique des migrations
// MySQL). Idempotent (CREATE TABLE/INDEX IF NOT EXISTS) : peut tourner sans
// risque à chaque démarrage. Une instruction par élément — database/sql
// n'exécute qu'une seule instruction par appel Exec.
var sqliteSchemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (name)
	)`,
	`CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id INTEGER NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
		name TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'achat' CHECK (type IN ('achat', 'revenu')),
		color TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (account_id, name)
	)`,
	`CREATE TABLE IF NOT EXISTS sub_categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		category_id INTEGER NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (category_id, name)
	)`,
	`CREATE TABLE IF NOT EXISTS purchases (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		merchant TEXT NOT NULL,
		payment_method TEXT NOT NULL,
		category_id INTEGER NOT NULL REFERENCES categories (id) ON DELETE RESTRICT,
		account_id INTEGER NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
		amount DECIMAL(10, 2) NOT NULL,
		purchase_date DATE NOT NULL,
		note TEXT,
		reference TEXT NOT NULL DEFAULT '',
		operation_label TEXT NOT NULL DEFAULT '',
		additional_info TEXT,
		sub_category TEXT NOT NULL DEFAULT '',
		operation_date DATE,
		value_date DATE,
		is_reconciled INTEGER NOT NULL DEFAULT 0,
		statement_reference TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_purchases_purchase_date ON purchases (purchase_date)`,
	`CREATE INDEX IF NOT EXISTS idx_purchases_category_id ON purchases (category_id)`,
	`CREATE INDEX IF NOT EXISTS idx_purchases_account_id ON purchases (account_id)`,
	`CREATE TABLE IF NOT EXISTS incomes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id INTEGER NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
		source TEXT NOT NULL,
		amount DECIMAL(10, 2) NOT NULL,
		income_date DATE NOT NULL,
		note TEXT,
		reference TEXT NOT NULL DEFAULT '',
		operation_label TEXT NOT NULL DEFAULT '',
		additional_info TEXT,
		operation_type TEXT NOT NULL DEFAULT '',
		category TEXT NOT NULL DEFAULT '',
		sub_category TEXT NOT NULL DEFAULT '',
		operation_date DATE,
		value_date DATE,
		is_reconciled INTEGER NOT NULL DEFAULT 0,
		statement_reference TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_incomes_account_id ON incomes (account_id)`,
	`CREATE TABLE IF NOT EXISTS bank_statements (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id INTEGER NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
		reference TEXT NOT NULL,
		statement_date DATE,
		period_start DATE,
		period_end DATE,
		start_balance DECIMAL(12, 2) NOT NULL DEFAULT 0,
		end_balance DECIMAL(12, 2) NOT NULL DEFAULT 0,
		is_locked INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (account_id, reference)
	)`,
	`CREATE TABLE IF NOT EXISTS bank_statement_pdfs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		statement_id INTEGER NOT NULL REFERENCES bank_statements (id) ON DELETE CASCADE,
		filename TEXT NOT NULL,
		original_filename TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_bank_statement_pdfs_statement ON bank_statement_pdfs (statement_id)`,
	`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		first_name TEXT NOT NULL DEFAULT '',
		last_name TEXT NOT NULL DEFAULT '',
		avatar_url TEXT,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS sessions (
		token TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions (user_id)`,
	`CREATE TABLE IF NOT EXISTS password_reset_codes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
		code TEXT NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_reset_codes_user ON password_reset_codes (user_id)`,
	// Un compte par défaut est indispensable : achats/revenus/catégories
	// exigent tous un account_id, l'app serait inutilisable sans au moins un
	// compte existant dès le premier démarrage sur une base vierge.
	`INSERT INTO accounts (name)
		SELECT 'Compte principal'
		WHERE NOT EXISTS (SELECT 1 FROM accounts)`,
}
