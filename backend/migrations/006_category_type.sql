ALTER TABLE categories
  ADD COLUMN type ENUM('achat', 'revenu') NOT NULL DEFAULT 'achat' AFTER name;

INSERT INTO categories (name, type) VALUES
  ('Salaire', 'revenu'),
  ('Revenus complementaires', 'revenu'),
  ('Remboursements', 'revenu'),
  ('Aides et allocations', 'revenu'),
  ('Revenus financiers', 'revenu'),
  ('Pension et retraite', 'revenu'),
  ('Autres revenus', 'revenu')
ON DUPLICATE KEY UPDATE type = VALUES(type);

INSERT INTO sub_categories (category_id, name)
SELECT c.id, s.name
FROM categories c
JOIN (
  SELECT 'Salaire' AS cat, 'Salaire net' AS name
  UNION ALL SELECT 'Salaire', 'Primes / bonus'
  UNION ALL SELECT 'Revenus complementaires', 'Freelance / missions'
  UNION ALL SELECT 'Revenus complementaires', 'Ventes occasionnelles'
  UNION ALL SELECT 'Remboursements', 'Remboursements mutuelle / secu'
  UNION ALL SELECT 'Remboursements', 'Remboursements frais professionnels'
  UNION ALL SELECT 'Remboursements', 'Remboursements divers'
  UNION ALL SELECT 'Aides et allocations', 'Allocations familiales'
  UNION ALL SELECT 'Aides et allocations', 'Aides sociales (CAF, etc.)'
  UNION ALL SELECT 'Aides et allocations', 'Bourses d''etudes'
  UNION ALL SELECT 'Revenus financiers', 'Interets et dividendes'
  UNION ALL SELECT 'Revenus financiers', 'Plus-values (bourse, PEA)'
  UNION ALL SELECT 'Revenus financiers', 'Revenus locatifs'
  UNION ALL SELECT 'Pension et retraite', 'Pension de retraite'
  UNION ALL SELECT 'Pension et retraite', 'Pension alimentaire recue'
  UNION ALL SELECT 'Autres revenus', 'Cadeaux recus'
  UNION ALL SELECT 'Autres revenus', 'Autres revenus divers'
) s ON s.cat = c.name
ON DUPLICATE KEY UPDATE name = VALUES(name);
