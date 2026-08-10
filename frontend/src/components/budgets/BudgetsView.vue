<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import {
  Chart as ChartJS,
  Title,
  Tooltip,
  Legend,
  CategoryScale,
  LinearScale,
  BarElement,
  BarController,
} from 'chart.js'

import PageHero from '../Common/PageHero.vue'
import QuickCreateModal from '../Common/QuickCreateModal.vue'
import { usePurchasesStore } from '../../stores/purchases'

ChartJS.register(Title, Tooltip, Legend, CategoryScale, LinearScale, BarElement, BarController)

const purchasesStore = usePurchasesStore()

const {
  purchases,
  incomes,
  categoriesList,
  currentMonthExpenseBudget: monthlyBudget,
  currentMonthExpenseSpent: totalMonthAmount,
  currentMonthBudgetRemaining,
} = storeToRefs(purchasesStore)

function formatAmount(value) {
  return new Intl.NumberFormat('fr-FR', {
    style: 'currency',
    currency: 'EUR',
  }).format(Number(value || 0))
}

const currentMonthKey = new Date().toISOString().slice(0, 7)

function formatMonthLabel(monthKey) {
  const [year, month] = monthKey.split('-').map(Number)
  const formatted = new Intl.DateTimeFormat('fr-FR', { month: 'long', year: 'numeric' }).format(
    new Date(year, month - 1, 1)
  )
  return formatted.charAt(0).toUpperCase() + formatted.slice(1)
}

function actualsForMonth(monthKey) {
  const actualExpense = purchases.value
    .filter((p) => p.date?.slice(0, 7) === monthKey)
    .reduce((sum, p) => sum + Number(p.amount || 0), 0)

  const actualIncome = incomes.value
    .filter((i) => i.income_date?.slice(0, 7) === monthKey)
    .reduce((sum, i) => sum + Number(i.amount || 0), 0)

  return { actualExpense, actualIncome }
}

// Historique mois par mois. Un mois déjà terminé est une certitude : on
// affiche simplement ce qui s'est réellement passé, pas de "budget" à lui
// comparer. Le mois en cours, lui, n'est pas terminé — on lui associe une
// provision (moyenne des mois précédents) à comparer à ce qui est déjà
// enregistré, sans extrapoler ce réalisé partiel sur le mois complet.
const monthlyRetrospective = computed(() => {
  const monthKeys = new Set([currentMonthKey])

  for (const purchase of purchases.value) {
    if (purchase.date) monthKeys.add(purchase.date.slice(0, 7))
  }
  for (const income of incomes.value) {
    if (income.income_date) monthKeys.add(income.income_date.slice(0, 7))
  }

  const sortedKeys = Array.from(monthKeys).sort()
  const pastKeys = sortedKeys.filter((key) => key < currentMonthKey)

  const pastActuals = pastKeys.map(actualsForMonth)
  const provisionedExpense = pastActuals.length
    ? pastActuals.reduce((sum, m) => sum + m.actualExpense, 0) / pastActuals.length
    : 0
  const provisionedIncome = pastActuals.length
    ? pastActuals.reduce((sum, m) => sum + m.actualIncome, 0) / pastActuals.length
    : 0

  return sortedKeys.map((monthKey) => {
    const isCurrent = monthKey === currentMonthKey
    const { actualExpense, actualIncome } = actualsForMonth(monthKey)

    return {
      key: monthKey,
      label: formatMonthLabel(monthKey),
      isCurrent,
      actualExpense,
      actualIncome,
      actualSavings: actualIncome - actualExpense,
      provisionedExpense: isCurrent ? provisionedExpense : null,
      provisionedIncome: isCurrent ? provisionedIncome : null,
      provisionedSavings: isCurrent ? provisionedIncome - provisionedExpense : null,
    }
  })
})

const pastMonths = computed(() => monthlyRetrospective.value.filter((m) => !m.isCurrent))
const currentMonthRow = computed(() => monthlyRetrospective.value.find((m) => m.isCurrent) ?? null)

const currentMonthLabel = formatMonthLabel(currentMonthKey)

// Les achats référencent leur catégorie par id, mais les revenus ne stockent
// que le nom de la catégorie (source) — on construit cette table pour les
// rattacher à la même ligne de budget que leur catégorie « revenu ».
const revenueCategoryIdByName = computed(() => {
  const map = new Map()
  for (const category of categoriesList.value) {
    if (category.type === 'revenu') map.set(category.name, category.id)
  }
  return map
})

