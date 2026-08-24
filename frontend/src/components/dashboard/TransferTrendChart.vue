<script setup>
import { computed, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { Chart as ChartJS, LineElement, PointElement, LinearScale, CategoryScale, Tooltip, Legend, Filler } from 'chart.js'
import { Line } from 'vue-chartjs'
import { usePurchasesStore } from '../../stores/purchases'
import { formatCurrency } from '../../utils/format'
import { signTransferLeg } from '../../utils/realBalance'

ChartJS.register(LineElement, PointElement, LinearScale, CategoryScale, Tooltip, Legend, Filler)

const store = usePurchasesStore()
const { transfers } = storeToRefs(store)

// 'month' : un point par jour du mois sélectionné.
// 'year'  : un point par mois de l'année sélectionnée (la vue par défaut,
//           celle qui donne son nom au widget — "mois par mois").
// 'range' : bornes libres — un point par jour si la période fait moins de
//           ~2 mois, sinon un point par mois (sinon un an de données
//           journalières serait illisible).
const viewMode = ref('year')

const today = new Date()
const currentYear = today.getFullYear()

const selectedYear = ref(currentYear)
const selectedMonth = ref(today.getMonth())

function pad(n) {
  return String(n).padStart(2, '0')
}

const rangeStart = ref(`${currentYear}-01-01`)
const rangeEnd = ref(today.toISOString().slice(0, 10))

function daysInMonth(year, monthIndex) {
  return new Date(year, monthIndex + 1, 0).getDate()
}

function monthShortLabel(year, monthIndex) {
  const formatted = new Intl.DateTimeFormat('fr-FR', { month: 'short' }).format(new Date(year, monthIndex, 1))
  const clean = formatted.replace('.', '')
  return clean.charAt(0).toUpperCase() + clean.slice(1)
}

function monthFullLabel(year, monthIndex) {
  const formatted = new Intl.DateTimeFormat('fr-FR', { month: 'long' }).format(new Date(year, monthIndex, 1))
  return formatted.charAt(0).toUpperCase() + formatted.slice(1)
}

const monthPeriodLabel = computed(() => `${monthFullLabel(selectedYear.value, selectedMonth.value)} ${selectedYear.value}`)

// Sélecteur direct (en plus des flèches ‹ › pour naviguer pas à pas) — un
// <input type="month"> ouvre le calendrier natif du navigateur, plus rapide
// que de cliquer plusieurs fois pour atteindre un mois éloigné.
const monthInputValue = computed({
  get: () => `${selectedYear.value}-${pad(selectedMonth.value + 1)}`,
  set: (value) => {
    const [year, month] = String(value || '').split('-').map(Number)
    if (!year || !month) return
    selectedYear.value = year
    selectedMonth.value = month - 1
  },
})

function goToPreviousMonth() {
  if (selectedMonth.value === 0) {
    selectedMonth.value = 11
    selectedYear.value -= 1
  } else {
    selectedMonth.value -= 1
  }
}

function goToNextMonth() {
  if (selectedMonth.value === 11) {
    selectedMonth.value = 0
    selectedYear.value += 1
  } else {
    selectedMonth.value += 1
  }
}

// Nombre de jours entre les deux bornes de la période libre — détermine si
// on regroupe par jour ou par mois pour ce mode.
const rangeSpanDays = computed(() => {
  const start = new Date(rangeStart.value)
  const end = new Date(rangeEnd.value)
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) return null
  return Math.round((end - start) / 86_400_000)
})

const rangeGroupByDay = computed(() => rangeSpanDays.value !== null && rangeSpanDays.value <= 62)

