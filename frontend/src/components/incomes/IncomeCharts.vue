<script setup>
import { computed, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { usePurchasesStore } from '../../stores/purchases'
import { formatCurrency } from '../../utils/format'

const store = usePurchasesStore()
const { incomes } = storeToRefs(store)

const SOURCE_CHART_COLORS = [
  '#d7ddd8',
  '#a9b8c6',
  '#dcbf8a',
  '#96b4a7',
  '#b19dcb',
  '#708999',
  '#c99478',
]

const MAX_VISIBLE_SOURCES = 6

const isCollapsed = ref(localStorage.getItem('incomeChartsCollapsed') === '1')

function toggleCollapsed() {
  isCollapsed.value = !isCollapsed.value
  localStorage.setItem('incomeChartsCollapsed', isCollapsed.value ? '1' : '0')
}

const selectedRange = ref('12m')
const selectedSource = ref('all')
const selectedYear = ref(new Date().getFullYear())
const searchQuery = ref('')
const selectedMonthIndex = ref(null)

const periodOptions = [
  { value: '30d', label: '30 jours' },
  { value: '90d', label: '90 jours' },
  { value: '12m', label: '12 mois' },
  { value: 'ytd', label: 'Cette année' },
  { value: 'all', label: 'Tout' },
]

const sourceOptions = computed(() => {
  const values = new Set()
  for (const income of incomes.value) values.add(income?.source?.trim() || 'Sans source')

  return [
    { value: 'all', label: 'Toutes les sources' },
    ...Array.from(values)
      .sort((a, b) => a.localeCompare(b, 'fr', { sensitivity: 'base' }))
      .map((source) => ({ value: source, label: source })),
  ]
})

const availableYears = computed(() => {
  const years = new Set()

  for (const income of incomes.value) {
    if (!income?.income_date) continue
    const date = new Date(income.income_date)
    if (Number.isNaN(date.getTime())) continue
    years.add(date.getFullYear())
  }

  return Array.from(years).sort((a, b) => b - a)
})

watch(
  availableYears,
  (years) => {
    if (!years.length) {
      selectedYear.value = new Date().getFullYear()
      return
    }

    if (!years.includes(selectedYear.value)) selectedYear.value = years[0]
  },
  { immediate: true }
)

const chartGranularity = computed(() =>
  selectedRange.value === '30d' || selectedRange.value === '90d' ? 'day' : 'month'
)



function formatPercent(value) {
  if (!Number.isFinite(value)) return '—'
  return `${value > 0 ? '+' : ''}${value.toFixed(1)}%`
}

function formatShortMonth(index) {
  return new Intl.DateTimeFormat('fr-FR', { month: 'short' })
    .format(new Date(2026, index, 1))
    .replace('.', '')
}

function normalizeSearchValue(value) {
  return String(value ?? '')
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .trim()
}

const filteredIncomes = computed(() => {
  let result = incomes.value

  if (selectedRange.value !== 'all') {
    const now = new Date()
    const startDate = new Date(now)

    if (selectedRange.value === '30d') startDate.setDate(now.getDate() - 30)
    else if (selectedRange.value === '90d') startDate.setDate(now.getDate() - 90)
    else if (selectedRange.value === '12m') startDate.setMonth(now.getMonth() - 12)
    else if (selectedRange.value === 'ytd') {
      startDate.setMonth(0, 1)
      startDate.setHours(0, 0, 0, 0)
    }

    result = result.filter((income) => {
      if (!income?.income_date) return false
      const date = new Date(income.income_date)
      if (Number.isNaN(date.getTime())) return false
      return date >= startDate && date <= now
    })
  }

  if (selectedSource.value !== 'all') {
    result = result.filter((income) => {
      const source = income?.source?.trim() || 'Sans source'
      return source === selectedSource.value
    })
  }

  const normalizedQuery = normalizeSearchValue(searchQuery.value)
  if (normalizedQuery) {
    result = result.filter((income) => {
      const searchableValues = [
        income?.source,
        income?.notes,
        income?.description,
        income?.label,
        income?.category,
        income?.amount,
        income?.income_date,
      ]

      return normalizeSearchValue(searchableValues.join(' ')).includes(normalizedQuery)
    })
  }

  return result
})

const annualSummary = computed(() => {
  const monthlyTotals = Array.from({ length: 12 }, (_, monthIndex) => ({
    monthIndex,
    label: formatShortMonth(monthIndex),
    total: 0,
  }))

  let total = 0
  let count = 0
  const normalizedQuery = normalizeSearchValue(searchQuery.value)

  for (const income of incomes.value) {
    if (!income?.income_date) continue

    const date = new Date(income.income_date)
    if (Number.isNaN(date.getTime())) continue
    if (date.getFullYear() !== selectedYear.value) continue

    const source = income?.source?.trim() || 'Sans source'
    if (selectedSource.value !== 'all' && source !== selectedSource.value) continue

    if (normalizedQuery) {
      const searchableValues = [
        income?.source,
        income?.notes,
        income?.description,
        income?.label,
        income?.category,
        income?.amount,
        income?.income_date,
      ]

      if (!normalizeSearchValue(searchableValues.join(' ')).includes(normalizedQuery)) continue
    }

    const amount = Number(income.amount || 0)
    total += amount
    count += 1
    monthlyTotals[date.getMonth()].total += amount
  }

  const activeMonths = monthlyTotals.filter((month) => month.total > 0)
  const averagePerActiveMonth = activeMonths.length ? total / activeMonths.length : 0
  const bestMonth = monthlyTotals.reduce(
    (best, current) => (current.total > best.total ? current : best),
    monthlyTotals[0]
  )

  const nonZeroMonths = monthlyTotals.filter((month) => month.total > 0)
  const currentMonth = nonZeroMonths.at(-1) || monthlyTotals[0]
  const referenceMonth =
    currentMonth && currentMonth.monthIndex > 0 ? monthlyTotals[currentMonth.monthIndex - 1] : null
  const trendValue =
    referenceMonth && referenceMonth.total > 0
      ? ((currentMonth.total - referenceMonth.total) / referenceMonth.total) * 100
      : Number.NaN

  return {
    year: selectedYear.value,
    total,
    count,
    averagePerActiveMonth,
    bestMonth,
    currentMonth,
    trendValue,
    monthlyTotals,
  }
})

const incomeAggregates = computed(() => {
  const timelineMap = new Map()
  const sourceMap = new Map()

  for (const income of filteredIncomes.value) {
    const amount = Number(income.amount || 0)
    const date = new Date(income.income_date)
    if (Number.isNaN(date.getTime())) continue

    const timelineKey =
      chartGranularity.value === 'day'
        ? `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
        : `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`

    const sourceKey = income?.source?.trim() || 'Sans source'

    timelineMap.set(timelineKey, (timelineMap.get(timelineKey) || 0) + amount)
    sourceMap.set(sourceKey, (sourceMap.get(sourceKey) || 0) + amount)
  }

  const timelineEntries = Array.from(timelineMap.entries()).sort((a, b) => a[0].localeCompare(b[0]))
  const sortedSourceEntries = Array.from(sourceMap.entries()).sort((a, b) => b[1] - a[1])
  const topSources = sortedSourceEntries.slice(0, MAX_VISIBLE_SOURCES)
  const otherSources = sortedSourceEntries.slice(MAX_VISIBLE_SOURCES)
  const otherTotal = otherSources.reduce((sum, [, amount]) => sum + amount, 0)
  const finalSourceEntries = otherTotal > 0 ? [...topSources, ['Autres', otherTotal]] : topSources

  return {
    timelineValues: timelineEntries.map(([, total]) => total),
    sourceLabels: finalSourceEntries.map(([source]) => source),
    sourceValues: finalSourceEntries.map(([, total]) => total),
  }
})

const comparisonCards = computed(() => {
  const months = annualSummary.value.monthlyTotals

  return months.map((month, index) => {
    const previous = index > 0 ? months[index - 1].total : 0
    const variation = previous > 0 ? ((month.total - previous) / previous) * 100 : Number.NaN

    return {
      ...month,
      variation,
      isBest: month.monthIndex === annualSummary.value.bestMonth.monthIndex && month.total > 0,
      isSelected: month.monthIndex === selectedMonthIndex.value,
    }
  })
})

const selectedMonthDetails = computed(() => {
  if (selectedMonthIndex.value == null) return null

  const month = comparisonCards.value.find((item) => item.monthIndex === selectedMonthIndex.value)
  if (!month) return null

  const breakdown = new Map()
  let entryCount = 0

  for (const income of incomes.value) {
    if (!income?.income_date) continue
    const date = new Date(income.income_date)
    if (Number.isNaN(date.getTime())) continue
    if (date.getFullYear() !== selectedYear.value || date.getMonth() !== month.monthIndex) continue

    const source = income?.source?.trim() || 'Sans source'
    const amount = Number(income.amount || 0)
    entryCount += 1
    breakdown.set(source, (breakdown.get(source) || 0) + amount)
  }

  const topSources = Array.from(breakdown.entries())
    .sort((a, b) => b[1] - a[1])
    .slice(0, 4)
    .map(([label, value]) => ({ label, value }))

  return {
    ...month,
    entryCount,
    topSources,
  }
})

const topSourcesSummary = computed(() => {
  const total = incomeAggregates.value.sourceValues.reduce((sum, value) => sum + value, 0)

  return incomeAggregates.value.sourceLabels.map((label, index) => {
    const value = incomeAggregates.value.sourceValues[index] || 0
    const share = total > 0 ? (value / total) * 100 : 0

    return {
      label,
      value,
      share,
      color: SOURCE_CHART_COLORS[index % SOURCE_CHART_COLORS.length],
    }
  })
})

const annualPulse = computed(() => {
  const months = annualSummary.value.monthlyTotals.filter((month) => month.total > 0)
  if (months.length < 2) return 'Lecture en construction'

  const first = months[0].total
  const last = months.at(-1)?.total || 0
  if (!first) return 'Lecture en construction'

  const delta = ((last - first) / first) * 100
  return `${delta > 0 ? 'Trajectoire haussière' : delta < 0 ? 'Trajectoire plus douce' : 'Rythme stable'} · ${formatPercent(delta)}`
})

const dashboardHighlights = computed(() => {
  const totalVisible = incomeAggregates.value.timelineValues.reduce((sum, value) => sum + value, 0)
  const topSource = topSourcesSummary.value[0]
  const periodsCount = incomeAggregates.value.timelineValues.length

  return [
    {
      label: 'Total affiché',
      value: formatCurrency(totalVisible),
      caption: 'Périmètre filtré',
      tone: 'primary',
    },
    {
      label: 'Mois record',
      value:
        annualSummary.value.bestMonth.total > 0
          ? `${annualSummary.value.bestMonth.label} · ${formatCurrency(annualSummary.value.bestMonth.total, { decimals: 0 })}`
          : 'Aucune donnée',
      caption: 'Pic de performance',
      tone: 'champagne',
    },
    {
      label: 'Source dominante',
      value: topSource ? `${topSource.label} · ${topSource.share.toFixed(0)}%` : 'Aucune',
      caption: 'Poids principal',
      tone: 'slate',
    },
    {
      label: 'Périodes comparées',
      value: `${periodsCount} ${chartGranularity.value === 'day' ? 'jours' : 'mois'}`,
      caption: annualPulse.value,
      tone: 'neutral',
    },
  ]
})

const hasAnnualData = computed(() => annualSummary.value.total > 0)

function toggleSelectedMonth(monthIndex) {
  selectedMonthIndex.value = selectedMonthIndex.value === monthIndex ? null : monthIndex
}
</script>

<template>
  <section class="income-dashboard-fintech">
    <article class="fin-hero panel fin-shell">
      <div class="fin-hero-grid">
        <div class="fin-hero-copy">
          <div class="fin-hero-copy-top">
            <p class="fin-kicker">Pilotage des revenus</p>
            <button
              class="fin-collapse-toggle"
              type="button"
              :aria-expanded="!isCollapsed"
              @click="toggleCollapsed"
            >
              {{ isCollapsed ? 'Afficher' : 'Masquer' }}
            </button>
          </div>
          <h2>Comprendre d'où vient votre argent, mois après mois</h2>
          <p class="fin-hero-text">
            Filtrez par période ou source, comparez les mois entre eux et repérez rapidement
            les rentrées qui pèsent le plus dans votre budget.
          </p>

          <div class="fin-hero-chips">
            <span class="fin-chip">KPI plus structurés</span>
            <span class="fin-chip">Filtres plus premium</span>
            <span class="fin-chip">Lecture mensuelle guidée</span>
          </div>
        </div>

        <aside class="fin-hero-rail">
          <div class="fin-rail-card">
            <span class="fin-rail-label">Signal récent</span>
            <strong
              class="fin-rail-value"
              :class="{ positive: annualSummary.trendValue > 0, negative: annualSummary.trendValue < 0 }"
            >
              {{ formatPercent(annualSummary.trendValue) }}
            </strong>
            <span class="fin-rail-caption">
              {{ annualSummary.currentMonth.label }} vs mois précédent
            </span>
          </div>

          <div class="fin-rail-card muted-card">
            <span class="fin-rail-label">Mouvement annuel</span>
            <strong class="fin-rail-secondary">{{ annualPulse }}</strong>
          </div>
        </aside>
      </div>
    </article>

    <template v-if="!isCollapsed">
    <section class="fin-kpi-row">
      <article
        v-for="item in dashboardHighlights"
        :key="item.label"
        class="panel fin-shell fin-kpi-card"
        :class="[`tone-${item.tone}`]"
      >
        <span class="fin-kpi-label">{{ item.label }}</span>
        <strong class="fin-kpi-value">{{ item.value }}</strong>
        <span class="fin-kpi-caption">{{ item.caption }}</span>
      </article>
    </section>

    <section class="panel fin-shell fin-controls">
      <div class="fin-section-head">
        <div>
          <p class="fin-kicker">Paramètres</p>
          <h3>Affiner la lecture</h3>
        </div>
      </div>

      <div class="fin-controls-grid">
        <label class="fin-field">
          <span class="fin-field-label">Période</span>
          <select v-model="selectedRange" class="fin-input">
            <option v-for="option in periodOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </label>

        <label class="fin-field">
          <span class="fin-field-label">Source</span>
          <select v-model="selectedSource" class="fin-input">
            <option v-for="option in sourceOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </label>

        <label class="fin-field">
          <span class="fin-field-label">Exercice</span>
          <select v-model="selectedYear" class="fin-input">
            <option v-for="year in availableYears" :key="year" :value="year">
              {{ year }}
            </option>
          </select>
        </label>

        <label class="fin-field fin-field-search">
          <span class="fin-field-label">Recherche</span>
          <input
            v-model="searchQuery"
            type="text"
            class="fin-input"
            placeholder="Source, note, montant..."
          />
        </label>
      </div>
    </section>

    <article class="panel fin-shell fin-list-card">
      <div class="fin-section-head compact-head">
        <div>
          <p class="fin-kicker">Classement</p>
          <h3>Sources clés</h3>
        </div>
      </div>

      <div v-if="topSourcesSummary.length" class="fin-source-list">
        <div
          v-for="source in topSourcesSummary"
          :key="source.label"
          class="fin-source-row"
        >
          <div class="fin-source-meta">
            <span class="fin-source-dot" :style="{ backgroundColor: source.color }"></span>
            <span class="fin-source-name">{{ source.label }}</span>
          </div>
          <div class="fin-source-values">
            <strong>{{ formatCurrency(source.value, { decimals: 0 }) }}</strong>
            <span>{{ source.share.toFixed(0) }}%</span>
          </div>
        </div>
      </div>
    </article>

    <section class="fin-bottom-grid">
      <article class="panel fin-shell fin-summary-card">
        <div class="fin-section-head">
          <div>
            <p class="fin-kicker">Vue annuelle</p>
            <h3>Résumé {{ annualSummary.year }}</h3>
          </div>
        </div>

        <div v-if="hasAnnualData" class="fin-summary-metrics">
          <article class="fin-summary-pill">
            <span>Total annuel</span>
            <strong>{{ formatCurrency(annualSummary.total) }}</strong>
          </article>

          <article class="fin-summary-pill">
            <span>Moyenne / mois actif</span>
            <strong>{{ formatCurrency(annualSummary.averagePerActiveMonth) }}</strong>
          </article>

          <article class="fin-summary-pill featured-pill">
            <span>Meilleur mois</span>
            <strong>{{ annualSummary.bestMonth.label }} · {{ formatCurrency(annualSummary.bestMonth.total, { decimals: 0 }) }}</strong>
          </article>

          <article class="fin-summary-pill">
            <span>Entrées</span>
            <strong>{{ annualSummary.count }}</strong>
          </article>
        </div>

        <div v-else class="fin-empty-state fin-empty-state-small">
          <p>Aucun revenu trouvé sur cet exercice.</p>
        </div>
      </article>

      <article class="panel fin-shell fin-comparison-card">
        <div class="fin-section-head">
          <div>
            <p class="fin-kicker">Lecture fine</p>
            <h3>Mois interactifs</h3>
            <p class="fin-muted">Sélectionne un mois pour ouvrir sa fiche détaillée.</p>
          </div>
        </div>

        <div v-if="comparisonCards.length" class="fin-month-grid">
          <button
            v-for="month in comparisonCards"
            :key="month.monthIndex"
            type="button"
            class="fin-month-card"
            :class="{
              'is-best': month.isBest,
              'is-selected': month.isSelected,
            }"
            @click="toggleSelectedMonth(month.monthIndex)"
          >
            <div class="fin-month-top">
              <span class="fin-month-name">{{ month.label }}</span>
              <span v-if="month.isBest" class="fin-month-tag">Top</span>
            </div>
            <strong class="fin-month-value">{{ formatCurrency(month.total, { decimals: 0 }) }}</strong>
            <span class="fin-month-variation" :class="{ positive: month.variation > 0, negative: month.variation < 0 }">
              {{ Number.isFinite(month.variation) ? formatPercent(month.variation) : '—' }}
            </span>
          </button>
        </div>

        <Transition name="fin-fade-up">
          <div v-if="selectedMonthDetails" class="fin-focus-panel">
            <div class="fin-focus-head">
              <div>
                <p class="fin-kicker">Fiche du mois</p>
                <h4>{{ selectedMonthDetails.label }}</h4>
              </div>
              <strong>{{ formatCurrency(selectedMonthDetails.total) }}</strong>
            </div>

            <div class="fin-focus-meta">
              <span>{{ selectedMonthDetails.entryCount }} entrées</span>
              <span>{{ Number.isFinite(selectedMonthDetails.variation) ? formatPercent(selectedMonthDetails.variation) : 'Pas de comparaison' }}</span>
            </div>

            <div v-if="selectedMonthDetails.topSources.length" class="fin-focus-list">
              <div
                v-for="source in selectedMonthDetails.topSources"
                :key="source.label"
                class="fin-focus-row"
              >
                <span>{{ source.label }}</span>
                <strong>{{ formatCurrency(source.value, { decimals: 0 }) }}</strong>
              </div>
            </div>
          </div>
        </Transition>
      </article>
    </section>
    </template>
  </section>
</template>

<style scoped>
.income-dashboard-fintech {
  --fin-bg: var(--bg-elevated, #0a0d12);
  --fin-surface: var(--bg-elevated, rgba(18, 22, 29, 0.86));
  --fin-surface-soft: rgba(var(--tint-rgb), 0.03);
  --fin-border: var(--line-soft, rgba(var(--tint-rgb), 0.07));
  --fin-text: var(--text, #f2f6f7);
  --fin-muted: var(--text-soft, #a5b0b8);
  --fin-faint: var(--text-dim, #7f8a93);
  --fin-accent-text: #f3e0b8;
  display: grid;
  gap: 0.65rem;
}

:root[data-theme='light'] .income-dashboard-fintech {
  --fin-accent-text: #8a6a1f;
}

.fin-shell {
  position: relative;
  overflow: hidden;
  border-radius: 24px;
  border: 1px solid var(--fin-border);
  background:
    linear-gradient(180deg, rgba(var(--tint-rgb), 0.05), rgba(var(--tint-rgb), 0.02)),
    linear-gradient(135deg, rgba(215, 221, 216, 0.03), transparent 38%),
    var(--fin-surface);
  box-shadow:
    0 24px 70px rgba(0, 0, 0, 0.24),
    inset 0 1px 0 rgba(var(--tint-rgb), 0.04);
  backdrop-filter: blur(18px);
}

.fin-hero,
.fin-controls,
.fin-list-card,
.fin-summary-card,
.fin-comparison-card {
  padding: 0.75rem;
}

.fin-hero-copy-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.fin-collapse-toggle {
  flex-shrink: 0;
  height: 28px;
  padding: 0 0.75rem;
  border-radius: 10px;
  border: 1px solid var(--fin-border);
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--fin-muted);
  font-size: 0.74rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 140ms ease, transform 140ms ease;
}

.fin-collapse-toggle:hover {
  background: rgba(var(--tint-rgb), 0.08);
  transform: translateY(-1px);
}

.fin-hero-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(300px, 0.78fr);
  gap: 0.65rem;
}

.fin-hero-copy h2 {
  margin: 0.25rem 0 0;
  color: var(--fin-text);
  font-size: 1.25rem;
  line-height: 1.2;
}

.fin-section-head h3,
.fin-focus-head h4 {
  margin: 0.25rem 0 0;
  color: var(--fin-text);
  font-size: 0.92rem;
  line-height: 1.2;
}

.fin-kicker,
.fin-field-label,
.fin-kpi-label,
.fin-rail-label,
.fin-summary-pill span,
.fin-month-name {
  text-transform: uppercase;
  letter-spacing: 0.07em;
  font-size: 0.66rem;
  color: var(--fin-faint);
}

.fin-hero-text,
.fin-muted,
.fin-kpi-caption,
.fin-rail-caption,
.fin-focus-meta,
.fin-source-values span {
  color: var(--fin-muted);
  font-size: 0.8rem;
  line-height: 1.4;
}

.fin-hero-text {
  margin-top: 0.5rem;
  max-width: 62ch;
}

.fin-hero-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
  margin-top: 0.6rem;
}

.fin-chip {
  padding: 0.34rem 0.6rem;
  border-radius: 999px;
  border: 1px solid rgba(var(--tint-rgb),0.08);
  background: rgba(var(--tint-rgb),0.04);
  color: var(--fin-text);
  font-size: 0.7rem;
}

.fin-hero-rail {
  display: grid;
  gap: 0.6rem;
  align-content: start;
}

.fin-rail-card {
  padding: 0.7rem;
  border-radius: 18px;
  border: 1px solid rgba(var(--tint-rgb),0.07);
  background: var(--fin-surface);
}

.muted-card {
  background: rgba(var(--tint-rgb),0.035);
}

.fin-rail-value,
.fin-rail-secondary {
  display: block;
  margin-top: 0.3rem;
  color: var(--fin-text);
}

.fin-rail-value {
  font-size: 1.3rem;
  line-height: 1;
}

.fin-rail-secondary {
  font-size: 0.85rem;
  line-height: 1.3;
}

.fin-rail-value.positive {
  color: var(--positive-text);
}

.fin-rail-value.negative {
  color: var(--negative-text);
}

.fin-kpi-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.65rem;
}

.fin-kpi-card {
  padding: 0.6rem 0.7rem;
}

.fin-kpi-card.tone-primary {
  background:
    linear-gradient(180deg, rgba(215, 221, 216, 0.12), rgba(var(--tint-rgb),0.02)),
    rgba(var(--tint-rgb),0.02);
}

.fin-kpi-card.tone-champagne {
  background:
    linear-gradient(180deg, rgba(220, 191, 138, 0.14), rgba(var(--tint-rgb),0.02)),
    rgba(var(--tint-rgb),0.02);
}

.fin-kpi-card.tone-slate {
  background:
    linear-gradient(180deg, rgba(169, 184, 198, 0.12), rgba(var(--tint-rgb),0.02)),
    rgba(var(--tint-rgb),0.02);
}

.fin-kpi-value {
  display: block;
  margin-top: 0.4rem;
  color: var(--fin-text);
  font-size: 0.88rem;
  line-height: 1.25;
}

.fin-controls-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.6rem;
  margin-top: 0.65rem;
}

.fin-field {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.fin-input {
  width: 100%;
  height: 34px;
  padding: 0 0.7rem;
  border-radius: 11px;
  border: 1px solid rgba(var(--tint-rgb),0.08);
  background: rgba(var(--tint-rgb),0.045);
  color: var(--fin-text);
  outline: none;
  font-size: 0.8rem;
}

.fin-bottom-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.52fr) minmax(320px, 0.88fr);
  gap: 0.65rem;
}

.fin-source-list,
.fin-focus-list,
.fin-summary-metrics {
  display: grid;
  gap: 0.5rem;
  margin-top: 0.65rem;
}

.fin-source-row,
.fin-focus-row,
.fin-summary-pill {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.7rem;
  padding: 0.55rem 0.7rem;
  border-radius: 14px;
  border: 1px solid rgba(var(--tint-rgb),0.05);
  background: rgba(var(--tint-rgb),0.03);
}

.featured-pill {
  background:
    linear-gradient(180deg, rgba(220, 191, 138, 0.14), rgba(var(--tint-rgb),0.02)),
    rgba(var(--tint-rgb),0.02);
  border-color: rgba(220, 191, 138, 0.16);
}

.fin-summary-pill strong,
.fin-source-values strong,
.fin-focus-head strong,
.fin-focus-row strong,
.fin-month-value {
  color: var(--fin-text);
  font-size: 0.86rem;
  line-height: 1.2;
}

.fin-source-meta,
.fin-source-values,
.fin-month-top {
  display: flex;
  align-items: center;
  gap: 0.45rem;
}

.fin-source-dot {
  width: 9px;
  height: 9px;
  border-radius: 999px;
  box-shadow: 0 0 0 3px rgba(var(--tint-rgb),0.04);
}

.fin-source-name,
.fin-focus-row span {
  color: var(--fin-text);
  font-size: 0.82rem;
}

.fin-month-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.5rem;
  margin-top: 0.65rem;
}

.fin-month-card {
  padding: 0.6rem;
  border-radius: 15px;
  border: 1px solid rgba(var(--tint-rgb),0.05);
  background:
    linear-gradient(180deg, rgba(var(--tint-rgb),0.03), rgba(var(--tint-rgb),0.02)),
    rgba(var(--tint-rgb),0.02);
  text-align: left;
  display: flex;
  flex-direction: column;
  gap: 0.32rem;
}

.fin-month-card.is-best {
  background:
    linear-gradient(180deg, rgba(220, 191, 138, 0.14), rgba(var(--tint-rgb),0.02)),
    rgba(var(--tint-rgb),0.02);
}

.fin-month-card.is-selected {
  border-color: rgba(215, 221, 216, 0.24);
  box-shadow: 0 14px 34px rgba(0,0,0,0.18);
}

.fin-month-tag {
  padding: 0.18rem 0.42rem;
  border-radius: 999px;
  background: rgba(220, 191, 138, 0.18);
  color: var(--fin-accent-text);
  font-size: 0.62rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.fin-month-variation {
  color: var(--fin-muted);
  font-size: 0.76rem;
}

.fin-month-variation.positive {
  color: var(--positive-text);
}

.fin-month-variation.negative {
  color: var(--negative-text);
}

.fin-focus-panel {
  margin-top: 0.65rem;
  padding: 0.7rem;
  border-radius: 16px;
  border: 1px solid rgba(var(--tint-rgb),0.07);
  background:
    linear-gradient(180deg, rgba(var(--tint-rgb),0.04), rgba(var(--tint-rgb),0.02)),
    rgba(var(--tint-rgb),0.02);
}

.fin-focus-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.85rem;
}

.fin-focus-meta {
  display: flex;
  gap: 0.85rem;
  flex-wrap: wrap;
  margin-top: 0.55rem;
}

.fin-empty-state {
  min-height: 170px;
  margin-top: 0.65rem;
  border-radius: 16px;
  display: grid;
  place-items: center;
  text-align: center;
  color: var(--fin-muted);
  background: rgba(var(--tint-rgb),0.022);
  border: 1px dashed rgba(var(--tint-rgb),0.08);
  padding: 0.9rem;
}

.fin-empty-state-small {
  min-height: 110px;
}

.fin-fade-up-enter-active,
.fin-fade-up-leave-active {
  transition: opacity 220ms ease, transform 220ms ease;
}

.fin-fade-up-enter-from,
.fin-fade-up-leave-to {
  opacity: 0;
  transform: translateY(8px);
}

@media (max-width: 1240px) {
  .fin-kpi-row,
  .fin-controls-grid,
  .fin-bottom-grid,
  .fin-hero-grid {
    grid-template-columns: 1fr 1fr;
  }

  .fin-hero-grid,
  .fin-bottom-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 860px) {
  .fin-kpi-row,
  .fin-controls-grid,
  .fin-month-grid {
    grid-template-columns: 1fr;
  }

  .fin-focus-head,
  .fin-summary-pill,
  .fin-source-row,
  .fin-focus-row {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
