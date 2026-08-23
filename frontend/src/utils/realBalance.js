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

// Solde réel = solde de fin du dernier relevé verrouillé + tous les achats/
// revenus pas encore rattachés à un relevé (statement_reference vide).
export function computeRealBalance({ statements, purchases, incomes }) {
  const anchor = findLatestLockedStatement(statements)
  if (!anchor) return null

  const unreconciledIncome = incomes
    .filter((i) => !i.statement_reference)
    .reduce((sum, i) => sum + Number(i.amount || 0), 0)

  const unreconciledExpense = purchases
    .filter((p) => !p.statement_reference)
    .reduce((sum, p) => sum + Number(p.amount || 0), 0)

  return Number(anchor.end_balance || 0) + unreconciledIncome - unreconciledExpense
}
