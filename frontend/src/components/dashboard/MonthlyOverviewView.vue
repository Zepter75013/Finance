<script setup>
import { computed, ref } from 'vue'
import { storeToRefs } from 'pinia'
import {
  Chart as ChartJS,
  Title,
  Tooltip,
  Legend,
  CategoryScale,
  LinearScale,
  BarElement,
  LineElement,
  PointElement,
  BarController,
  LineController,
} from 'chart.js'
import { Chart } from 'vue-chartjs'

import PageHero from '../Common/PageHero.vue'
import { usePurchasesStore } from '../../stores/purchases'
import { formatCurrency } from '../../utils/format'

ChartJS.register(
  Title,
  Tooltip,
  Legend,
  CategoryScale,
  LinearScale,
  BarElement,
  LineElement,
  PointElement,
  BarController,
  LineController
)

const store = usePurchasesStore()
const { purchases, incomes, accountsList } = storeToRefs(store)

// Passe par l'action du store (et non un simple v-model direct sur le ref)
// pour que le changement de compte actif soit bien persisté en localStorage.
const activeAccountId = computed({
  get: () => store.activeAccountId,
  set: (value) => store.setActiveAccountId(value),
})

function monthKey(date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`
}

function formatMonthLabel(date) {
  const formatted = new Intl.DateTimeFormat('fr-FR', { month: 'short' }).format(date)
  return formatted.replace('.', '').charAt(0).toUpperCase() + formatted.replace('.', '').slice(1)
}

function formatFullMonthLabel(date) {
  const formatted = new Intl.DateTimeFormat('fr-FR', { month: 'long', year: 'numeric' }).format(date)
  return formatted.charAt(0).toUpperCase() + formatted.slice(1)
}

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

const monthWindow = computed(() => {
  const months = []

  for (let month = 0; month < 12; month += 1) {
    const date = new Date(selectedYear.value, month, 1)
    months.push({
      key: monthKey(date),
      label: formatMonthLabel(date),
      fullLabel: formatFullMonthLabel(date),
    })
  }

  return months
})


const trendData = computed(() => {
  const spendingByMonth = new Map()
  const incomeByMonth = new Map()

  for (const purchase of purchases.value) {
    if (!purchase?.date) continue
    const date = new Date(purchase.date)
    if (Number.isNaN(date.getTime())) continue
    const key = monthKey(date)
    spendingByMonth.set(key, (spendingByMonth.get(key) || 0) + Number(purchase.amount || 0))
  }

  for (const income of incomes.value) {
    if (!income?.income_date) continue
    const date = new Date(income.income_date)
    if (Number.isNaN(date.getTime())) continue
    const key = monthKey(date)
    incomeByMonth.set(key, (incomeByMonth.get(key) || 0) + Number(income.amount || 0))
  }

  const labels = monthWindow.value.map((month) => month.label)
  const spending = monthWindow.value.map((month) => spendingByMonth.get(month.key) || 0)
  const income = monthWindow.value.map((month) => incomeByMonth.get(month.key) || 0)
  const net = spending.map((value, index) => income[index] - value)

  return { labels, spending, income, net }
})

const hasTrendData = computed(() => {
  return trendData.value.spending.some((value) => value > 0) || trendData.value.income.some((value) => value > 0)
})

const trendChartData = computed(() => ({
  labels: trendData.value.labels,
  datasets: [
    {
      type: 'bar',
      label: 'Dépenses',
      data: trendData.value.spending,
      backgroundColor: 'rgba(141, 116, 116, 0.55)',
      borderColor: 'rgb(141, 116, 116)',
      borderRadius: 8,
      order: 2,
    },
    {
      type: 'bar',
      label: 'Revenus',
      data: trendData.value.income,
      backgroundColor: 'rgba(129, 155, 141, 0.55)',
      borderColor: 'rgb(129, 155, 141)',
      borderRadius: 8,
      order: 2,
    },
    {
      type: 'line',
      label: 'Solde net',
      data: trendData.value.net,
      borderColor: '#d7ddd8',
      backgroundColor: '#d7ddd8',
      tension: 0.35,
      pointRadius: 3,
      pointHoverRadius: 5,
      borderWidth: 2,
      order: 1,
      yAxisID: 'y',
    },
  ],
}))

const trendChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: { mode: 'index', intersect: false },
  plugins: {
    legend: {
      position: 'bottom',
      labels: {
        color: '#c1cbd1',
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 14,
        boxWidth: 9,
        boxHeight: 9,
      },
    },
    tooltip: {
      backgroundColor: 'rgba(10, 13, 18, 0.96)',
      borderColor: 'rgba(var(--tint-rgb), 0.08)',
      borderWidth: 1,
      padding: 12,
      callbacks: {
        title(items) {
          const index = items[0]?.dataIndex
          return monthWindow.value[index]?.fullLabel ?? items[0]?.label ?? ''
        },
        label(context) {
          return `${context.dataset.label} : ${formatCurrency(context.raw, { decimals: 0 })}`
        },
        labelColor(context) {
          const color = context.dataset.borderColor || context.dataset.backgroundColor
          return {
            borderColor: color,
            backgroundColor: color,
          }
        },
      },
    },
  },
  scales: {
    x: {
      ticks: { color: '#99a5ae', maxRotation: 0, autoSkip: true, maxTicksLimit: 12 },
      grid: { display: false },
      border: { display: false },
    },
    y: {
      beginAtZero: true,
      grace: '15%',
      ticks: { color: '#99a5ae', callback: (value) => formatCurrency(value, { decimals: 0 }) },
      grid: { color: 'rgba(var(--tint-rgb), 0.055)' },
      border: { display: false },
    },
  },
}

// Tableau mois par mois — même complément que le graphique, mais avec des
// montants exacts en chiffres (le graphique donne la tendance d'un coup
// d'œil, ce tableau sert à vérifier précisément un mois donné).
const monthlyRows = computed(() =>
  monthWindow.value.map((month, index) => ({
    key: month.key,
    label: month.fullLabel,
    income: trendData.value.income[index],
    expense: trendData.value.spending[index],
    net: trendData.value.net[index],
  }))
)

const yearIncomeTotal = computed(() => trendData.value.income.reduce((sum, value) => sum + value, 0))
const yearExpenseTotal = computed(() => trendData.value.spending.reduce((sum, value) => sum + value, 0))
const yearNetTotal = computed(() => yearIncomeTotal.value - yearExpenseTotal.value)
</script>

<template>
  <main class="monthly-overview-view">
    <PageHero
      eyebrow="Vue mensuelle"
      title="Revenus & Dépenses"
      description="Le détail mois par mois, du 1er au dernier jour de chaque mois."
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

    <section class="monthly-stats-grid">
      <article class="panel stat-card">
        <span>Revenus ({{ selectedYear }})</span>
        <strong>{{ formatCurrency(yearIncomeTotal) }}</strong>
        <p>Total de l’année</p>
      </article>

      <article class="panel stat-card">
        <span>Dépenses ({{ selectedYear }})</span>
        <strong>{{ formatCurrency(yearExpenseTotal) }}</strong>
        <p>Total de l’année</p>
      </article>

      <article class="panel stat-card">
        <span>Solde net ({{ selectedYear }})</span>
        <strong>{{ formatCurrency(yearNetTotal) }}</strong>
        <p>{{ yearNetTotal >= 0 ? 'Solde positif' : 'Solde négatif' }}</p>
      </article>
    </section>

    <article class="panel trend-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Tendance</p>
          <h2>Dépenses vs revenus ({{ selectedYear }})</h2>
        </div>
      </div>

      <div v-if="hasTrendData" class="chart-box chart-box-trend">
        <Chart type="bar" :data="trendChartData" :options="trendChartOptions" />
      </div>

      <div v-else class="empty-chart-state">
        <p>Pas encore assez de données pour afficher une tendance.</p>
      </div>
    </article>

    <article v-if="hasTrendData" class="panel monthly-table-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Détail</p>
          <h2>Par mois ({{ selectedYear }})</h2>
        </div>
      </div>

      <div class="monthly-table-wrapper">
        <table class="monthly-table">
          <thead>
            <tr>
              <th>Mois</th>
              <th>Revenus</th>
              <th>Dépenses</th>
              <th>Solde</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in monthlyRows" :key="row.key">
              <td>{{ row.label }}</td>
              <td class="cell-income">{{ formatCurrency(row.income, { decimals: 0 }) }}</td>
              <td class="cell-expense">{{ formatCurrency(row.expense, { decimals: 0 }) }}</td>
              <td :class="row.net >= 0 ? 'cell-net-positive' : 'cell-net-negative'">
                {{ formatCurrency(row.net, { decimals: 0 }) }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </article>
  </main>
</template>

<style scoped>
.monthly-overview-view {
  display: grid;
  gap: 0.9rem;
}

.monthly-stats-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.7rem;
}

@media (max-width: 720px) {
  .monthly-stats-grid {
    grid-template-columns: 1fr;
  }
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

.trend-card,
.monthly-table-card {
  padding: 0.95rem;
}

.chart-box {
  position: relative;
  width: 100%;
  margin-top: 0.75rem;
}

.chart-box-trend {
  height: 320px;
}

.empty-chart-state {
  min-height: 170px;
  margin-top: 0.75rem;
  border-radius: 16px;
  display: grid;
  place-items: center;
  text-align: center;
  color: var(--text-soft, #b3bbc4);
  background: rgba(var(--tint-rgb), 0.022);
  border: 1px dashed rgba(var(--tint-rgb), 0.08);
  padding: 0.9rem;
}

.monthly-table-wrapper {
  margin-top: 0.75rem;
  overflow-x: auto;
}

.monthly-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.9rem;
}

.monthly-table th {
  text-align: left;
  padding: 0.6rem 0.75rem;
  color: var(--text-dim, #8a939d);
  font-weight: 600;
  font-size: 0.78rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  border-bottom: 1px solid rgba(var(--tint-rgb), 0.08);
}

.monthly-table td {
  padding: 0.65rem 0.75rem;
  border-bottom: 1px solid rgba(var(--tint-rgb), 0.045);
  color: var(--text, #eef1f3);
}

.monthly-table tbody tr:last-child td {
  border-bottom: none;
}

.cell-income {
  color: var(--positive-text, #81a99a);
}

.cell-expense {
  color: var(--text-soft, #b3bbc4);
}

.cell-net-positive {
  font-weight: 700;
  color: var(--positive-text, #81a99a);
}

.cell-net-negative {
  font-weight: 700;
  color: var(--negative-text, #e08585);
}

@media (max-width: 720px) {
  .chart-box-trend {
    height: 260px;
  }
}
</style>
