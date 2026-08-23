<script setup>
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import PageHero from '../Common/PageHero.vue'
import { usePurchasesStore } from '../../stores/purchases'
import { fetchPurchases } from '../../services/purchases'
import { fetchIncomes } from '../../services/incomes'
import { fetchStatements } from '../../services/statements'
import { fetchTransfers } from '../../services/transfers'
import { formatCurrency } from '../../utils/format'
import { computeRealBalance, signTransferLeg } from '../../utils/realBalance'

const emit = defineEmits(['open-account'])

const store = usePurchasesStore()
const { accountsList } = storeToRefs(store)

const rows = ref([])
const isLoading = ref(false)

const currentMonthKey = new Date().toISOString().slice(0, 7)

async function loadAccountRow(account) {
  try {
    const [purchases, incomes, statements, transfers] = await Promise.all([
      fetchPurchases(account.id),
      fetchIncomes(account.id),
      fetchStatements(account.id).catch(() => []),
      fetchTransfers(account.id).catch(() => []),
    ])

    const monthExpense = purchases
      .filter((p) => p.purchase_date?.slice(0, 7) === currentMonthKey)
      .reduce((sum, p) => sum + Number(p.amount || 0), 0)

    const monthIncome = incomes
      .filter((i) => i.income_date?.slice(0, 7) === currentMonthKey)
      .reduce((sum, i) => sum + Number(i.amount || 0), 0)

    const balanceInfo = computeRealBalance({
      statements,
      purchases,
      incomes,
      transfers: transfers.map((t) => signTransferLeg(t, account.id)),
      openingBalance: account.openingBalanceAmount != null
        ? { amount: account.openingBalanceAmount, date: account.openingBalanceDate }
        : null,
    })

    return {
      account,
      monthExpense,
      monthIncome,
      realBalance: balanceInfo.balance,
      isEstimateBalance: balanceInfo.source !== 'statement',
      loadError: false,
    }
  } catch {
    return { account, monthExpense: null, monthIncome: null, realBalance: null, isEstimateBalance: false, loadError: true }
  }
}

async function loadAll() {
  isLoading.value = true

  try {
    rows.value = await Promise.all(accountsList.value.map(loadAccountRow))
  } finally {
    isLoading.value = false
  }
}

onMounted(loadAll)

const validRows = computed(() => rows.value.filter((row) => !row.loadError))

const totals = computed(() => ({
  monthExpense: validRows.value.reduce((sum, row) => sum + (row.monthExpense || 0), 0),
  monthIncome: validRows.value.reduce((sum, row) => sum + (row.monthIncome || 0), 0),
  realBalance: validRows.value.reduce((sum, row) => sum + (row.realBalance || 0), 0),
}))

// Le total mélange des soldes ancrés à un relevé bancaire et des soldes non
// ancrés (comptes type Livret, ou compte courant sans relevé verrouillé
// pour l'instant) — jamais présenté comme un total homogène dans ce cas.
const hasIncompleteBalance = computed(() =>
  validRows.value.some((row) => row.isEstimateBalance) || rows.value.some((row) => row.loadError)
)

function openAccount(accountId) {
  store.setActiveAccountId(accountId)
  emit('open-account')
}
</script>

