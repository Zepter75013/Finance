CREATE TABLE IF NOT EXISTS bank_statement_pdfs (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  statement_id BIGINT UNSIGNED NOT NULL,
  filename VARCHAR(255) NOT NULL,
  original_filename VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_bank_statement_pdfs_statement
    FOREIGN KEY (statement_id) REFERENCES bank_statements(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Reprend les PDF déjà attachés (un seul par relevé jusqu'ici) dans la
-- nouvelle table, sans toucher aux fichiers déjà sur disque.
INSERT INTO bank_statement_pdfs (statement_id, filename, original_filename)
SELECT id, pdf_filename, pdf_filename FROM bank_statements WHERE pdf_filename IS NOT NULL;

ALTER TABLE bank_statements DROP COLUMN pdf_filename;
