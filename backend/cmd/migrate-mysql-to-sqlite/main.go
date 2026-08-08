// Outil ponctuel (mais rejouable) : copie l'intégralité des données de la
// base MySQL configurée (.env) vers le fichier SQLite local (data/finance.db
// par défaut, ou le chemin choisi dans Préférences) — pour "initialiser" la
// base SQLite avec les données réelles au lieu de démarrer vide.
//
// Rejouable sans risque : vide d'abord les tables SQLite (jamais MySQL, qui
// n'est lu qu'en lecture seule) puis réimporte tout, dans l'ordre des
// dépendances de clé étrangère. Les identifiants MySQL sont conservés tels
// quels côté SQLite pour que les relations (catégorie ↔ achat, compte ↔
// achat, etc.) restent identiques.
package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"finance/backend/internal/config"
	"finance/backend/internal/database"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system environment variables")
	}

	cfg := config.Load()

	mysqlDB, err := database.OpenMySQL(cfg)
	if err != nil {
		log.Fatalf("connexion MySQL: %v", err)
	}
	defer mysqlDB.Close()
	log.Println("MySQL OK")

	sqlitePath := cfg.SQLitePath
	if sqlitePath == "" {
		sqlitePath = "data/finance.db"
	}

	sqliteDB, err := database.OpenSQLite(sqlitePath)
	if err != nil {
		log.Fatalf("connexion SQLite (%s): %v", sqlitePath, err)
	}
	defer sqliteDB.Close()
	log.Printf("SQLite OK (%s)\n", sqlitePath)

	ctx := context.Background()

	if err := clearSQLiteTables(ctx, sqliteDB); err != nil {
		log.Fatalf("vidage des tables SQLite: %v", err)
	}

	counts := map[string]int{}

	counts["accounts"], err = copyAccounts(ctx, mysqlDB, sqliteDB)
	if err != nil {
		log.Fatalf("copie accounts: %v", err)
	}

	counts["users"], err = copyUsers(ctx, mysqlDB, sqliteDB)
	if err != nil {
		log.Fatalf("copie users: %v", err)
	}

	counts["categories"], err = copyCategories(ctx, mysqlDB, sqliteDB)
	if err != nil {
		log.Fatalf("copie categories: %v", err)
	}

	counts["sub_categories"], err = copySubCategories(ctx, mysqlDB, sqliteDB)
	if err != nil {
		log.Fatalf("copie sub_categories: %v", err)
	}

	counts["purchases"], err = copyPurchases(ctx, mysqlDB, sqliteDB)
	if err != nil {
		log.Fatalf("copie purchases: %v", err)
	}

	counts["incomes"], err = copyIncomes(ctx, mysqlDB, sqliteDB)
	if err != nil {
		log.Fatalf("copie incomes: %v", err)
	}

	counts["bank_statements"], err = copyBankStatements(ctx, mysqlDB, sqliteDB)
	if err != nil {
		log.Fatalf("copie bank_statements: %v", err)
	}

	counts["bank_statement_pdfs"], err = copyBankStatementPdfs(ctx, mysqlDB, sqliteDB)
	if err != nil {
		log.Fatalf("copie bank_statement_pdfs: %v", err)
	}

	log.Println("=== Import terminé ===")
	for _, table := range []string{
		"accounts", "users", "categories", "sub_categories",
		"purchases", "incomes", "bank_statements", "bank_statement_pdfs",
	} {
		log.Printf("  %-20s %d lignes", table, counts[table])
	}
}

// L'ordre inverse des dépendances de clé étrangère : on peut vider sans
// désactiver les contraintes.
func clearSQLiteTables(ctx context.Context, db *sql.DB) error {
	tables := []string{
		"bank_statement_pdfs",
		"bank_statements",
		"incomes",
		"purchases",
		"sub_categories",
		"categories",
		"sessions",
		"password_reset_codes",
		"users",
		"accounts",
	}

	for _, table := range tables {
		if _, err := db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return err
		}
	}

	return nil
}

