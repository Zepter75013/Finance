<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'

import PageHero from '../Common/PageHero.vue'
import CircularProgress from '../Common/CircularProgress.vue'
import DashboardCharts from './DashboardCharts.vue'
import CategorySubTrendGrid from './CategorySubTrendGrid.vue'
import { usePurchasesStore } from '../../stores/purchases'
import { useRecurringStore } from '../../stores/recurring'
import { formatCurrency } from '../../utils/format'

const emit = defineEmits(['open-recurring'])

const store = usePurchasesStore()
const {
  purchases,
  incomes,
  currentMonthBudgetRemaining,
  currentMonthExpenseBudget,
  currentMonthExpenseSpent,
  accountsList,
  realCurrentBalance,
  latestLockedStatement,
} = storeToRefs(store)

const recurringStore = useRecurringStore()
const { overdueItems: overdueRecurringItems, upcomingItems: upcomingRecurringItems } =
  storeToRefs(recurringStore)

onMounted(() => {
  recurringStore.loadRecurring()
})

const budgetUsedPercent = computed(() => {
  if (!currentMonthExpenseBudget.value) return 0
  return (currentMonthExpenseSpent.value / currentMonthExpenseBudget.value) * 100
})

// La bannière de dépassement de budget se masque pour le mois en cours une
// fois fermée, mais réapparaît naturellement le mois suivant si le budget
// est de nouveau dépassé (clé de stockage incluant le mois).
const BUDGET_ALERT_DISMISSED_KEY = 'budgetAlertDismissedMonth'
const currentMonthKey = new Date().toISOString().slice(0, 7)
const dismissedMonth = ref(localStorage.getItem(BUDGET_ALERT_DISMISSED_KEY))

const showBudgetAlert = computed(
  () => currentMonthBudgetRemaining.value < 0 && dismissedMonth.value !== currentMonthKey
)

function dismissBudgetAlert() {
  dismissedMonth.value = currentMonthKey
  localStorage.setItem(BUDGET_ALERT_DISMISSED_KEY, currentMonthKey)
}

// Passe par l'action du store (et non un simple v-model direct sur le ref)
// pour que le changement de compte actif soit bien persisté en localStorage.
const activeAccountId = computed({
  get: () => store.activeAccountId,
  set: (value) => store.setActiveAccountId(value),
})

// Contrairement à la bannière de budget (masquée par mois), le statut d'une
// récurrence peut changer n'importe quel jour — on retient donc les id déjà
// vus plutôt qu'une période fixe : la bannière réapparaît dès qu'un nouvel
// id (nouveau retard, nouvelle échéance à moins de 7 jours) apparaît. La clé
// est namespacée par compte car les id de récurrence sont globaux, pas par
// compte.
function recurringDismissedKey() {
  return `recurringAlertDismissedIds:${activeAccountId.value}`
}

const dismissedRecurringIds = ref(JSON.parse(localStorage.getItem(recurringDismissedKey()) || '[]'))

watch(activeAccountId, () => {
  dismissedRecurringIds.value = JSON.parse(localStorage.getItem(recurringDismissedKey()) || '[]')
})

const recurringAlertIds = computed(() =>
  [...overdueRecurringItems.value, ...upcomingRecurringItems.value].map((item) => item.id).sort((a, b) => a - b)
)

const showRecurringAlert = computed(() => {
  const dismissedSet = new Set(dismissedRecurringIds.value)
  return recurringAlertIds.value.some((id) => !dismissedSet.has(id))
})

function dismissRecurringAlert() {
  dismissedRecurringIds.value = recurringAlertIds.value
  localStorage.setItem(recurringDismissedKey(), JSON.stringify(recurringAlertIds.value))
}

const recurringAlertText = computed(() => {
  const overdueCount = overdueRecurringItems.value.length
  const upcomingCount = upcomingRecurringItems.value.length
  const total = overdueCount + upcomingCount

  const parts = []
  if (overdueCount) parts.push(`${overdueCount} en retard`)
  if (upcomingCount) parts.push(`${upcomingCount} dans les 7 prochains jours`)

  return `${total} échéance${total > 1 ? 's' : ''} récurrente${total > 1 ? 's' : ''} à traiter : ${parts.join(', ')}.`
})

