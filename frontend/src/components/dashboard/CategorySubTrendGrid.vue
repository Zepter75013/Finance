<script setup>
import { computed, ref } from 'vue'
import { storeToRefs } from 'pinia'
import {
  Chart as ChartJS,
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
import ChartDataLabels from 'chartjs-plugin-datalabels'
import { usePurchasesStore } from '../../stores/purchases'
import { formatCurrency } from '../../utils/format'

ChartJS.register(
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

// Même palette catégorielle que le reste du dashboard (donuts, tendance).
const SERIES_COLORS = ['#d7ddd8', '#a9b8c6', '#dcbf8a', '#96b4a7', '#b19dcb', '#708999', '#c99478']
const MAX_VISIBLE_SUBCATEGORIES = 5
const NO_SUBCATEGORY_LABEL = 'Sans sous-catégorie'
const OTHERS_LABEL = 'Autres'

const props = defineProps({
  selectedYear: {
    type: Number,
    required: true,
  },
})

const store = usePurchasesStore()
const { purchases, categoriesList } = storeToRefs(store)


function monthLabel(monthIndex, format) {
  const formatted = new Intl.DateTimeFormat('fr-FR', { month: format }).format(
    new Date(props.selectedYear, monthIndex, 1)
  )
  const clean = formatted.replace('.', '')
  return clean.charAt(0).toUpperCase() + clean.slice(1)
}

const monthLabels = computed(() => Array.from({ length: 12 }, (_, i) => monthLabel(i, 'short')))
const monthFullLabels = computed(() => Array.from({ length: 12 }, (_, i) => monthLabel(i, 'long')))

const yearPurchasesByCategory = computed(() => {
  const map = new Map()

  for (const purchase of purchases.value) {
    if (!purchase.date) continue

    const date = new Date(purchase.date)
    if (Number.isNaN(date.getTime()) || date.getFullYear() !== props.selectedYear) continue

    const list = map.get(purchase.categoryId) || []
    list.push(purchase)
    map.set(purchase.categoryId, list)
  }

  return map
})

// Nombre de mois de l'année sélectionnée où il existe au moins un achat (toutes
// catégories confondues) — même logique que la moyenne de l'écran Budget, pour
// que les deux moyennes restent cohérentes entre elles. Le mois en cours est
// exclu car incomplet : sinon, dès qu'une autre catégorie a une dépense ce
// mois-ci, il gonflerait le dénominateur et diluerait la moyenne — même si
// CETTE catégorie n'a encore rien ce mois-ci.
const currentMonthKey = new Date().toISOString().slice(0, 7)

const monthsTrackedInYear = computed(() => {
  const months = new Set()

  for (const purchase of purchases.value) {
    if (!purchase.date || purchase.date.slice(0, 7) === currentMonthKey) continue

    const date = new Date(purchase.date)
    if (Number.isNaN(date.getTime()) || date.getFullYear() !== props.selectedYear) continue

    months.add(date.getMonth())
  }

  return months.size
})

function buildCategoryCard(category) {
  const categoryPurchases = yearPurchasesByCategory.value.get(category.id) || []
  if (categoryPurchases.length === 0) return null

  const totalsBySub = new Map()
  for (const purchase of categoryPurchases) {
    const key = purchase.subCategory?.trim() || NO_SUBCATEGORY_LABEL
    totalsBySub.set(key, (totalsBySub.get(key) || 0) + Number(purchase.amount || 0))
  }

  const sortedSubs = Array.from(totalsBySub.entries()).sort((a, b) => b[1] - a[1])
  const topSubNames = sortedSubs.slice(0, MAX_VISIBLE_SUBCATEGORIES).map(([name]) => name)
  const hasOthers = sortedSubs.length > MAX_VISIBLE_SUBCATEGORIES
  const seriesNames = hasOthers ? [...topSubNames, OTHERS_LABEL] : topSubNames

  const nameIndex = new Map(seriesNames.map((name, index) => [name, index]))
  const monthlyTotals = seriesNames.map(() => Array(12).fill(0))

  for (const purchase of categoryPurchases) {
    const date = new Date(purchase.date)
    const key = purchase.subCategory?.trim() || NO_SUBCATEGORY_LABEL
    const seriesIndex = nameIndex.has(key) ? nameIndex.get(key) : nameIndex.get(OTHERS_LABEL)
    if (seriesIndex === undefined) continue
    monthlyTotals[seriesIndex][date.getMonth()] += Number(purchase.amount || 0)
  }

  const yearTotal = categoryPurchases.reduce((sum, purchase) => sum + Number(purchase.amount || 0), 0)

  return {
    id: category.id,
    name: category.name,
    seriesNames,
    monthlyTotals,
    yearTotal,
  }
}

const categoryCards = computed(() =>
  categoriesList.value
    .filter((category) => category.type === 'achat')
    .map(buildCategoryCard)
    .filter(Boolean)
    .sort((a, b) => b.yearTotal - a.yearTotal)
)

function stackedBarDatasets(card) {
  return card.seriesNames.map((name, index) => ({
    label: name,
    data: card.monthlyTotals[index],
    backgroundColor: SERIES_COLORS[index % SERIES_COLORS.length],
    borderRadius: 2,
    stack: 'total',
    // Ordre plus élevé que les traits de moyenne (order: 0) pour que ceux-ci
    // se dessinent PAR-DESSUS les barres et restent entièrement visibles.
    order: 1,
  }))
}

function sparklineData(card) {
  return {
    labels: monthLabels.value,
    datasets: stackedBarDatasets(card),
  }
}

const sparklineOptions = {
  responsive: true,
  maintainAspectRatio: false,
  animation: false,
  plugins: {
    legend: { display: false },
    tooltip: { enabled: false },
  },
  scales: {
    x: { display: false, stacked: true },
    y: { display: false, stacked: true },
  },
}

const activeCard = ref(null)

function openCard(card) {
  activeCard.value = card
}

function closeModal() {
  activeCard.value = null
}

const AVERAGE_LINE_ID = 'average-line'
const AVERAGE_LINE_COLOR = '#3b82f6'

function linearGradientColor(context, colorStart, colorEnd, fallback) {
  const { chart } = context
  if (!chart.chartArea) return fallback
  const gradient = chart.ctx.createLinearGradient(chart.chartArea.left, 0, chart.chartArea.right, 0)
  gradient.addColorStop(0, colorStart)
  gradient.addColorStop(1, colorEnd)
  return gradient
}

// Même méthode que la moyenne de l'écran Budget : total ÷ nombre de mois
// réellement suivis (tous achats confondus, sans filtre de pointage) — pas
// une division fixe par 12, qui sous-estimerait la moyenne tant que l'année
// n'est pas terminée.
const activeCardAverages = computed(() => {
  const card = activeCard.value
  if (!card) return null

  const average = monthsTrackedInYear.value > 0 ? card.yearTotal / monthsTrackedInYear.value : 0

  return { average }
})

const modalChartData = computed(() => {
  if (!activeCard.value || !activeCardAverages.value) return null

  const card = activeCard.value
  const { average } = activeCardAverages.value

  return {
    labels: monthLabels.value,
    datasets: [
      ...stackedBarDatasets(card),
      {
        id: AVERAGE_LINE_ID,
        type: 'line',
        label: 'Moyenne mensuelle',
        data: Array(12).fill(average),
        borderWidth: 2.5,
        borderCapStyle: 'round',
        pointRadius: 0,
        order: 0,
        // Groupe d'empilement dédié : sans ça, l'axe "stacked" cumule cette
        // ligne avec les autres jeux de données au lieu d'afficher sa vraie valeur.
        stack: AVERAGE_LINE_ID,
        borderColor: (context) => linearGradientColor(context, '#93c5fd', '#2563eb', AVERAGE_LINE_COLOR),
      },
    ],
  }
})

const REFERENCE_LINES = [{ id: AVERAGE_LINE_ID, shadow: 'rgba(37, 99, 235, 0.3)' }]

function referenceLineConfig(datasetId) {
  return REFERENCE_LINES.find((line) => line.id === datasetId)
}

// Glow discret sous chaque trait de référence + pastille flottante indiquant sa
// valeur, pour qu'ils se distinguent des barres sans surcharger la légende.
const averageLineGlowPlugin = {
  id: 'averageLineGlow',
  beforeDatasetDraw(chart, args) {
    const config = referenceLineConfig(chart.data.datasets[args.index]?.id)
    if (!config) return
    chart.ctx.save()
    chart.ctx.shadowColor = config.shadow
    chart.ctx.shadowBlur = 10
    chart.ctx.shadowOffsetY = 2
  },
  afterDatasetDraw(chart, args) {
    if (!referenceLineConfig(chart.data.datasets[args.index]?.id)) return
    chart.ctx.restore()
  },
}

const modalChartPlugins = [averageLineGlowPlugin, ChartDataLabels]

function topBarDatasetIndex(chart) {
  const { datasets } = chart.data
  for (let i = datasets.length - 1; i >= 0; i -= 1) {
    if (datasets[i].stack === 'total') return i
  }
  return -1
}

function monthTotal(chart, monthIndex) {
  return chart.data.datasets
    .filter((dataset) => dataset.stack === 'total')
    .reduce((sum, dataset) => sum + (dataset.data[monthIndex] || 0), 0)
}

const modalChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: { mode: 'nearest', intersect: true },
  plugins: {
    datalabels: {
      anchor: 'end',
      align: 'top',
      offset: 6,
      color: '#ffffff',
      backgroundColor: 'rgba(51, 65, 85, 0.92)',
      borderRadius: 10,
      padding: { top: 3, bottom: 3, left: 7, right: 7 },
      font: { weight: '700', size: 11 },
      display(context) {
        if (context.datasetIndex !== topBarDatasetIndex(context.chart)) return false
        return monthTotal(context.chart, context.dataIndex) > 0
      },
      formatter(_value, context) {
        return formatCurrency(monthTotal(context.chart, context.dataIndex))
      },
    },
    legend: {
      position: 'bottom',
      labels: {
        color: '#c1cbd1',
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 12,
        boxWidth: 9,
        boxHeight: 9,
        // Les moyennes ont déjà leur propre pastille flottante sur le graphique —
        // pas besoin de les dupliquer dans la légende.
        filter: (legendItem, data) => data.datasets[legendItem.datasetIndex]?.id !== AVERAGE_LINE_ID,
      },
    },
    tooltip: {
      backgroundColor: 'rgba(10, 13, 18, 0.96)',
      borderColor: 'rgba(var(--tint-rgb), 0.08)',
      borderWidth: 1,
      padding: 12,
      callbacks: {
        title(items) {
          return monthFullLabels.value[items[0]?.dataIndex] ?? ''
        },
        label(context) {
          return `${context.dataset.label} : ${formatCurrency(context.raw)}`
        },
      },
    },
  },
  // Marge en haut du graphique : sans elle, la bulle du total mensuel des
  // barres les plus hautes se retrouve coupée par le bord du canevas.
  layout: {
    padding: { top: 26 },
  },
  scales: {
    x: {
      stacked: true,
      ticks: { color: '#99a5ae' },
      grid: { display: false },
      border: { display: false },
    },
    y: {
      stacked: true,
      beginAtZero: true,
      grace: '12%',
      ticks: { color: '#99a5ae', callback: (value) => formatCurrency(value) },
      grid: { color: 'rgba(var(--tint-rgb), 0.055)' },
      border: { display: false },
    },
  },
}
</script>

<template>
  <section v-if="categoryCards.length" class="panel category-trend-card">
    <div class="panel-header">
      <div>
        <p class="eyebrow">Détail par catégorie</p>
        <h2>Évolution des sous-catégories ({{ props.selectedYear }})</h2>
      </div>
    </div>

    <div class="category-trend-grid">
      <button
        v-for="card in categoryCards"
        :key="card.id"
        class="category-trend-tile"
        type="button"
        @click="openCard(card)"
      >
        <div class="category-trend-tile-header">
          <span class="category-trend-name">{{ card.name }}</span>
          <span class="category-trend-total">{{ formatCurrency(card.yearTotal) }}</span>
        </div>
        <div class="category-trend-sparkline">
          <Chart type="bar" :data="sparklineData(card)" :options="sparklineOptions" />
        </div>
      </button>
    </div>
  </section>

  <div v-if="activeCard" class="modal-overlay" @click.self="closeModal">
    <section class="modal-card category-trend-modal" role="dialog" aria-modal="true">
      <div class="modal-header category-trend-modal-header">
        <div>
          <p class="eyebrow">{{ props.selectedYear }}</p>
          <div class="category-trend-modal-title">
            <h2>{{ activeCard.name }}</h2>
            <div v-if="activeCardAverages" class="average-chips">
              <span class="average-chip average-chip-blue">
                Moyenne · {{ formatCurrency(activeCardAverages.average) }}
              </span>
            </div>
          </div>
        </div>
        <button class="ghost-btn" type="button" @click="closeModal">Fermer</button>
      </div>

      <div class="category-trend-modal-chart">
        <Chart type="bar" :data="modalChartData" :options="modalChartOptions" :plugins="modalChartPlugins" />
      </div>
    </section>
  </div>
</template>

<style scoped>
.category-trend-card {
  padding: 0.95rem;
}

.category-trend-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 0.7rem;
  margin-top: 0.75rem;
}

