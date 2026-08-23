-- Fait vivre côté serveur les ajustements de budget par catégorie, qui ne
-- vivaient jusqu'ici que dans le localStorage du navigateur — nécessaire
-- pour que le planificateur de résumé quotidien par email puisse calculer
-- "ce compte dépasse son budget" sans dépendre d'un état client.
CREATE TABLE category_budgets (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  category_id BIGINT UNSIGNED NOT NULL,
  month_key VARCHAR(7) NOT NULL,
  amount DECIMAL(12,2) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uniq_category_month (category_id, month_key),
  CONSTRAINT fk_category_budgets_category FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
