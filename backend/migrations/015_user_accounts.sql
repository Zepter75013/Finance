-- Assignation d'un utilisateur à un ou plusieurs comptes. "accounts_restricted"
-- distingue "jamais configuré" (accès à tous les comptes, valeur par défaut
-- rétrocompatible avec les utilisateurs existants) de "explicitement restreint"
-- (même à zéro compte) — une liste de comptes assignés vide ne suffirait pas à
-- elle seule à faire cette distinction.
ALTER TABLE users
  ADD COLUMN accounts_restricted TINYINT(1) NOT NULL DEFAULT 0 AFTER password_hash;

CREATE TABLE user_accounts (
  user_id BIGINT UNSIGNED NOT NULL,
  account_id BIGINT UNSIGNED NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, account_id),
  KEY idx_user_accounts_account_id (account_id),
  CONSTRAINT fk_user_accounts_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_user_accounts_account FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
