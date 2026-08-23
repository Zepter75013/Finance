-- Distingue un compte administrateur (peut gérer les autres utilisateurs et
-- leurs comptes assignés) des utilisateurs normaux (ne peuvent éditer que
-- leur propre profil). Le premier utilisateur existant (le plus ancien)
-- devient admin lors de la migration, cohérent avec le compte bootstrap déjà
-- créé par EnsureAdmin sur une installation neuve.
ALTER TABLE users
  ADD COLUMN is_admin TINYINT(1) NOT NULL DEFAULT 0 AFTER accounts_restricted;

UPDATE users
  SET is_admin = 1
  WHERE id = (SELECT MIN(id) FROM (SELECT id FROM users) AS u);
