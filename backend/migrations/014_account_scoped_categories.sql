-- Les catégories deviennent propres à chaque compte (comme les achats,
-- revenus et relevés) : deux comptes peuvent avoir chacun une catégorie
-- "Alimentation" indépendante. L'historique existant est rattaché au compte
-- principal (id 1), qui détenait déjà tous les achats/revenus/relevés.
ALTER TABLE categories
  ADD COLUMN account_id BIGINT UNSIGNED NULL AFTER id;

UPDATE categories
  SET account_id = 1;

ALTER TABLE categories
  MODIFY COLUMN account_id BIGINT UNSIGNED NOT NULL,
  DROP INDEX uq_categories_name,
  ADD UNIQUE KEY uq_categories_account_name (account_id, name),
  ADD CONSTRAINT fk_categories_account FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE RESTRICT ON UPDATE CASCADE;
