package notify

import "time"

// daysUntil renvoie le nombre de jours (négatif si en retard) avant t.
//
// Piège évité ici : la connexion MySQL utilise loc=Local
// (backend/internal/database/mysql.go), donc un time.Time scanné depuis une
// colonne DATE (comme recurring_transactions.next_run_date) porte déjà le
// bon jour calendaire dans SES PROPRES champs Year/Month/Day — le pilote ne
// fait aucune conversion, il tague juste la date littérale de la ligne SQL
// avec time.Local. Convertir explicitement en UTC via .UTC() AVANT de lire
// ces champs décalerait le jour d'un cran si le fuseau local du serveur
// n'est pas UTC (exactement le bug déjà corrigé côté frontend dans
// utils/dates.js). La bonne approche est donc symétrique : lire Year/Month/
// Day directement sur t ET sur time.Now(), tous deux implicitement dans
// time.Local, sans jamais appeler .UTC() sur l'un sans l'autre.
func daysUntil(t time.Time) int {
	target := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	return int(target.Sub(today).Hours() / 24)
}
