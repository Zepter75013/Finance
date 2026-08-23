// Opère sur la forme brute (snake_case) telle que renvoyée par
// fetchPurchases/fetchIncomes/fetchStatements — pas la forme mappée en
// camelCase utilisée pour l'affichage dans stores/purchases.js.

export function findLatestLockedStatement(statements) {
  const locked = statements.filter((s) => s.is_locked)
  if (!locked.length) return null

  return [...locked].sort((a, b) => {
    const dateA = new Date(a.period_end || a.statement_date || 0)
    const dateB = new Date(b.period_end || b.statement_date || 0)
    return dateB - dateA
  })[0]
}

// Tolère à la fois la forme mappée du store (purchase.date) et la forme
// brute utilisée par ConsolidatedView.vue (purchase_date/income_date).
function dateOf(item) {
  return item.date || item.purchase_date || item.income_date || ''
}

// Signe un virement du point de vue d'un compte donné : négatif si ce compte
// en est la source (débit), positif s'il en est la destination (crédit).
// Tolère la forme brute (from_account_id) et mappée (fromAccountId) — un
// seul endroit calcule "ce virement sort-il de ce compte", partagé par le
// store, la vue consolidée et l'écran Pointage.
export function signTransferLeg(transfer, accountId) {
  const fromId = transfer.from_account_id ?? transfer.fromAccountId
  const isOutgoing = Number(fromId) === Number(accountId)
  const amount = Number(transfer.amount || 0)

  return {
    amount: isOutgoing ? -amount : amount,
    date: transfer.transfer_date ?? transfer.date,
    statement_reference:
      (isOutgoing
        ? transfer.from_statement_reference ?? transfer.fromStatementReference
        : transfer.to_statement_reference ?? transfer.toStatementReference) || '',
    isOutgoing,
  }
}

// Solde réel — trois niveaux d'ancrage, du plus fiable au moins informé :
//
// 1. Relevé verrouillé : solde de fin du relevé + mouvements pas encore
//    rattachés à un relevé (statement_reference vide) depuis.
// 2. Solde initial déclaré (compte type Livret, sans relevé) : montant
//    déclaré à une date + mouvements non rattachés à un relevé depuis CETTE
//    date seulement — pas tout l'historique, pour ne pas compter deux fois
//    des mouvements que ce solde initial est censé déjà représenter.
// 3. Repli : mouvements non rattachés à un relevé, sans restriction de date.
//    Correct pour un Livret sans solde initial déclaré (rien n'y est jamais
//    rattaché à un relevé, donc ça revient à sommer tout l'historique) ; pour
//    un compte courant ayant eu un relevé verrouillé puis débloqué/supprimé,
//    ne compte que le delta depuis le dernier rattachement, pas le solde
//    complet — limite connue, pas corrigible sans cet ancrage.
//
// `transfers` est déjà signé par l'appelant (signTransferLeg) — sommé
// directement, sans valeur absolue, à la différence des achats/revenus.
//
// `source` ('statement' | 'opening_balance' | 'ledger') indique seulement le
// niveau d'ancrage utilisé, pas un degré d'incertitude : le calcul reste
// exact dans tous les cas.
export function computeRealBalance({ statements, purchases, incomes, transfers = [], openingBalance }) {
  const anchor = findLatestLockedStatement(statements)
  const sinceDate = !anchor && openingBalance?.date ? openingBalance.date : null

  const sumUnreconciled = (items) =>
    items
      .filter((i) => !i.statement_reference && (!sinceDate || dateOf(i) >= sinceDate))
      .reduce((sum, i) => sum + Number(i.amount || 0), 0)

  const unreconciledIncome = sumUnreconciled(incomes)
  const unreconciledExpense = sumUnreconciled(purchases)
  const unreconciledTransferNet = sumUnreconciled(transfers)

  if (anchor) {
    return {
      balance:
        Number(anchor.end_balance || 0) + unreconciledIncome - unreconciledExpense + unreconciledTransferNet,
      source: 'statement',
    }
  }

  if (openingBalance?.date && openingBalance.amount != null) {
    return {
      balance: Number(openingBalance.amount) + unreconciledIncome - unreconciledExpense + unreconciledTransferNet,
      source: 'opening_balance',
    }
  }

  return {
    balance: unreconciledIncome - unreconciledExpense + unreconciledTransferNet,
    source: 'ledger',
  }
}
