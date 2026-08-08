ALTER TABLE bank_statements
  ADD COLUMN pdf_filename VARCHAR(255) NULL AFTER is_locked;