<template>
  <main class="dashboard-content consolidated-view">
    <PageHero
      eyebrow="Vue d'ensemble"
      title="Tous les comptes"
      description="Solde réel et mouvements du mois, compte par compte."
    />

    <section v-if="!accountsList.length" class="panel empty-state">
      <div class="empty-state-icon">€</div>
      <h2>Aucun compte accessible</h2>
      <p>Crée un premier compte pour voir apparaître une vision consolidée.</p>
    </section>

    <section v-else-if="accountsList.length === 1" class="panel empty-state">
      <div class="empty-state-icon">€</div>
      <h2>Un seul compte accessible</h2>
      <p>Tu n'as accès qu'à un seul compte — ce tableau n'apporte rien de plus que le Dashboard.</p>
    </section>

    <section v-else class="panel consolidated-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Comptes</p>
          <h2>{{ accountsList.length }} comptes</h2>
        </div>
      </div>

      <p v-if="isLoading" class="consolidated-empty">Chargement...</p>

      <div v-else class="consolidated-table-wrap">
        <table class="consolidated-table">
          <thead>
            <tr>
              <th>Compte</th>
              <th>Solde réel</th>
              <th>Dépenses ({{ currentMonthKey }})</th>
              <th>Revenus ({{ currentMonthKey }})</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in rows" :key="row.account.id">
              <td>{{ row.account.name }}</td>

              <template v-if="row.loadError">
                <td colspan="3" class="consolidated-row-error">Impossible de charger ce compte.</td>
              </template>
              <template v-else>
                <td>
                  <strong>{{ formatCurrency(row.realBalance) }}<span v-if="row.isEstimateBalance">&nbsp;*</span></strong>
                </td>
                <td>{{ formatCurrency(row.monthExpense) }}</td>
                <td>{{ formatCurrency(row.monthIncome) }}</td>
              </template>

              <td>
                <button class="ghost-btn" type="button" @click="openAccount(row.account.id)">Voir</button>
              </td>
            </tr>

            <tr class="consolidated-total-row">
              <td>Total</td>
              <td>
                {{ formatCurrency(totals.realBalance) }}<span v-if="hasIncompleteBalance">&nbsp;*</span>
              </td>
              <td>{{ formatCurrency(totals.monthExpense) }}</td>
              <td>{{ formatCurrency(totals.monthIncome) }}</td>
              <td></td>
            </tr>
          </tbody>
        </table>

        <p v-if="hasIncompleteBalance" class="consolidated-incomplete-caption">
          * Solde non ancré à un relevé bancaire (basé sur les mouvements non rapprochés) pour un ou plusieurs
          comptes, ou compte n'ayant pas pu être chargé — le total mélange donc des soldes vérifiés et non
          vérifiés.
        </p>
      </div>
    </section>
  </main>
</template>

<style scoped>
.consolidated-view {
  display: grid;
  gap: 0.9rem;
}

.empty-state {
  padding: 2.2rem 1.5rem;
  text-align: center;
  display: grid;
  justify-items: center;
  gap: 0.5rem;
}

.empty-state-icon {
  width: 48px;
  height: 48px;
  border-radius: 14px;
  display: grid;
  place-items: center;
  background: rgba(var(--tint-rgb), 0.08);
  color: var(--text, #eef1f3);
  font-size: 1.3rem;
  font-weight: 700;
}

.empty-state h2 {
  margin: 0;
  font-size: 1.1rem;
  color: var(--text, #eef1f3);
}

.empty-state p {
  margin: 0;
  color: var(--text-dim, #8a939d);
  font-size: 0.88rem;
}

.consolidated-card {
  padding: 1.1rem;
}

.consolidated-empty {
  margin: 0.9rem 0 0;
  color: var(--text-dim, #8a939d);
  font-size: 0.88rem;
}

.consolidated-table-wrap {
  margin-top: 0.9rem;
  overflow-x: auto;
}

.consolidated-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.88rem;
}

.consolidated-table th {
  text-align: left;
  padding: 0.6rem 0.75rem;
  color: var(--text-dim, #8a939d);
  font-weight: 600;
  font-size: 0.76rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  border-bottom: 1px solid rgba(var(--tint-rgb), 0.08);
  white-space: nowrap;
}

.consolidated-table td {
  padding: 0.6rem 0.75rem;
  border-bottom: 1px solid rgba(var(--tint-rgb), 0.045);
  color: var(--text, #eef1f3);
}

.consolidated-table tbody tr:last-child td {
  border-bottom: none;
}

.consolidated-row-error {
  color: var(--negative-text);
}

.consolidated-total-row td {
  font-weight: 700;
  background: rgba(var(--tint-rgb), 0.03);
}

.consolidated-incomplete-caption {
  margin: 0.6rem 0 0;
  color: var(--text-dim, #8a939d);
  font-size: 0.76rem;
}
</style>
