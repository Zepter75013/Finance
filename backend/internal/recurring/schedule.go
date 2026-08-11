package recurring

import "time"

// dateForMonth construit une date pour dayOfMonth dans le mois donné, en
// ramenant au dernier jour réel de ce mois si dayOfMonth le dépasse (ex: 31
// dans un mois de 30 jours, ou en février).
func dateForMonth(year int, month time.Month, dayOfMonth int) time.Time {
	firstOfNextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
	lastDayOfMonth := firstOfNextMonth.AddDate(0, 0, -1).Day()

	day := dayOfMonth
	if day > lastDayOfMonth {
		day = lastDayOfMonth
	}

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// advanceOneMonth calcule la prochaine occurrence après exécution, en
// avançant d'un mois depuis la date PLANIFIÉE (pas l'heure d'exécution
// réelle) — évite toute dérive si le conteneur était arrêté au moment prévu.
func advanceOneMonth(scheduledDate time.Time, dayOfMonth int) time.Time {
	return dateForMonth(scheduledDate.Year(), scheduledDate.Month()+1, dayOfMonth)
}