const categorySpendForCurrentMonth = computed(() => {
  const totals = new Map()

  for (const purchase of purchases.value) {
    if (purchase.date?.slice(0, 7) !== currentMonthKey) continue

    totals.set(purchase.categoryId, (totals.get(purchase.categoryId) || 0) + Number(purchase.amount || 0))
  }

  for (const income of incomes.value) {
    if (income.income_date?.slice(0, 7) !== currentMonthKey) continue

    const categoryId = revenueCategoryIdByName.value.get(income.category)
    if (categoryId === undefined) continue

    totals.set(categoryId, (totals.get(categoryId) || 0) + Number(income.amount || 0))
  }

  return totals
})

// Le budget d'une catégorie pour le mois en cours : la moyenne réelle des
// mois précédents (calculée par le store, seule source de vérité — le
// Dashboard s'y réfère aussi), sauf si un ajustement ponctuel a été
// enregistré pour ce mois précis (via "Ajuster ce mois").
const categoryBudgetRows = computed(() => {
  return categoriesList.value
    .map((category) => {
      const isRevenue = category.type === 'revenu'
      const spent = categorySpendForCurrentMonth.value.get(category.id) || 0
      const average = purchasesStore.suggestedCategoryBudget(category)
      const isOverridden = purchasesStore.isCategoryBudgetOverridden(category.id, currentMonthKey)
      const budget = purchasesStore.currentMonthCategoryBudget(category)

      const remaining = budget - spent
      const usagePercent = budget > 0 ? Math.min(Math.round((spent / budget) * 100), 100) : 0

      return {
        ...category,
        isRevenue,
        spent,
        budget,
        average,
        remaining,
        usagePercent,
        isOverridden,
      }
    })
    .sort((a, b) => a.name.localeCompare(b.name, 'fr', { sensitivity: 'base' }))
})

const isBudgetModalOpen = ref(false)
const budgetModalCategory = ref(null)
const budgetModalError = ref('')

const budgetModalConfig = computed(() => {
  const category = budgetModalCategory.value
  const isRevenue = category?.type === 'revenu'

  return {
    eyebrow: `Ajuster ce mois — ${currentMonthLabel}`,
    title: `${isRevenue ? 'Revenu prévu' : 'Budget'} pour « ${category?.name ?? ''} »`,
    label: isRevenue ? 'Montant prévu pour ce mois (€)' : 'Montant pour ce mois (€)',
    placeholder: 'Ex: 150',
    confirmLabel: 'Enregistrer',
    initialValue: category?.budget > 0 ? category.budget.toFixed(2) : '',
  }
})

function handleAdjustCategoryBudget(category) {
  budgetModalCategory.value = category
  budgetModalError.value = ''
  isBudgetModalOpen.value = true
}

function handleClearCategoryBudget(category) {
  purchasesStore.clearCategoryBudgetOverride(category.id, currentMonthKey)
}

function submitBudgetModal(rawValue) {
  budgetModalError.value = ''
  const normalized = rawValue.replace(',', '.').trim()

  try {
    purchasesStore.setCategoryBudgetOverride(budgetModalCategory.value.id, currentMonthKey, normalized)
    isBudgetModalOpen.value = false
  } catch (err) {
    budgetModalError.value = err instanceof Error ? err.message : 'Montant invalide.'
  }
}

// Carte "Budget mensuel" du haut : "monthlyBudget"/"totalMonthAmount" viennent
// directement du store (currentMonthExpenseBudget/Spent) — même source que le
// détail par catégorie ci-dessous et que la carte "Budget restant" du
// Dashboard, pour ne jamais afficher deux chiffres différents.
const budgetRemaining = computed(() => Math.max(currentMonthBudgetRemaining.value, 0))
const budgetUsagePercent = computed(() => {
  if (!monthlyBudget.value) return 0
  return Math.min(Math.round((totalMonthAmount.value / monthlyBudget.value) * 100), 100)
})
const purchaseCount = computed(
  () => purchases.value.filter((p) => p.date?.slice(0, 7) === currentMonthKey).length
)

const formattedMonthlyBudget = computed(() => formatAmount(monthlyBudget.value))
const formattedSpentAmount = computed(() => formatAmount(totalMonthAmount.value))
const formattedRemainingBudget = computed(() => formatAmount(budgetRemaining.value))

