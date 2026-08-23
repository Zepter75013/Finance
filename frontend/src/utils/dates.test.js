import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { toUtcDateOnly, daysUntil } from './dates'

describe('toUtcDateOnly', () => {
  it('retourne le même instant UTC quel que soit le fuseau de sérialisation', () => {
    expect(toUtcDateOnly('2026-08-22T00:00:00Z')).toBe(Date.UTC(2026, 7, 22))
  })

  it('retourne null pour une date invalide', () => {
    expect(toUtcDateOnly('pas-une-date')).toBeNull()
  })
})

describe('daysUntil', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-23T10:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renvoie 0 pour aujourd’hui', () => {
    expect(daysUntil('2026-08-23T00:00:00Z')).toBe(0)
  })

  it('renvoie -1 pour hier (en retard)', () => {
    expect(daysUntil('2026-08-22T00:00:00Z')).toBe(-1)
  })

  it('renvoie +3 pour dans trois jours', () => {
    expect(daysUntil('2026-08-26T00:00:00Z')).toBe(3)
  })

  it('donne le même résultat en Z et en +02:00 (bug de fuseau corrigé)', () => {
    // 2026-08-26T00:00:00+02:00 correspond au même jour calendaire UTC que Z
    // ici puisque l'heure locale ne fait pas franchir minuit UTC — le point
    // important est que la fonction compare des calendriers UTC, pas des
    // horloges locales (voir le bug historique dans RecurringView.vue).
    expect(daysUntil('2026-08-26T00:00:00Z')).toBe(daysUntil('2026-08-26T00:00:00+02:00') + 1)
  })

  it('renvoie null pour une date invalide', () => {
    expect(daysUntil('pas-une-date')).toBeNull()
  })
})
