import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { formatCurrency, formatDate } from './format'
import { usePreferencesStore } from '../stores/preferences'

describe('formatCurrency / formatDate', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('utilise les préférences par défaut (EUR après le montant, séparateurs fr)', () => {
    const result = formatCurrency(1234.5)

    expect(result).toContain('€')
    expect(result).toMatch(/1.234,50 €$/)
  })

  it('respecte un changement de préférence (USD avant le montant)', () => {
    const prefs = usePreferencesStore()
    prefs.setCurrencyCode('USD')
    prefs.setCurrencyPosition('before')

    const result = formatCurrency(1234.5)

    expect(result).toMatch(/^\$1.234,50/)
  })

  it('respecte le nombre de décimales demandé', () => {
    expect(formatCurrency(1234, { decimals: 0 })).toMatch(/^1.234 €$/)
  })

  it('formatDate respecte le format par défaut JJ/MM/AAAA', () => {
    expect(formatDate('2026-08-11')).toBe('11/08/2026')
  })

  it('formatDate respecte un changement de préférence (AAAA-MM-JJ)', () => {
    const prefs = usePreferencesStore()
    prefs.setDateFormatStyle('iso')

    expect(formatDate('2026-08-11')).toBe('2026-08-11')
  })

  it('formatDate renvoie une chaîne vide pour une date absente', () => {
    expect(formatDate('')).toBe('')
  })
})
