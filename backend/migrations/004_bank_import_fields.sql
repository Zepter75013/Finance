ALTER TABLE purchases
  ADD COLUMN reference VARCHAR(64) NOT NULL DEFAULT '' AFTER note,
  ADD COLUMN operation_label VARCHAR(255) NOT NULL DEFAULT '' AFTER reference,
  ADD COLUMN additional_info TEXT NULL AFTER operation_label,
  ADD COLUMN sub_category VARCHAR(120) NOT NULL DEFAULT '' AFTER additional_info,
  ADD COLUMN operation_date DATE NULL AFTER sub_category,
  ADD COLUMN value_date DATE NULL AFTER operation_date,
  ADD COLUMN is_reconciled TINYINT(1) NOT NULL DEFAULT 0 AFTER value_date;

ALTER TABLE incomes
  ADD COLUMN reference VARCHAR(64) NOT NULL DEFAULT '' AFTER note,
  ADD COLUMN operation_label VARCHAR(255) NOT NULL DEFAULT '' AFTER reference,
  ADD COLUMN additional_info TEXT NULL AFTER operation_label,
  ADD COLUMN operation_type VARCHAR(120) NOT NULL DEFAULT '' AFTER additional_info,
  ADD COLUMN category VARCHAR(120) NOT NULL DEFAULT '' AFTER operation_type,
  ADD COLUMN sub_category VARCHAR(120) NOT NULL DEFAULT '' AFTER category,
  ADD COLUMN operation_date DATE NULL AFTER sub_category,
  ADD COLUMN value_date DATE NULL AFTER operation_date,
  ADD COLUMN is_reconciled TINYINT(1) NOT NULL DEFAULT 0 AFTER value_date;
