import { describe, it, expect } from 'vitest'
import { findLatestLockedStatement, computeRealBalance, signTransferLeg } from './realBalance'

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
  it('sans relevé verrouillé ni solde initial, retombe sur les mouvements non rapprochés (compte type Livret)', () => {
    const purchases = [{ amount: 50, statement_reference: '' }]
    const incomes = [
      { amount: 200, statement_reference: '' },
      { amount: 999, statement_reference: 'REF-ANCIEN' }, // déjà rapproché à un relevé disparu, exclu
    ]

    expect(
      computeRealBalance({ statements: [{ is_locked: false }], purchases, incomes })
    ).toEqual({ balance: 200 - 50, source: 'ledger' })
  })

  it('ajoute les revenus non rapprochés et retire les achats non rapprochés, ancré au relevé verrouillé', () => {
    const statements = [{ is_locked: true, period_end: '2026-08-15', end_balance: 1200 }]
    const purchases = [
      { amount: 50, statement_reference: '' },
      { amount: 30, statement_reference: 'REF-1' }, // déjà rapproché, exclu
    ]
    const incomes = [{ amount: 200, statement_reference: '' }]

    expect(computeRealBalance({ statements, purchases, incomes })).toEqual({
      balance: 1200 + 200 - 50,
      source: 'statement',
    })
  })

  it('sans relevé, ancré sur le solde initial + mouvements depuis cette date seulement', () => {
    const purchases = [
      { amount: 50, statement_reference: '', date: '2026-08-10' }, // après le solde initial, compté
      { amount: 999, statement_reference: '', date: '2026-01-01' }, // avant, déjà représenté, exclu
    ]
    const incomes = [{ amount: 200, statement_reference: '', date: '2026-08-12' }]

    expect(
      computeRealBalance({
        statements: [{ is_locked: false }],
        purchases,
        incomes,
        openingBalance: { amount: 8000, date: '2026-08-01' },
      })
    ).toEqual({ balance: 8000 + 200 - 50, source: 'opening_balance' })
  })

  it('un relevé verrouillé reste prioritaire même si un solde initial est déclaré', () => {
    const statements = [{ is_locked: true, period_end: '2026-08-15', end_balance: 1200 }]
    const purchases = [{ amount: 50, statement_reference: '', date: '2026-08-10' }]
    const incomes = []

    expect(
      computeRealBalance({
        statements,
        purchases,
        incomes,
        openingBalance: { amount: 8000, date: '2026-08-01' },
      })
    ).toEqual({ balance: 1200 - 50, source: 'statement' })
  })

  it('inclut le net des virements signés, sans les toucher au filtre statement_reference/date des achats-revenus', () => {
    const purchases = []
    const incomes = []
    const transfers = [
      { amount: -500, statement_reference: '', date: '2026-08-10' }, // sortant (déjà signé par l'appelant)
      { amount: 200, statement_reference: 'REF-1', date: '2026-08-05' }, // déjà pointé, exclu
    ]

    expect(
      computeRealBalance({ statements: [{ is_locked: false }], purchases, incomes, transfers })
    ).toEqual({ balance: -500, source: 'ledger' })
  })
})

describe('signTransferLeg', () => {
  it('signe négativement pour le compte source (sortant), avec sa propre référence de pointage', () => {
    const transfer = {
      from_account_id: 1,
      to_account_id: 2,
      amount: 500,
      transfer_date: '2026-08-10',
      from_statement_reference: 'REF-A',
      to_statement_reference: 'REF-B',
    }

    expect(signTransferLeg(transfer, 1)).toEqual({
      amount: -500,
      date: '2026-08-10',
      statement_reference: 'REF-A',
      isOutgoing: true,
    })
  })

  it('signe positivement pour le compte destination (entrant), avec sa propre référence de pointage', () => {
    const transfer = {
      from_account_id: 1,
      to_account_id: 2,
      amount: 500,
      transfer_date: '2026-08-10',
      from_statement_reference: 'REF-A',
      to_statement_reference: 'REF-B',
    }

    expect(signTransferLeg(transfer, 2)).toEqual({
      amount: 500,
      date: '2026-08-10',
      statement_reference: 'REF-B',
      isOutgoing: false,
    })
  })

  it('tolère la forme mappée camelCase (store) en plus de la forme brute', () => {
    const transfer = {
      fromAccountId: 1,
      toAccountId: 2,
      amount: 500,
      date: '2026-08-10',
      fromStatementReference: '',
      toStatementReference: '',
    }

    expect(signTransferLeg(transfer, 1).isOutgoing).toBe(true)
    expect(signTransferLeg(transfer, 2).isOutgoing).toBe(false)
  })
})
