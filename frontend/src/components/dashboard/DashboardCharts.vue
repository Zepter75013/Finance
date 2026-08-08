<script setup>
import { computed } from 'vue'
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
  ArcElement,
  BarController,
  LineController,
  PieController,
} from 'chart.js'
import { Chart, Pie } from 'vue-chartjs'
import { usePurchasesStore } from '../../stores/purchases'

ChartJS.register(
  Title,
  Tooltip,
  Legend,
  CategoryScale,
  LinearScale,
  BarElement,
  LineElement,
  PointElement,
  ArcElement,
  BarController,
  LineController,
  PieController
)

const props = defineProps({
  selectedYear: {
    type: Number,
    required: true,
  },
})

const CATEGORY_CHART_COLORS = [
  '#d7ddd8',
  '#a9b8c6',
  '#dcbf8a',
  '#96b4a7',
  '#b19dcb',
  '#708999',
  '#c99478',
]
const MAX_VISIBLE_SLICES = 5

const store = usePurchasesStore()
const { purchases, incomes } = storeToRefs(store)

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

const monthWindow = computed(() => {
  const months = []

  for (let month = 0; month < 12; month += 1) {
    const date = new Date(props.selectedYear, month, 1)
    months.push({
      key: monthKey(date),
      label: formatMonthLabel(date),
      fullLabel: formatFullMonthLabel(date),
    })
  }

  return months
})

function formatCurrency(value) {
  return new Intl.NumberFormat('fr-FR', {
    style: 'currency',
    currency: 'EUR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(Number(value || 0))
}

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

function buildBreakdown(items, amountOf, keyOf) {
  const totals = new Map()

  for (const item of items) {
    const key = keyOf(item)?.trim() || 'Sans catégorie'
    totals.set(key, (totals.get(key) || 0) + amountOf(item))
  }

  const sorted = Array.from(totals.entries()).sort((a, b) => b[1] - a[1])
  const top = sorted.slice(0, MAX_VISIBLE_SLICES)
  const others = sorted.slice(MAX_VISIBLE_SLICES)
  const othersTotal = others.reduce((sum, [, amount]) => sum + amount, 0)

  return othersTotal > 0 ? [...top, ['Autres', othersTotal]] : top
}

function isInSelectedYear(dateStr) {
  if (!dateStr) return false
  const date = new Date(dateStr)
  return !Number.isNaN(date.getTime()) && date.getFullYear() === props.selectedYear
}

const yearPurchases = computed(() => purchases.value.filter((purchase) => isInSelectedYear(purchase.date)))
const yearIncomes = computed(() => incomes.value.filter((income) => isInSelectedYear(income.income_date)))

const categoryBreakdown = computed(() =>
  buildBreakdown(yearPurchases.value, (purchase) => Number(purchase.amount || 0), (purchase) => purchase.category)
)

const sourceBreakdown = computed(() =>
  buildBreakdown(yearIncomes.value, (income) => Number(income.amount || 0), (income) => income.source)
)

const hasCategoryData = computed(() => categoryBreakdown.value.length > 0)
const hasSourceData = computed(() => sourceBreakdown.value.length > 0)

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
          return `${context.dataset.label} : ${formatCurrency(context.raw)}`
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
      ticks: { color: '#99a5ae', callback: (value) => formatCurrency(value) },
      grid: { color: 'rgba(var(--tint-rgb), 0.055)' },
      border: { display: false },
    },
  },
}

function buildPieData(entries) {
  return {
    labels: entries.map(([label]) => label),
    datasets: [
      {
        data: entries.map(([, value]) => value),
        backgroundColor: entries.map((_, index) => CATEGORY_CHART_COLORS[index % CATEGORY_CHART_COLORS.length]),
        borderColor: 'rgba(10, 13, 18, 0.96)',
        borderWidth: 3,
      },
    ],
  }
}

