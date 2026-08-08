ALTER TABLE purchases
  ADD COLUMN statement_reference VARCHAR(64) NOT NULL DEFAULT '' AFTER is_reconciled;

ALTER TABLE incomes
  ADD COLUMN statement_reference VARCHAR(64) NOT NULL DEFAULT '' AFTER is_reconciled;