function yearOf(dateStr) {
  if (!dateStr) return null
  const date = new Date(dateStr)
  return Number.isNaN(date.getTime()) ? null : date.getFullYear()
}

const currentYear = new Date().getFullYear()

// Années réellement disponibles dans les données (+ l'année en cours, même
// sans données) pour borner le sélecteur — pas d'années vides à parcourir.
const availableYears = computed(() => {
  const years = new Set([currentYear])

  for (const purchase of purchases.value) {
    const year = yearOf(purchase.date)
    if (year) years.add(year)
  }

  for (const income of incomes.value) {
    const year = yearOf(income.income_date)
    if (year) years.add(year)
  }

  return Array.from(years).sort((a, b) => b - a)
})

const selectedYear = ref(currentYear)

const canGoToPreviousYear = computed(() => selectedYear.value > Math.min(...availableYears.value))
const canGoToNextYear = computed(() => selectedYear.value < Math.max(...availableYears.value))

function goToPreviousYear() {
  if (canGoToPreviousYear.value) selectedYear.value -= 1
}

function goToNextYear() {
  if (canGoToNextYear.value) selectedYear.value += 1
}

// Les cartes du haut portent désormais sur l'année sélectionnée (et non le
// mois — l'ancien libellé "du mois" ne correspondait plus à des totaux
// calculés sur toutes les données).
const yearPurchases = computed(() => purchases.value.filter((p) => yearOf(p.date) === selectedYear.value))
const yearIncomes = computed(() => incomes.value.filter((i) => yearOf(i.income_date) === selectedYear.value))

const yearExpenseTotal = computed(() =>
  yearPurchases.value.reduce((sum, p) => sum + Number(p.amount || 0), 0)
)
const yearIncomeTotal = computed(() =>
  yearIncomes.value.reduce((sum, i) => sum + Number(i.amount || 0), 0)
)
const yearNetBalance = computed(() => yearIncomeTotal.value - yearExpenseTotal.value)

</script>