const budgetStatusLabel = computed(() => {
  if (budgetUsagePercent.value >= 100) return 'Budget atteint'
  if (budgetUsagePercent.value >= 80) return 'Budget sous tension'
  return 'Budget maîtrisé'
})

const hasMonthlyHistory = computed(() => pastMonths.value.length > 0)

const totalActualSavings = computed(() =>
  pastMonths.value.reduce((sum, m) => sum + m.actualSavings, 0)
)
const monthsThatWouldHaveSaved = computed(
  () => pastMonths.value.filter((m) => m.actualSavings > 0).length
)

const historyChartCanvas = ref(null)
let historyChart = null

function destroyHistoryChart() {
  historyChart?.destroy()
  historyChart = null
}

async function renderHistoryChart() {
  destroyHistoryChart()
  await nextTick()

  if (!historyChartCanvas.value || !monthlyRetrospective.value.length) return

  historyChart = new ChartJS(historyChartCanvas.value.getContext('2d'), {
    type: 'bar',
    data: {
      labels: monthlyRetrospective.value.map((m) => (m.isCurrent ? `${m.label} (en cours)` : m.label)),
      datasets: [
        {
          label: 'Épargne réelle',
          data: monthlyRetrospective.value.map((m) => m.actualSavings),
          backgroundColor: 'rgba(114, 137, 152, 0.85)',
          borderRadius: 4,
        },
        {
          label: 'Provision (mois en cours)',
          data: monthlyRetrospective.value.map((m) => m.provisionedSavings),
          backgroundColor: 'rgba(143, 168, 160, 0.55)',
          borderRadius: 4,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        x: { grid: { display: false } },
        y: { grid: { color: 'rgba(148, 163, 184, 0.14)' } },
      },
      plugins: {
        legend: { position: 'top' },
      },
    },
  })
}

watch(monthlyRetrospective, renderHistoryChart)
onMounted(renderHistoryChart)
onBeforeUnmount(destroyHistoryChart)
</script>

<template>
  <main class="dashboard-content budgets-view">
    <PageHero
      eyebrow="Pilotage mensuel"
      title="Budget"
      description="Suivez votre enveloppe mensuelle et gardez un œil sur vos dépenses."
    />

    <section class="budgets-stats-grid">
      <article class="panel stat-card">
        <p class="eyebrow">Budget mensuel</p>
        <h2>{{ formattedMonthlyBudget }}</h2>
        <p>Moyenne des mois précédents, ajustements inclus</p>
      </article>

      <article class="panel stat-card">
        <p class="eyebrow">Dépensé</p>
        <h2>{{ formattedSpentAmount }}</h2>
        <p>
          {{ purchaseCount }} achat{{ purchaseCount > 1 ? 's' : '' }}
          enregistré{{ purchaseCount > 1 ? 's' : '' }}
        </p>
      </article>

      <article class="panel stat-card">
        <p class="eyebrow">Reste</p>
        <h2>{{ formattedRemainingBudget }}</h2>
        <p>{{ budgetStatusLabel }}</p>
      </article>

      <article class="panel stat-card">
        <p class="eyebrow">Utilisation</p>
        <h2>{{ budgetUsagePercent }}%</h2>
        <p>Taux de consommation du budget</p>
      </article>
    </section>

    <section class="budgets-layout">
      <article class="panel budget-overview-card">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Vue d’ensemble</p>
            <h2>Progression du budget</h2>
          </div>
        </div>

        <div class="budget-progress-summary">
          <div>
            <span class="budget-progress-label">Budget utilisé</span>
            <strong>{{ budgetUsagePercent }}%</strong>
          </div>
          <span class="budget-progress-caption">{{ budgetStatusLabel }}</span>
        </div>

        <div class="budget-progress-track" aria-hidden="true">
          <div
            class="budget-progress-bar"
            :style="{ width: `${Math.min(budgetUsagePercent, 100)}%` }"
          ></div>
        </div>

        <div class="budget-progress-meta">
          <span>{{ formattedSpentAmount }} dépensés</span>
          <span>{{ formattedMonthlyBudget }} prévus</span>
        </div>
      </article>

      <aside class="panel budget-summary-card">
        <p class="eyebrow">Résumé</p>
        <h2>Ce mois-ci</h2>

        <div class="summary-list">
          <div class="summary-row">
            <span>Budget mensuel</span>
            <strong>{{ formattedMonthlyBudget }}</strong>
          </div>

          <div class="summary-row">
            <span>Dépenses suivies</span>
            <strong>{{ formattedSpentAmount }}</strong>
          </div>

          <div class="summary-row">
            <span>Budget restant</span>
            <strong>{{ formattedRemainingBudget }}</strong>
          </div>

          <div class="summary-row">
            <span>Achats enregistrés</span>
            <strong>{{ purchaseCount }}</strong>
          </div>
        </div>
      </aside>
    </section>

    <section class="panel category-budgets-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">{{ currentMonthLabel }}</p>
          <h2>Budgets par catégorie</h2>
        </div>
      </div>

      <p class="category-budgets-caption">
        Le budget de chaque catégorie correspond à sa moyenne réelle sur les mois précédents.
        Utilise « Ajuster ce mois » pour t'en écarter ponctuellement — l'ajustement ne vaut que
        pour {{ currentMonthLabel }} et retombera sur la moyenne le mois suivant.
      </p>

      <div v-if="!categoryBudgetRows.length" class="category-budgets-empty">
        Crée d'abord une catégorie pour lui définir un budget.
      </div>

      <div v-else class="category-budgets-list">
        <article v-for="category in categoryBudgetRows" :key="category.id" class="category-budget-row">
          <div class="category-budget-heading">
            <div class="category-budget-name">
              <h3>{{ category.name }}</h3>
              <span class="category-type-badge" :class="category.isRevenue ? 'is-revenue' : 'is-expense'">
                {{ category.isRevenue ? 'Revenu' : 'Achat' }}
              </span>
            </div>
            <span v-if="category.isOverridden" class="category-budget-override-badge">
              Ajusté pour {{ currentMonthLabel }}
            </span>
          </div>

          <div v-if="category.budget > 0" class="budget-progress-track" aria-hidden="true">
            <div
              class="budget-progress-bar"
              :class="{ 'is-revenue': category.isRevenue }"
              :style="{ width: `${category.usagePercent}%` }"
            ></div>
          </div>

          <div class="category-budget-meta">
            <span :class="category.isRevenue ? 'amount-positive' : 'amount-negative'">
              {{ formatAmount(category.spent) }}
              {{ category.isRevenue ? 'reçus' : 'dépensés' }}
            </span>
            <span>{{ category.budget > 0 ? `${formatAmount(category.budget)} prévus` : 'Pas encore de moyenne' }}</span>
          </div>

          <p v-if="category.isOverridden && category.average > 0" class="category-budget-suggestion">
            Moyenne habituelle : {{ formatAmount(category.average) }} / mois
          </p>

          <div class="category-budget-actions">
            <button class="ghost-btn" type="button" @click="handleAdjustCategoryBudget(category)">
              Ajuster ce mois
            </button>

            <button
              v-if="category.isOverridden"
              class="ghost-btn"
              type="button"
              @click="handleClearCategoryBudget(category)"
            >
              Revenir à la moyenne
            </button>
          </div>
        </article>
      </div>
    </section>

    <section class="panel budget-history-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Historique</p>
          <h2>Épargne mois par mois</h2>
        </div>
      </div>

      <div v-if="!hasMonthlyHistory" class="category-budgets-empty">
        Aucun achat ou revenu enregistré pour le moment.
      </div>

      <template v-else>
        <p class="history-caption">
          Sur {{ pastMonths.length }} mois terminé{{ pastMonths.length > 1 ? 's' : '' }}, {{ monthsThatWouldHaveSaved }}
          aurai{{ monthsThatWouldHaveSaved > 1 ? 'ent' : 't' }} permis d'épargner.
        </p>

        <div class="history-totals">
          <div class="history-total-figure">
            <span class="savings-label">Épargne réelle cumulée</span>
            <strong :class="totalActualSavings >= 0 ? 'amount-positive' : 'amount-negative'">
              {{ formatAmount(totalActualSavings) }}
            </strong>
            <span class="savings-hint">Sur les mois déjà terminés</span>
          </div>

          <div v-if="currentMonthRow" class="history-total-figure">
            <span class="savings-label">{{ currentMonthRow.label }} — en cours</span>
            <strong :class="currentMonthRow.actualSavings >= 0 ? 'amount-positive' : 'amount-negative'">
              {{ formatAmount(currentMonthRow.actualSavings) }}
            </strong>
            <span class="savings-hint">
              Réalisé partiel — provision : {{ formatAmount(currentMonthRow.provisionedSavings) }}
            </span>
          </div>
        </div>

        <div class="history-chart-wrap">
          <canvas ref="historyChartCanvas"></canvas>
        </div>

        <div class="history-table-wrap">
          <table class="history-table">
            <thead>
              <tr>
                <th>Mois</th>
                <th>Dépenses réelles</th>
                <th>Revenus réels</th>
                <th>Épargne réelle</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="month in pastMonths" :key="month.key">
                <td>{{ month.label }}</td>
                <td>{{ formatAmount(month.actualExpense) }}</td>
                <td>{{ formatAmount(month.actualIncome) }}</td>
                <td :class="month.actualSavings >= 0 ? 'amount-positive' : 'amount-negative'">
                  {{ formatAmount(month.actualSavings) }}
                </td>
              </tr>
              <tr v-if="currentMonthRow" class="history-current-row">
                <td>{{ currentMonthRow.label }} (en cours)</td>
                <td>
                  {{ formatAmount(currentMonthRow.actualExpense) }}
                  <span class="history-provision-hint">/ provision {{ formatAmount(currentMonthRow.provisionedExpense) }}</span>
                </td>
                <td>
                  {{ formatAmount(currentMonthRow.actualIncome) }}
                  <span class="history-provision-hint">/ provision {{ formatAmount(currentMonthRow.provisionedIncome) }}</span>
                </td>
                <td :class="currentMonthRow.actualSavings >= 0 ? 'amount-positive' : 'amount-negative'">
                  {{ formatAmount(currentMonthRow.actualSavings) }}
                  <span class="history-provision-hint">/ provision {{ formatAmount(currentMonthRow.provisionedSavings) }}</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </section>

    <QuickCreateModal
      v-model="isBudgetModalOpen"
      :eyebrow="budgetModalConfig.eyebrow"
      :title="budgetModalConfig.title"
      :label="budgetModalConfig.label"
      :placeholder="budgetModalConfig.placeholder"
      :confirm-label="budgetModalConfig.confirmLabel"
      :initial-value="budgetModalConfig.initialValue"
      :error-message="budgetModalError"
      @confirm="submitBudgetModal"
    />
  </main>
</template>

<style scoped>
.budgets-view {
  display: grid;
  gap: 0.9rem;
}

.budgets-stats-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.7rem;
}