func nullableString(v sql.NullString) any {
	if !v.Valid {
		return nil
	}
	return v.String
}

func nullableTime(v sql.NullTime) any {
	if !v.Valid {
		return nil
	}
	return v.Time
}

func copyAccounts(ctx context.Context, src, dst *sql.DB) (int, error) {
	rows, err := src.QueryContext(ctx, `SELECT id, name, created_at, updated_at FROM accounts`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id uint64
		var name string
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &name, &createdAt, &updatedAt); err != nil {
			return count, err
		}

		_, err := dst.ExecContext(ctx,
			`INSERT INTO accounts (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`,
			id, name, createdAt, updatedAt,
		)
		if err != nil {
			return count, err
		}
		count++
	}

	return count, rows.Err()
}

func copyUsers(ctx context.Context, src, dst *sql.DB) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, username, first_name, last_name, avatar_url, password_hash, created_at, updated_at
		FROM users
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id uint64
		var username, firstName, lastName, passwordHash string
		var avatarURL sql.NullString
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &username, &firstName, &lastName, &avatarURL, &passwordHash, &createdAt, &updatedAt); err != nil {
			return count, err
		}

		_, err := dst.ExecContext(ctx,
			`INSERT INTO users (id, username, first_name, last_name, avatar_url, password_hash, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			id, username, firstName, lastName, nullableString(avatarURL), passwordHash, createdAt, updatedAt,
		)
		if err != nil {
			return count, err
		}
		count++
	}

	return count, rows.Err()
}

func copyCategories(ctx context.Context, src, dst *sql.DB) (int, error) {
	rows, err := src.QueryContext(ctx, `SELECT id, account_id, name, type, created_at, updated_at FROM categories`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, accountID uint64
		var name, categoryType string
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &accountID, &name, &categoryType, &createdAt, &updatedAt); err != nil {
			return count, err
		}

		_, err := dst.ExecContext(ctx,
			`INSERT INTO categories (id, account_id, name, type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			id, accountID, name, categoryType, createdAt, updatedAt,
		)
		if err != nil {
			return count, err
		}
		count++
	}

	return count, rows.Err()
}

func copySubCategories(ctx context.Context, src, dst *sql.DB) (int, error) {
	rows, err := src.QueryContext(ctx, `SELECT id, category_id, name, created_at, updated_at FROM sub_categories`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, categoryID uint64
		var name string
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &categoryID, &name, &createdAt, &updatedAt); err != nil {
			return count, err
		}

		_, err := dst.ExecContext(ctx,
			`INSERT INTO sub_categories (id, category_id, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			id, categoryID, name, createdAt, updatedAt,
		)
		if err != nil {
			return count, err
		}
		count++
	}

	return count, rows.Err()
}

func copyPurchases(ctx context.Context, src, dst *sql.DB) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT
			id, merchant, payment_method, category_id, account_id, amount, purchase_date,
			note, reference, operation_label, additional_info, sub_category,
			operation_date, value_date, is_reconciled, statement_reference,
			created_at, updated_at
		FROM purchases
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, categoryID, accountID uint64
		var merchant, paymentMethod, reference, operationLabel, subCategory, statementReference string
		var amount float64
		var purchaseDate time.Time
		var note, additionalInfo sql.NullString
		var operationDate, valueDate sql.NullTime
		var isReconciled bool
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&id, &merchant, &paymentMethod, &categoryID, &accountID, &amount, &purchaseDate,
			&note, &reference, &operationLabel, &additionalInfo, &subCategory,
			&operationDate, &valueDate, &isReconciled, &statementReference,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return count, err
		}

		_, err = dst.ExecContext(ctx, `
			INSERT INTO purchases (
				id, merchant, payment_method, category_id, account_id, amount, purchase_date,
				note, reference, operation_label, additional_info, sub_category,
				operation_date, value_date, is_reconciled, statement_reference,
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			id, merchant, paymentMethod, categoryID, accountID, amount, purchaseDate,
			nullableString(note), reference, operationLabel, nullableString(additionalInfo), subCategory,
			nullableTime(operationDate), nullableTime(valueDate), isReconciled, statementReference,
			createdAt, updatedAt,
		)
		if err != nil {
			return count, err
		}
		count++
	}

	return count, rows.Err()
}