const categoryChartData = computed(() => buildPieData(categoryBreakdown.value))
const sourceChartData = computed(() => buildPieData(sourceBreakdown.value))

// Effet "3D" : le même camembert est dessiné deux fois, empilé — une copie
// assombrie légèrement décalée vers le bas (le "socle") derrière la copie
// normale (la "face" du dessus). Les deux sont inclinées ensemble en CSS
// (perspective + rotateX), ce qui fait apparaître le socle comme une tranche
// épaisse sous la face avant, sans jamais toucher aux angles des parts —
// donc sans fausser les proportions représentées.
//
// Le canevas étant incliné en CSS, les coordonnées de la souris ne
// correspondent plus à ce que Chart.js attend : la légende et l'infobulle
// intégrées seraient donc mal alignées ou inertes. On les désactive ici et on
// affiche à la place une légende HTML classique (non inclinée) sous le graphique.
const pieOptions = {
  responsive: true,
  maintainAspectRatio: false,
  events: [],
  plugins: {
    legend: { display: false },
    tooltip: { enabled: false },
  },
}

const pieBaseOptions = {
  ...pieOptions,
  animation: false,
}

function buildLegendEntries(entries) {
  const total = entries.reduce((sum, [, value]) => sum + value, 0) || 1

  return entries.map(([label, value], index) => ({
    label,
    value,
    percent: Math.round((value / total) * 100),
    color: CATEGORY_CHART_COLORS[index % CATEGORY_CHART_COLORS.length],
  }))
}

const categoryLegend = computed(() => buildLegendEntries(categoryBreakdown.value))
const sourceLegend = computed(() => buildLegendEntries(sourceBreakdown.value))
</script>

<template>
  <section class="dashboard-charts">
    <article class="panel trend-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Tendance</p>
          <h2>Dépenses vs revenus ({{ props.selectedYear }})</h2>
        </div>
      </div>

      <div v-if="hasTrendData" class="chart-box chart-box-trend">
        <Chart type="bar" :data="trendChartData" :options="trendChartOptions" />
      </div>

      <div v-else class="empty-chart-state">
        <p>Pas encore assez de données pour afficher une tendance.</p>
      </div>
    </article>

    <div class="breakdown-grid">
      <article class="panel breakdown-card">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Répartition</p>
            <h2>Achats par catégorie ({{ props.selectedYear }})</h2>
          </div>
        </div>

        <template v-if="hasCategoryData">
          <div class="chart-box chart-box-pie3d">
            <div class="pie3d-stage">
              <div class="pie3d-layer pie3d-base">
                <Pie class="pie3d-canvas-base" :data="categoryChartData" :options="pieBaseOptions" />
              </div>
              <div class="pie3d-layer pie3d-top">
                <Pie class="pie3d-canvas-top" :data="categoryChartData" :options="pieOptions" />
              </div>
            </div>
          </div>

          <ul class="pie3d-legend">
            <li v-for="entry in categoryLegend" :key="entry.label">
              <span class="pie3d-legend-swatch" :style="{ background: entry.color }"></span>
              <span class="pie3d-legend-label">{{ entry.label }}</span>
              <span class="pie3d-legend-value">{{ formatCurrency(entry.value) }} · {{ entry.percent }}%</span>
            </li>
          </ul>
        </template>

        <div v-else class="empty-chart-state empty-chart-state-small">
          <p>Aucun achat enregistré.</p>
        </div>
      </article>

      <article class="panel breakdown-card">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Répartition</p>
            <h2>Revenus par source ({{ props.selectedYear }})</h2>
          </div>
        </div>

        <template v-if="hasSourceData">
          <div class="chart-box chart-box-pie3d">
            <div class="pie3d-stage">
              <div class="pie3d-layer pie3d-base">
                <Pie class="pie3d-canvas-base" :data="sourceChartData" :options="pieBaseOptions" />
              </div>
              <div class="pie3d-layer pie3d-top">
                <Pie class="pie3d-canvas-top" :data="sourceChartData" :options="pieOptions" />
              </div>
            </div>
          </div>

          <ul class="pie3d-legend">
            <li v-for="entry in sourceLegend" :key="entry.label">
              <span class="pie3d-legend-swatch" :style="{ background: entry.color }"></span>
              <span class="pie3d-legend-label">{{ entry.label }}</span>
              <span class="pie3d-legend-value">{{ formatCurrency(entry.value) }} · {{ entry.percent }}%</span>
            </li>
          </ul>
        </template>

        <div v-else class="empty-chart-state empty-chart-state-small">
          <p>Aucun revenu enregistré.</p>
        </div>
      </article>
    </div>
  </section>
