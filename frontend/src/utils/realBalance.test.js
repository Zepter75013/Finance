import { describe, it, expect } from 'vitest'
import { findLatestLockedStatement, computeRealBalance } from './realBalance'

describe('findLatestLockedStatement', () => {
  it('renvoie null si aucun relevé n’est verrouillé', () => {
    const statements = [{ is_locked: false, period_end: '2026-08-01' }]
    expect(findLatestLockedStatement(statements)).toBeNull()
  })

  it('choisit le relevé verrouillé le plus récent, ignore les non verrouillés', () => {
    const statements = [
      { is_locked: true, period_end: '2026-07-31', end_balance: 1000 },
      { is_locked: true, period_end: '2026-08-15', end_balance: 1200 },
      { is_locked: false, period_end: '2026-08-20', end_balance: 9999 },
    ]

    expect(findLatestLockedStatement(statements)).toEqual(
      expect.objectContaining({ period_end: '2026-08-15', end_balance: 1200 })
    )
  })
})

describe('computeRealBalance', () => {
  it('renvoie null si aucun relevé n’est verrouillé', () => {
    expect(
      computeRealBalance({ statements: [{ is_locked: false }], purchases: [], incomes: [] })
    ).toBeNull()
  })

  it('ajoute les revenus non rapprochés et retire les achats non rapprochés', () => {
    const statements = [{ is_locked: true, period_end: '2026-08-15', end_balance: 1200 }]
    const purchases = [
      { amount: 50, statement_reference: '' },
      { amount: 30, statement_reference: 'REF-1' }, // déjà rapproché, exclu
    ]
    const incomes = [{ amount: 200, statement_reference: '' }]

    expect(computeRealBalance({ statements, purchases, incomes })).toBe(1200 + 200 - 50)
  })
})