// La liste des points affichés (clé + libellé), calculée une fois par mode —
// bucketKeyOf (plus bas) doit produire exactement les mêmes clés à partir
// d'une date de virement pour que l'agrégation retombe sur ces points.
const buckets = computed(() => {
  if (viewMode.value === 'year') {
    return Array.from({ length: 12 }, (_, i) => ({
      key: `${selectedYear.value}-${pad(i + 1)}`,
      label: monthShortLabel(selectedYear.value, i),
    }))
  }

  if (viewMode.value === 'month') {
    const total = daysInMonth(selectedYear.value, selectedMonth.value)
    return Array.from({ length: total }, (_, i) => ({
      key: `${selectedYear.value}-${pad(selectedMonth.value + 1)}-${pad(i + 1)}`,
      label: String(i + 1),
    }))
  }

  // range
  const start = new Date(rangeStart.value)
  const end = new Date(rangeEnd.value)
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime()) || start > end) return []

  if (rangeGroupByDay.value) {
    const list = []
    const cursor = new Date(start)
    while (cursor <= end) {
      list.push({
        key: cursor.toISOString().slice(0, 10),
        label: `${pad(cursor.getDate())}/${pad(cursor.getMonth() + 1)}`,
      })
      cursor.setDate(cursor.getDate() + 1)
    }
    return list
  }

  const list = []
  const cursor = new Date(start.getFullYear(), start.getMonth(), 1)
  const endMonth = new Date(end.getFullYear(), end.getMonth(), 1)
  while (cursor <= endMonth) {
    list.push({
      key: `${cursor.getFullYear()}-${pad(cursor.getMonth() + 1)}`,
      label: `${monthShortLabel(cursor.getFullYear(), cursor.getMonth())} ${String(cursor.getFullYear()).slice(2)}`,
    })
    cursor.setMonth(cursor.getMonth() + 1)
  }
  return list
})

// Même granularité que buckets ci-dessus, appliquée à la date d'un virement
// — les clés qui n'apparaissent dans aucun bucket (hors période) sont
// simplement ignorées à la lecture, pas besoin de filtrer en amont.
function bucketKeyOf(dateStr) {
  if (!dateStr) return null
  if (viewMode.value === 'month') return dateStr.slice(0, 10)
  if (viewMode.value === 'range') return rangeGroupByDay.value ? dateStr.slice(0, 10) : dateStr.slice(0, 7)
  return dateStr.slice(0, 7)
}

const allLegs = computed(() => transfers.value.map((t) => signTransferLeg(t, store.activeAccountId)))

const bucketedTotals = computed(() => {
  const creditMap = new Map()
  const debitMap = new Map()

  for (const leg of allLegs.value) {
    const key = bucketKeyOf(leg.date)
    if (!key) continue

    if (leg.isOutgoing) {
      debitMap.set(key, (debitMap.get(key) || 0) + Math.abs(leg.amount))
    } else {
      creditMap.set(key, (creditMap.get(key) || 0) + leg.amount)
    }
  }

  return { creditMap, debitMap }
})

const chartData = computed(() => {
  const { creditMap, debitMap } = bucketedTotals.value

  return {
    labels: buckets.value.map((b) => b.label),
    datasets: [
      {
        label: 'Crédit',
        data: buckets.value.map((b) => creditMap.get(b.key) || 0),
        borderColor: '#5ecb8f',
        backgroundColor: 'rgba(94, 203, 143, 0.15)',
        tension: 0.3,
        fill: true,
        pointRadius: 3,
        pointHoverRadius: 5,
      },
      {
        label: 'Débit',
        data: buckets.value.map((b) => debitMap.get(b.key) || 0),
        borderColor: '#e1786a',
        backgroundColor: 'rgba(225, 120, 106, 0.12)',
        tension: 0.3,
        fill: true,
        pointRadius: 3,
        pointHoverRadius: 5,
      },
    ],
  }
})

const hasAnyData = computed(() => chartData.value.datasets.some((dataset) => dataset.data.some((value) => value > 0)))

const lineOptions = {
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
        boxWidth: 9,
        boxHeight: 9,
        padding: 14,
      },
    },
    tooltip: {
      backgroundColor: 'rgba(10, 13, 18, 0.96)',
      borderColor: 'rgba(var(--tint-rgb), 0.08)',
      borderWidth: 1,
      padding: 12,
      callbacks: {
        label(context) {
          return `${context.dataset.label} : ${formatCurrency(context.raw)}`
        },
      },
    },
  },
  scales: {
    x: {
      ticks: { color: '#99a5ae' },
      grid: { display: false },
      border: { display: false },
    },
    y: {
      beginAtZero: true,
      ticks: { color: '#99a5ae', callback: (value) => formatCurrency(value, { decimals: 0 }) },
      grid: { color: 'rgba(var(--tint-rgb), 0.055)' },
      border: { display: false },
    },
  },
}
</script>