.stat-card h2 {
  margin: 0.22rem 0;
}

.stat-card p:last-child {
  color: var(--text-muted, #94a3b8);
}

.budgets-layout {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(320px, 1fr);
  gap: 0.7rem;
}

.budget-overview-card,
.budget-summary-card {
  min-height: 100%;
  padding: 1.05rem;
}

.budget-overview-card .panel-header,
.budget-summary-card > .eyebrow,
.budget-summary-card > h2 {
  padding-inline: 0.15rem;
}

.budget-progress-summary {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 1rem;
  margin: 0.9rem 0 0.7rem;
}

.budget-progress-label {
  display: block;
  margin-bottom: 0.35rem;
  color: var(--text-muted, #94a3b8);
  font-size: 0.92rem;
}

.budget-progress-caption {
  color: var(--text-muted, #94a3b8);
  font-size: 0.92rem;
}

.budget-progress-track {
  width: 100%;
  height: 12px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.16);
  overflow: hidden;
}

.budget-progress-bar {
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(90deg, #c7d9d1 0%, #9fb8ae 100%);
}

.budget-progress-bar.is-revenue {
  background: linear-gradient(90deg, #bfe0c9 0%, #7fb894 100%);
}

.amount-positive {
  color: var(--positive-text, #bfe0c9);
  font-weight: 600;
}

.amount-negative {
  color: var(--negative-text, #e1b4b4);
  font-weight: 600;
}

.budget-progress-meta {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  margin-top: 0.85rem;
  color: var(--text-muted, #94a3b8);
  font-size: 0.92rem;
}

.summary-list {
  display: grid;
  gap: 0.8rem;
  margin-top: 0.9rem;
}

.summary-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding-bottom: 0.85rem;
  border-bottom: 1px solid rgba(148, 163, 184, 0.12);
}

.summary-row:last-child {
  padding-bottom: 0;
  border-bottom: none;
}

@media (max-width: 1100px) {
  .budgets-stats-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .budgets-layout {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .budgets-stats-grid {
    grid-template-columns: 1fr;
  }

  .budget-progress-summary,
  .budget-progress-meta,
  .summary-row,
  .category-budget-meta {
    flex-direction: column;
    align-items: flex-start;
  }

  .category-budget-actions {
    flex-wrap: wrap;
  }
}

.savings-label {
  color: var(--text-dim, #8a939d);
  font-size: 0.76rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.savings-figure strong {
  font-size: 1.4rem;
}

.savings-hint {
  color: var(--text-muted, #94a3b8);
  font-size: 0.78rem;
}

.category-budgets-card {
  padding: 1.05rem;
}

.category-budgets-caption {
  margin: 0.7rem 0 0;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.85rem;
}

.category-budgets-empty {
  margin-top: 0.9rem;
  padding: 1.2rem;
  text-align: center;
  color: var(--text-soft, #b3bbc4);
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.025);
  border: 1px solid rgba(var(--tint-rgb), 0.05);
}

.category-budgets-list {
  display: grid;
  gap: 0.65rem;
  margin-top: 0.9rem;
}

.category-budget-row {
  padding: 0.8rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.025);
  border: 1px solid rgba(var(--tint-rgb), 0.05);
  display: grid;
  gap: 0.55rem;
}

.category-budget-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.category-budget-name {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  min-width: 0;
}

.category-budget-heading h3 {
  margin: 0;
  color: var(--text, #eef1f3);
  font-size: 0.95rem;
}

.category-type-badge {
  flex-shrink: 0;
  padding: 0.15rem 0.5rem;
  border-radius: 999px;
  font-size: 0.68rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.category-type-badge.is-expense {
  background: rgba(225, 180, 180, 0.14);
  color: var(--negative-text, #e1b4b4);
}

.category-type-badge.is-revenue {
  background: rgba(143, 168, 160, 0.16);
  color: var(--positive-text, #bfe0c9);
}

.category-budget-override-badge {
  flex-shrink: 0;
  padding: 0.2rem 0.55rem;
  border-radius: 999px;
  background: rgba(143, 168, 160, 0.16);
  color: var(--positive-text);
  border: 1px solid rgba(143, 168, 160, 0.24);
  font-size: 0.72rem;
  font-weight: 600;
}

.category-budget-meta {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  color: var(--text-muted, #94a3b8);
  font-size: 0.84rem;
}

.category-budget-suggestion {
  margin: 0.35rem 0 0;
  color: var(--text-dim, #8a939d);
  font-size: 0.78rem;
}

.category-budget-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.55rem;
}

.category-budget-actions .ghost-btn {
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  border-radius: 10px;
  padding: 0.4rem 0.7rem;
  font-size: 0.78rem;
  font-weight: 600;
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--text-soft, #b3bbc4);
  cursor: pointer;
  transition: background 140ms ease, transform 140ms ease;
}

.category-budget-actions .ghost-btn:hover {
  background: rgba(var(--tint-rgb), 0.08);
  transform: translateY(-1px);
}

.budget-history-card {
  padding: 1.05rem;
}

.history-caption {
  margin: 0.6rem 0 0;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.85rem;
}

.history-totals {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.7rem;
  margin-top: 0.9rem;
}

.history-total-figure {
  display: grid;
  gap: 0.3rem;
  padding: 0.9rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.025);
  border: 1px solid rgba(var(--tint-rgb), 0.05);
}

.history-total-figure strong {
  font-size: 1.4rem;
}

.history-chart-wrap {
  height: 220px;
  margin-top: 1.1rem;
}

.history-table-wrap {
  max-height: 420px;
  overflow: auto;
  margin-top: 1.1rem;
  border-radius: 14px;
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.06));
}

.history-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.84rem;
}

.history-table th {
  position: sticky;
  top: 0;
  z-index: 1;
  background: var(--bg-elevated, rgba(20, 23, 27, 0.98));
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-size: 0.68rem;
  text-align: left;
  padding: 0.6rem 0.6rem;
  white-space: nowrap;
}

.history-table td {
  padding: 0.55rem 0.6rem;
  border-top: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
  color: var(--text, #eef1f3);
  white-space: nowrap;
}

.history-table tr:hover td {
  background: rgba(var(--tint-rgb), 0.03);
}

.history-current-row td {
  background: rgba(143, 168, 160, 0.06);
  font-weight: 600;
}

.history-provision-hint {
  display: block;
  margin-top: 0.1rem;
  color: var(--text-dim, #8a939d);
  font-size: 0.72rem;
  font-weight: 400;
  white-space: nowrap;
}

@media (max-width: 640px) {
  .history-totals {
    grid-template-columns: 1fr;
  }
}
</style>
