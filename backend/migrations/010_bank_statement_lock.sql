ALTER TABLE bank_statements
  ADD COLUMN is_locked TINYINT(1) NOT NULL DEFAULT 0 AFTER end_balance;