func copyIncomes(ctx context.Context, src, dst *sql.DB) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT
			id, account_id, source, amount, income_date,
			note, reference, operation_label, additional_info, operation_type,
			category, sub_category, operation_date, value_date, is_reconciled,
			statement_reference, created_at, updated_at
		FROM incomes
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, accountID uint64
		var source, reference, operationLabel, operationType, category, subCategory, statementReference string
		var amount float64
		var incomeDate time.Time
		var note, additionalInfo sql.NullString
		var operationDate, valueDate sql.NullTime
		var isReconciled bool
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(
			&id, &accountID, &source, &amount, &incomeDate,
			&note, &reference, &operationLabel, &additionalInfo, &operationType,
			&category, &subCategory, &operationDate, &valueDate, &isReconciled,
			&statementReference, &createdAt, &updatedAt,
		)
		if err != nil {
			return count, err
		}

		_, err = dst.ExecContext(ctx, `
			INSERT INTO incomes (
				id, account_id, source, amount, income_date,
				note, reference, operation_label, additional_info, operation_type,
				category, sub_category, operation_date, value_date, is_reconciled,
				statement_reference, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			id, accountID, source, amount, incomeDate,
			nullableString(note), reference, operationLabel, nullableString(additionalInfo), operationType,
			category, subCategory, nullableTime(operationDate), nullableTime(valueDate), isReconciled,
			statementReference, nullableTime(createdAt), nullableTime(updatedAt),
		)
		if err != nil {
			return count, err
		}
		count++
	}

	return count, rows.Err()
}

func copyBankStatements(ctx context.Context, src, dst *sql.DB) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT
			id, account_id, reference, statement_date, period_start, period_end,
			start_balance, end_balance, is_locked, created_at, updated_at
		FROM bank_statements
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, accountID uint64
		var reference string
		var statementDate, periodStart, periodEnd sql.NullTime
		var startBalance, endBalance float64
		var isLocked bool
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&id, &accountID, &reference, &statementDate, &periodStart, &periodEnd,
			&startBalance, &endBalance, &isLocked, &createdAt, &updatedAt,
		)
		if err != nil {
			return count, err
		}

		_, err = dst.ExecContext(ctx, `
			INSERT INTO bank_statements (
				id, account_id, reference, statement_date, period_start, period_end,
				start_balance, end_balance, is_locked, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			id, accountID, reference, nullableTime(statementDate), nullableTime(periodStart), nullableTime(periodEnd),
			startBalance, endBalance, isLocked, createdAt, updatedAt,
		)
		if err != nil {
			return count, err
		}
		count++
	}

	return count, rows.Err()
}

func copyBankStatementPdfs(ctx context.Context, src, dst *sql.DB) (int, error) {
	rows, err := src.QueryContext(ctx, `
		SELECT id, statement_id, filename, original_filename, created_at
		FROM bank_statement_pdfs
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, statementID uint64
		var filename, originalFilename string
		var createdAt time.Time

		if err := rows.Scan(&id, &statementID, &filename, &originalFilename, &createdAt); err != nil {
			return count, err
		}

		_, err := dst.ExecContext(ctx,
			`INSERT INTO bank_statement_pdfs (id, statement_id, filename, original_filename, created_at) VALUES (?, ?, ?, ?, ?)`,
			id, statementID, filename, originalFilename, createdAt,
		)
		if err != nil {
			return count, err
		}
		count++
	}

	return count, rows.Err()
}
