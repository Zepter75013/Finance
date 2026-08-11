-- Gabarits de transactions récurrentes (loyer, abonnements, salaire...).
-- Une seule table pour achats et revenus (discriminant "type") : le
-- planificateur a besoin d'une requête unique "qu'est-ce qui est dû
-- aujourd'hui" indépendamment du type, et le CRUD des gabarits est identique
-- pour les deux.
CREATE TABLE recurring_transactions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  account_id BIGINT UNSIGNED NOT NULL,
  type ENUM('achat', 'revenu') NOT NULL,
  merchant VARCHAR(255) NULL,
  source VARCHAR(255) NULL,
  category_id BIGINT UNSIGNED NULL,
  payment_method VARCHAR(100) NULL,
  operation_type VARCHAR(100) NULL,
  category VARCHAR(100) NULL,
  sub_category VARCHAR(100) NOT NULL DEFAULT '',
  amount DECIMAL(10, 2) NOT NULL,
  day_of_month TINYINT UNSIGNED NOT NULL,
  note TEXT NULL,
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  next_run_date DATE NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_recurring_account_id (account_id),
  KEY idx_recurring_next_run_date (next_run_date),
  CONSTRAINT fk_recurring_account FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_recurring_category FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT chk_recurring_day_of_month CHECK (day_of_month BETWEEN 1 AND 31)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
