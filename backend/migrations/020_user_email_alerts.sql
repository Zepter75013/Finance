-- Préférence personnelle permettant à chaque utilisateur de recevoir (ou
-- non) le résumé quotidien par email (budgets dépassés, échéances de
-- récurrence). Activé par défaut pour les comptes existants.
ALTER TABLE users
  ADD COLUMN email_alerts_enabled TINYINT(1) NOT NULL DEFAULT 1 AFTER is_admin;