<template>
  <main class="dashboard-content dashboard-view">
    <PageHero
      eyebrow="Vue d’ensemble"
      title="Dashboard"
      description="Un coup d’œil sur tes finances : dépenses, revenus, solde et budget."
    >
      <template #actions>
        <div v-if="accountsList.length > 1" class="account-switcher">
          <select v-model.number="activeAccountId" class="account-switcher-select">
            <option v-for="account in accountsList" :key="account.id" :value="account.id">
              {{ account.name }}
            </option>
          </select>
        </div>

        <div class="year-switcher">
          <button
            class="ghost-btn year-switcher-btn"
            type="button"
            :disabled="!canGoToPreviousYear"
            aria-label="Année précédente"
            @click="goToPreviousYear"
          >
            ‹
          </button>
          <span class="year-switcher-value">{{ selectedYear }}</span>
          <button
            class="ghost-btn year-switcher-btn"
            type="button"
            :disabled="!canGoToNextYear"
            aria-label="Année suivante"
            @click="goToNextYear"
          >
            ›
          </button>
        </div>
      </template>
    </PageHero>

    <div v-if="showBudgetAlert" class="budget-alert">
      <span class="budget-alert-icon" aria-hidden="true">⚠️</span>
      <p>
        Budget dépassé ce mois-ci de <strong>{{ formatCurrency(-currentMonthBudgetRemaining) }}</strong>.
      </p>
      <button class="ghost-btn" type="button" @click="dismissBudgetAlert">Masquer</button>
    </div>

    <div v-if="showRecurringAlert" class="budget-alert">
      <span class="budget-alert-icon" aria-hidden="true">📅</span>
      <p>{{ recurringAlertText }}</p>
      <button class="ghost-btn" type="button" @click="emit('open-recurring')">Voir</button>
      <button class="ghost-btn" type="button" @click="dismissRecurringAlert">Masquer</button>
    </div>

    <section class="dashboard-stats-grid">
      <article class="panel stat-card">
        <span>Dépenses ({{ selectedYear }})</span>
        <strong>{{ formatCurrency(yearExpenseTotal) }}</strong>
        <p>Total des achats de l’année</p>
      </article>

      <article class="panel stat-card accent-soft">
        <span>Revenus ({{ selectedYear }})</span>
        <strong>{{ formatCurrency(yearIncomeTotal) }}</strong>
        <p>Total des revenus de l’année</p>
      </article>

      <article class="panel stat-card">
        <span>Solde net ({{ selectedYear }})</span>
        <strong>{{ formatCurrency(yearNetBalance) }}</strong>
        <p>{{ yearNetBalance >= 0 ? 'Solde positif' : 'Solde négatif' }}</p>
      </article>
    </section>

    <section class="dashboard-current-grid">
      <article class="panel stat-card accent-line stat-card-ring">
        <CircularProgress
          :percent="budgetUsedPercent"
          :size="76"
          :stroke-width="8"
          :color="budgetUsedPercent > 100 ? 'var(--danger)' : 'var(--accent)'"
        />
        <div class="stat-card-ring-text">
          <span>Budget restant (mois en cours)</span>
          <strong>{{ formatCurrency(currentMonthBudgetRemaining) }}</strong>
          <p>{{ currentMonthBudgetRemaining >= 0 ? 'Budget encore disponible' : 'Budget dépassé' }}</p>
        </div>
      </article>

      <article class="panel stat-card">
        <span>Solde réel (dernier relevé + non pointé)</span>
        <template v-if="latestLockedStatement">
          <strong>{{ formatCurrency(realCurrentBalance) }}</strong>
          <p>D’après le relevé n°{{ latestLockedStatement.reference }}, hors budget</p>
        </template>
        <template v-else>
          <strong>—</strong>
          <p>Verrouille un premier relevé dans Pointage pour l’afficher</p>
        </template>
      </article>
    </section>

    <DashboardCharts :selected-year="selectedYear" />
    <CategorySubTrendGrid :selected-year="selectedYear" />
  </main>
</template>

<style scoped>
.dashboard-view {
  display: grid;
  gap: 0.9rem;
}

.budget-alert {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.85rem 1.1rem;
  border-radius: 16px;
  background: rgba(239, 68, 68, 0.12);
  border: 1px solid rgba(239, 68, 68, 0.24);
}

.budget-alert-icon {
  flex-shrink: 0;
  font-size: 1.1rem;
}

.budget-alert p {
  flex: 1;
  margin: 0;
  color: var(--negative-text, #fca5a5);
  font-size: 0.92rem;
}

.dashboard-stats-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.7rem;
}

.dashboard-current-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.7rem;
}

.stat-card-ring {
  display: flex;
  align-items: center;
  gap: 0.9rem;
}

.stat-card-ring-text {
  min-width: 0;
}

.stat-card-ring-text span {
  margin-bottom: 0.3rem;
}

.stat-card-ring-text strong {
  margin-bottom: 0.2rem;
}

.stat-card-ring-text p {
  margin: 0;
}

.account-switcher {
  display: flex;
  align-items: center;
  padding: 0.3rem 0.7rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.04);
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
}

.account-switcher-select {
  border: none;
  background: transparent;
  color: var(--text, #eef1f3);
  font-weight: 700;
  font-size: 0.95rem;
  padding: 0.1rem 0;
  outline: none;
  cursor: pointer;
}

.year-switcher {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.3rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.04);
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
}

.year-switcher-btn {
  width: 34px;
  height: 34px;
  min-width: 0;
  padding: 0;
  font-size: 1.1rem;
  line-height: 1;
  display: grid;
  place-items: center;
}

.year-switcher-value {
  min-width: 3.2rem;
  text-align: center;
  font-weight: 700;
  color: var(--text, #eef1f3);
}

@media (max-width: 1100px) {
  .dashboard-stats-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .dashboard-stats-grid,
  .dashboard-current-grid {
    grid-template-columns: 1fr;
  }
}
</style>