.category-trend-tile {
  display: grid;
  gap: 0.5rem;
  padding: 0.85rem 0.9rem;
  border: 1px solid rgba(var(--tint-rgb), 0.06);
  border-radius: 16px;
  background: rgba(var(--tint-rgb), 0.022);
  text-align: left;
  cursor: pointer;
  overflow: hidden;
  transition: transform 140ms ease, background 140ms ease, border-color 140ms ease;
}

.category-trend-tile:hover {
  transform: translateY(-2px);
  background: rgba(var(--tint-rgb), 0.045);
  border-color: rgba(var(--tint-rgb), 0.1);
}

.category-trend-tile-header {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.category-trend-name {
  color: var(--text, #eef1f3);
  font-weight: 600;
  font-size: 0.88rem;
  line-height: 1.25;
}

.category-trend-total {
  color: var(--text-soft, #b3bbc4);
  font-size: 0.8rem;
}

.category-trend-sparkline {
  position: relative;
  height: 56px;
}

.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 80;
  display: grid;
  place-items: center;
  padding: 1.5rem;
  background: var(--modal-overlay);
  backdrop-filter: blur(6px);
  animation: overlay-fade-in 160ms ease;
}

.modal-card {
  background: var(--modal-bg);
  border: 1px solid rgba(var(--tint-rgb), 0.06);
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.34);
  border-radius: 24px;
}

.category-trend-modal {
  width: min(100%, 720px);
  padding: 1.35rem;
  animation: modal-pop-in 180ms ease;
}

.category-trend-modal-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.5rem;
}

