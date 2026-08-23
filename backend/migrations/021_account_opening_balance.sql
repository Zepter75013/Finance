-- Solde initial déclaré par compte — sert d'ancrage pour computeRealBalance
-- quand aucun relevé n'est verrouillé (compte type Livret, qui n'en a
-- structurellement jamais), pour éviter d'avoir à ressaisir tout
-- l'historique des mouvements juste pour obtenir un solde correct.
ALTER TABLE accounts
  ADD COLUMN opening_balance_amount DECIMAL(12,2) NULL,
  ADD COLUMN opening_balance_date DATE NULL;
