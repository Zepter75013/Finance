CREATE TABLE accounts (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(100) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uniq_accounts_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Compte par défaut : tout l'historique existant (achats, revenus, relevés)
-- lui est rattaché automatiquement, pour ne rien casser rétroactivement.
INSERT INTO accounts (name) VALUES ('Compte principal');

ALTER TABLE purchases
  ADD COLUMN account_id BIGINT UNSIGNED NULL AFTER category_id;

UPDATE purchases
  SET account_id = (SELECT id FROM accounts WHERE name = 'Compte principal' LIMIT 1);

ALTER TABLE purchases
  MODIFY COLUMN account_id BIGINT UNSIGNED NOT NULL,
  ADD KEY idx_purchases_account_id (account_id),
  ADD CONSTRAINT fk_purchases_account FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE incomes
  ADD COLUMN account_id BIGINT UNSIGNED NULL AFTER id;

UPDATE incomes
  SET account_id = (SELECT id FROM accounts WHERE name = 'Compte principal' LIMIT 1);

ALTER TABLE incomes
  MODIFY COLUMN account_id BIGINT UNSIGNED NOT NULL,
  ADD KEY idx_incomes_account_id (account_id),
  ADD CONSTRAINT fk_incomes_account FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE bank_statements
  ADD COLUMN account_id BIGINT UNSIGNED NULL AFTER id;

UPDATE bank_statements
  SET account_id = (SELECT id FROM accounts WHERE name = 'Compte principal' LIMIT 1);

-- Le numéro de relevé n'est unique que par compte désormais (deux comptes
-- peuvent chacun avoir un relevé "1").
ALTER TABLE bank_statements
  MODIFY COLUMN account_id BIGINT UNSIGNED NOT NULL,
  DROP INDEX uniq_bank_statements_reference,
  ADD UNIQUE KEY uniq_bank_statements_account_reference (account_id, reference),
  ADD CONSTRAINT fk_bank_statements_account FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE RESTRICT ON UPDATE CASCADE;
