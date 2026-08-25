import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

const STORAGE_KEY = 'financeDisplayPreferences'

// Ces valeurs par défaut reproduisent exactement le rendu actuel de l'app
// (Intl.NumberFormat('fr-FR', { style: 'currency', currency: 'EUR' })) —
// aucun changement visuel tant que l'utilisateur n'a rien modifié.
const DEFAULTS = {
  currencyCode: 'EUR',
  currencyPosition: 'after',
  numberFormatStyle: 'fr',
  dateFormatStyle: 'dmy',
}

const VALID_CURRENCIES = ['EUR', 'USD', 'GBP', 'CHF']
const VALID_POSITIONS = ['before', 'after']
const VALID_NUMBER_STYLES = ['fr', 'us', 'ch']
const VALID_DATE_STYLES = ['dmy', 'mdy', 'iso']

function loadStored() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...DEFAULTS }

    const parsed = JSON.parse(raw)
    return {
      currencyCode: VALID_CURRENCIES.includes(parsed.currencyCode) ? parsed.currencyCode : DEFAULTS.currencyCode,
      currencyPosition: VALID_POSITIONS.includes(parsed.currencyPosition)
        ? parsed.currencyPosition
        : DEFAULTS.currencyPosition,
      numberFormatStyle: VALID_NUMBER_STYLES.includes(parsed.numberFormatStyle)
        ? parsed.numberFormatStyle
        : DEFAULTS.numberFormatStyle,
      dateFormatStyle: VALID_DATE_STYLES.includes(parsed.dateFormatStyle)
        ? parsed.dateFormatStyle
        : DEFAULTS.dateFormatStyle,
    }
  } catch {
    return { ...DEFAULTS }
  }
}

export const usePreferencesStore = defineStore('preferences', () => {
  const stored = loadStored()

  const currencyCode = ref(stored.currencyCode)
  const currencyPosition = ref(stored.currencyPosition)
  const numberFormatStyle = ref(stored.numberFormatStyle)
  const dateFormatStyle = ref(stored.dateFormatStyle)

  watch([currencyCode, currencyPosition, numberFormatStyle, dateFormatStyle], () => {
    try {
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({
          currencyCode: currencyCode.value,
          currencyPosition: currencyPosition.value,
          numberFormatStyle: numberFormatStyle.value,
          dateFormatStyle: dateFormatStyle.value,
        })
      )
    } catch {
      // Stockage indisponible (navigation privée, quota…) — les réglages
      // restent actifs pour la session en cours mais ne survivront pas à un
      // rechargement.
    }
  })

  function setCurrencyCode(value) {
    if (!VALID_CURRENCIES.includes(value)) return
    currencyCode.value = value
  }

  function setCurrencyPosition(value) {
    if (!VALID_POSITIONS.includes(value)) return
    currencyPosition.value = value
  }

  function setNumberFormatStyle(value) {
    if (!VALID_NUMBER_STYLES.includes(value)) return
    numberFormatStyle.value = value
  }

  function setDateFormatStyle(value) {
    if (!VALID_DATE_STYLES.includes(value)) return
    dateFormatStyle.value = value
  }

  return {
    currencyCode,
    currencyPosition,
    numberFormatStyle,
    dateFormatStyle,
    setCurrencyCode,
    setCurrencyPosition,
    setNumberFormatStyle,
    setDateFormatStyle,
  }
})