.category-trend-modal-title {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.6rem;
  margin-top: 0.3rem;
}

.category-trend-modal-title h2 {
  margin: 0;
  font-size: 1.35rem;
  color: var(--text, #eef1f3);
}

.average-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}

.average-chip {
  border-radius: 999px;
  padding: 0.28rem 0.7rem;
  font-size: 0.76rem;
  font-weight: 700;
  color: #ffffff;
  white-space: nowrap;
}

.average-chip-blue {
  background: linear-gradient(135deg, #60a5fa, #2563eb);
}

.category-trend-modal-chart {
  height: 360px;
  margin-top: 0.75rem;
}

.ghost-btn {
  border: none;
  border-radius: 12px;
  padding: 0.55rem 1rem;
  background: rgba(var(--tint-rgb), 0.06);
  color: var(--text);
  font-weight: 600;
  cursor: pointer;
  transition: background 140ms ease, transform 140ms ease;
}

.ghost-btn:hover {
  background: rgba(var(--tint-rgb), 0.1);
  transform: translateY(-1px);
}

@keyframes overlay-fade-in {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes modal-pop-in {
  from {
    opacity: 0;
    transform: translateY(8px) scale(0.985);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@media (max-width: 640px) {
  .category-trend-grid {
    grid-template-columns: 1fr;
  }
}

@media (prefers-reduced-motion: reduce) {
  .modal-overlay,
  .category-trend-modal,
  .category-trend-tile,
  .ghost-btn {
    animation: none;
    transition: none;
  }
}
</style>