</template>

<style scoped>
.dashboard-charts {
  display: grid;
  gap: 0.7rem;
}

.trend-card,
.breakdown-card {
  padding: 0.95rem;
}

.breakdown-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.7rem;
}

.chart-box {
  position: relative;
  width: 100%;
  margin-top: 0.75rem;
}

.chart-box-trend {
  height: 240px;
}

.chart-box-pie3d {
  height: 190px;
  /* Filet de sécurité : le canevas incliné ne doit jamais déborder de sa
     carte, quelle que soit la largeur disponible. */
  overflow: hidden;
  display: flex;
  justify-content: center;
}

/* Carrée, dérivée uniquement de la hauteur (fixe) — jamais de la largeur de
   la carte. Un rectangle large et bas, une fois incliné en perspective, voit
   ses coins avant déborder, et ce débordement s'aggrave avec la largeur
   disponible (repéré en testant plusieurs largeurs de carte : le débordement
   grandissait avec la largeur). En forçant un carré centré horizontalement,
   la forme inclinée ne dépend plus jamais de la largeur — donc plus jamais
   de débordement, à n'importe quelle taille d'écran.
*/
.pie3d-stage {
  position: relative;
  height: 100%;
  aspect-ratio: 1 / 1;
  max-width: 100%;
  min-width: 0;
}

.pie3d-layer {
  position: absolute;
  inset: 0;
}

/* L'inclinaison 3D est posée directement sur le <canvas> (via la classe
   transmise au composant <Pie>, dont la racine est ce canvas) — jamais sur
   son conteneur direct. Chart.js redimensionne le canvas d'après la taille
   de son conteneur : si celui-ci portait lui-même la transformation, sa
   taille mesurée deviendrait incohérente selon la largeur disponible et le
   graphique se dimensionnait mal (donc pas responsive). En n'inclinant que
   le rendu final du canvas, le conteneur reste plat et toujours mesuré
   correctement, à n'importe quelle largeur d'écran. */
.pie3d-layer canvas {
  transform-origin: 50% 50%;
}

.pie3d-canvas-top {
  transform: perspective(900px) rotateX(50deg);
}

.pie3d-canvas-base {
  transform: perspective(900px) rotateX(50deg) translateY(12px);
  filter: brightness(0.6) saturate(1.15);
  pointer-events: none;
}

.pie3d-legend {
  list-style: none;
  margin: 0.85rem 0 0;
  padding: 0;
  display: grid;
  gap: 0.4rem;
}

.pie3d-legend li {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.82rem;
}

.pie3d-legend-swatch {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

.pie3d-legend-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text, #eef1f3);
  font-weight: 600;
}

.pie3d-legend-value {
  flex-shrink: 0;
  color: var(--text-dim, #8a939d);
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

.empty-chart-state-small {
  min-height: 130px;
}

@media (max-width: 900px) {
  .breakdown-grid {
    grid-template-columns: 1fr;
  }

  /* Une seule carte par ligne en dessous de 900px : chaque carte a alors
     beaucoup plus de largeur disponible, un peu plus de hauteur évite un
     camembert écrasé par rapport à cette largeur. */
  .chart-box-pie3d {
    height: 220px;
  }
}
</style>
