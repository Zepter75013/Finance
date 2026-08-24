-- Un virement créé en convertissant une ligne d'achat/revenu existante
-- (écran Pointage) garde une copie de cette ligne d'origine, pour pouvoir
-- annuler le virement et recréer exactement l'achat/revenu initial. Un
-- virement créé directement (sans ligne d'origine, ex: import CSV) laisse
-- ces deux colonnes NULL — il n'y a alors rien à restaurer, seulement à
-- supprimer.
ALTER TABLE transfers
  ADD COLUMN origin_type VARCHAR(10) NULL,
  ADD COLUMN origin_payload TEXT NULL;
