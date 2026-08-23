-- Virement entre deux comptes de l'utilisateur (ex: compte courant -> Livret)
-- — débite from_account_id, crédite to_account_id. Volontairement une table
-- séparée de purchases/incomes : ni budgets ni résumé quotidien par email ne
-- doivent compter un virement comme une vraie dépense/rentrée d'argent.
-- Deux paires de champs de pointage indépendantes (from_*/to_*) car chaque
-- compte a son propre historique de relevés bancaires, sans lien entre eux.
CREATE TABLE transfers (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  from_account_id BIGINT UNSIGNED NOT NULL,
  to_account_id BIGINT UNSIGNED NOT NULL,
  amount DECIMAL(12,2) NOT NULL,
  transfer_date DATE NOT NULL,
  note VARCHAR(255) NOT NULL DEFAULT '',
  from_is_reconciled TINYINT(1) NOT NULL DEFAULT 0,
  from_statement_reference VARCHAR(64) NOT NULL DEFAULT '',
  to_is_reconciled TINYINT(1) NOT NULL DEFAULT 0,
  to_statement_reference VARCHAR(64) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_transfers_from_account (from_account_id),
  KEY idx_transfers_to_account (to_account_id),
  CONSTRAINT fk_transfers_from_account FOREIGN KEY (from_account_id) REFERENCES accounts (id) ON DELETE RESTRICT,
  CONSTRAINT fk_transfers_to_account FOREIGN KEY (to_account_id) REFERENCES accounts (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
