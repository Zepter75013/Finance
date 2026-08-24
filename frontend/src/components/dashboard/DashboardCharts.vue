<script setup>
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { Chart as ChartJS, Title, Tooltip, Legend, ArcElement, PieController } from 'chart.js'
import { Pie } from 'vue-chartjs'
import { usePurchasesStore } from '../../stores/purchases'
import { formatCurrency } from '../../utils/format'
import { signTransferLeg } from '../../utils/realBalance'

ChartJS.register(Title, Tooltip, Legend, ArcElement, PieController)

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
const { purchases, incomes, transfers } = storeToRefs(store)


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

// Un compte comme un Livret n'a jamais d'achat/revenu, seulement des
// virements — sans ça, ses deux camemberts resteraient vides. Un virement
// sortant rejoint la répartition "achats" sous le libellé « Virement », un
// virement entrant la répartition "revenus" sous « Virement interne » (même
// logique que les cartes du haut, voir DashboardView.vue).
const yearTransferLegs = computed(() =>
  transfers.value.map((t) => signTransferLeg(t, store.activeAccountId)).filter((leg) => isInSelectedYear(leg.date))
)

const categoryBreakdown = computed(() => {
  const entries = [
    ...yearPurchases.value.map((purchase) => ({ amount: Number(purchase.amount || 0), key: purchase.category })),
    ...yearTransferLegs.value
      .filter((leg) => leg.isOutgoing)
      .map((leg) => ({ amount: Math.abs(leg.amount), key: 'Virement' })),
  ]
  return buildBreakdown(entries, (entry) => entry.amount, (entry) => entry.key)
})

const sourceBreakdown = computed(() => {
  const entries = [
    ...yearIncomes.value.map((income) => ({ amount: Number(income.amount || 0), key: income.source })),
    ...yearTransferLegs.value
      .filter((leg) => !leg.isOutgoing)
      .map((leg) => ({ amount: leg.amount, key: 'Virement interne' })),
  ]
  return buildBreakdown(entries, (entry) => entry.amount, (entry) => entry.key)
})

const hasCategoryData = computed(() => categoryBreakdown.value.length > 0)
const hasSourceData = computed(() => sourceBreakdown.value.length > 0)

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
              <span class="pie3d-legend-value">{{ formatCurrency(entry.value, { decimals: 0 }) }} · {{ entry.percent }}%</span>
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
              <span class="pie3d-legend-value">{{ formatCurrency(entry.value, { decimals: 0 }) }} · {{ entry.percent }}%</span>
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
