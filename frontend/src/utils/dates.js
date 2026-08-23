// Compare des calendriers UTC plutôt que des horloges locales : le backend
// sérialise next_run_date en minuit UTC, donc comparer à un "minuit local"
// décale le résultat d'1 à 2h en France selon l'heure d'été/hiver.
export function toUtcDateOnly(value) {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return null

  return Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate())
}

export function daysUntil(value) {
  const target = toUtcDateOnly(value)
  if (target === null) return null

  const today = toUtcDateOnly(new Date())
  return Math.round((target - today) / 86_400_000)
}