<template>
  <section class="panel transfer-trend-card">
    <div class="panel-header transfer-trend-header">
      <div>
        <p class="eyebrow">Suivi</p>
        <h2>Crédit / Débit</h2>
      </div>

      <div class="transfer-trend-controls">
        <div class="view-mode-tabs">
          <button
            class="view-mode-tab"
            type="button"
            :class="{ 'is-active': viewMode === 'month' }"
            @click="viewMode = 'month'"
          >
            Mois
          </button>
          <button
            class="view-mode-tab"
            type="button"
            :class="{ 'is-active': viewMode === 'year' }"
            @click="viewMode = 'year'"
          >
            Année
          </button>
          <button
            class="view-mode-tab"
            type="button"
            :class="{ 'is-active': viewMode === 'range' }"
            @click="viewMode = 'range'"
          >
            Période
          </button>
        </div>

        <div v-if="viewMode === 'year'" class="period-switcher">
          <button class="ghost-btn period-switcher-btn" type="button" aria-label="Année précédente" @click="selectedYear -= 1">‹</button>
          <span class="period-switcher-value">{{ selectedYear }}</span>
          <button class="ghost-btn period-switcher-btn" type="button" aria-label="Année suivante" @click="selectedYear += 1">›</button>
        </div>

        <div v-else-if="viewMode === 'month'" class="period-switcher">
          <button class="ghost-btn period-switcher-btn" type="button" aria-label="Mois précédent" @click="goToPreviousMonth">‹</button>
          <input v-model="monthInputValue" type="month" class="month-input" :aria-label="monthPeriodLabel" />
          <button class="ghost-btn period-switcher-btn" type="button" aria-label="Mois suivant" @click="goToNextMonth">›</button>
        </div>

        <div v-else class="range-inputs">
          <input v-model="rangeStart" type="date" class="range-input" />
          <span class="range-arrow">→</span>
          <input v-model="rangeEnd" type="date" class="range-input" />
        </div>
      </div>
    </div>

    <div v-if="hasAnyData" class="chart-box-line">
      <Line :data="chartData" :options="lineOptions" />
    </div>
    <div v-else class="empty-chart-state">
      <p>Aucun virement sur cette période.</p>
    </div>
  </section>
</template>

<style scoped>
.transfer-trend-card {
  padding: 0.95rem;
}

.transfer-trend-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 0.7rem;
}

.transfer-trend-controls {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.6rem;
}

.view-mode-tabs {
  display: flex;
  gap: 0.35rem;
  padding: 0.25rem;
  border-radius: 999px;
  background: rgba(var(--tint-rgb), 0.04);
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
}

.view-mode-tab {
  border: none;
  border-radius: 999px;
  padding: 0.4rem 0.8rem;
  background: transparent;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 140ms ease, color 140ms ease;
}

.view-mode-tab:hover {
  color: var(--text, #eef1f3);
}

.view-mode-tab.is-active {
  background: rgba(219, 230, 223, 0.14);
  color: var(--text, #eef1f3);
}

.period-switcher {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.3rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.04);
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
}

.period-switcher-btn {
  width: 30px;
  height: 30px;
  min-width: 0;
  padding: 0;
  font-size: 1rem;
  line-height: 1;
  display: grid;
  place-items: center;
}

.period-switcher-value {
  min-width: 3.2rem;
  text-align: center;
  font-weight: 700;
  color: var(--text, #eef1f3);
  font-size: 0.88rem;
}

.range-inputs {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.range-input {
  height: 36px;
  padding: 0 0.6rem;
  border-radius: 10px;
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--text, #eef1f3);
  font-size: 0.82rem;
  outline: none;
}

.month-input {
  height: 30px;
  min-width: 9.5rem;
  padding: 0 0.5rem;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text, #eef1f3);
  font-size: 0.86rem;
  font-weight: 700;
  text-align: center;
  outline: none;
}

.range-arrow {
  color: var(--text-dim, #8a939d);
}

.ghost-btn {
  border: none;
  border-radius: 10px;
  background: rgba(var(--tint-rgb), 0.06);
  color: var(--text, #eef1f3);
  cursor: pointer;
  transition: transform 140ms ease, background 140ms ease;
}

.ghost-btn:hover {
  transform: translateY(-1px);
}

.chart-box-line {
  position: relative;
  height: 260px;
  margin-top: 1rem;
}

.empty-chart-state {
  min-height: 220px;
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

@media (max-width: 640px) {
  .transfer-trend-header {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
