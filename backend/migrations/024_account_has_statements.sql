-- Certains comptes (ex: Livret de Développement Durable et Solidaire) n'ont
-- structurellement jamais de relevé bancaire mensuel — seulement des
-- virements depuis/vers un autre compte. Le pointage (rapprochement contre
-- un relevé) n'a alors aucun sens. Ce booléen, vrai par défaut (aucun
-- changement de comportement pour les comptes existants), permet de le
-- désactiver pour ce type de compte.
ALTER TABLE accounts
  ADD COLUMN has_statements TINYINT(1) NOT NULL DEFAULT 1;
