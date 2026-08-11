import { usePreferencesStore } from '../stores/preferences'

const NUMBER_LOCALES = {
  fr: 'fr-FR',
  us: 'en-US',
  ch: 'de-CH',
}

const CURRENCY_SYMBOLS = {
  EUR: '€',
  USD: '$',
  GBP: '£',
  CHF: 'CHF',
}

// sv-SE est l'astuce standard pour obtenir un format AAAA-MM-JJ via Intl —
// le suédois formate nativement les dates dans cet ordre.
const DATE_LOCALES = {
  dmy: 'fr-FR',
  mdy: 'en-US',
  iso: 'sv-SE',
}

export function formatCurrency(value, { decimals = 2 } = {}) {
  const prefs = usePreferencesStore()
  const locale = NUMBER_LOCALES[prefs.numberFormatStyle] || NUMBER_LOCALES.fr

  const number = new Intl.NumberFormat(locale, {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  }).format(Number(value || 0))

  const symbol = CURRENCY_SYMBOLS[prefs.currencyCode] || prefs.currencyCode

  return prefs.currencyPosition === 'before' ? `${symbol}${number}` : `${number} ${symbol}`
}

export function formatDate(dateStr) {
  if (!dateStr) return ''

  const date = dateStr instanceof Date ? dateStr : new Date(dateStr)
  if (Number.isNaN(date.getTime())) return String(dateStr)

  const prefs = usePreferencesStore()
  const locale = DATE_LOCALES[prefs.dateFormatStyle] || DATE_LOCALES.dmy

  return new Intl.DateTimeFormat(locale, { day: '2-digit', month: '2-digit', year: 'numeric' }).format(date)
}
