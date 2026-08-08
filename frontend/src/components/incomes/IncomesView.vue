<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'

import PageHero from '../common/PageHero.vue'
import ConfirmModal from '../common/ConfirmModal.vue'
import ColumnVisibilityMenu from '../Common/ColumnVisibilityMenu.vue'
import IncomeCharts from './IncomeCharts.vue'
import IncomeRowCard from './IncomeRowCard.vue'
import { usePurchasesStore } from '../../stores/purchases'
import { useTableColumns } from '../../composables/useTableColumns'

const props = defineProps({
  isLoading: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['create', 'edit', 'delete', 'bulk-delete'])

const store = usePurchasesStore()
const { incomes } = storeToRefs(store)

// Contexte de la vue (tri, filtres, pagination) conservé au fil des
// navigations — sans ça, quitter l'écran puis y revenir remettait tout à
// zéro à chaque fois (la vue est démontée/remontée, pas juste masquée).
const CONTEXT_STORAGE_KEY = 'incomesViewContext'

function loadStoredContext() {
  try {
    return JSON.parse(localStorage.getItem(CONTEXT_STORAGE_KEY) || '{}')
  } catch {
    return {}
  }
}

const storedContext = loadStoredContext()

const selectedIncomeId = ref(null)
const sortBy = ref(storedContext.sortBy || 'date')
const sortDirection = ref(storedContext.sortDirection || 'desc')
const displayMode = ref(storedContext.displayMode || 'list')
const isGrouped = computed(() => displayMode.value === 'grouped')
const visibleCountBySource = ref({})

const INCOME_TABLE_COLUMNS = [
  { key: 'date', label: 'Date', width: 100 },
  { key: 'source', label: 'Source', width: 180 },
  { key: 'category', label: 'Catégorie', width: 160 },
  { key: 'subCategory', label: 'Sous-catégorie', width: 160 },
  { key: 'reference', label: 'Référence', width: 140 },
  { key: 'statementReference', label: 'Relevé', width: 120 },
  { key: 'isReconciled', label: 'Pointé', width: 90 },
  { key: 'amount', label: 'Montant', width: 110 },
]

const {
  isVisible: isColumnVisible,
  isFrozen: isColumnFrozen,
  toggleVisible: toggleColumnVisible,
  toggleFrozen: toggleColumnFrozen,
  columnStyle,
} = useTableColumns('incomesTableColumns', INCOME_TABLE_COLUMNS)
const activityRange = ref(storedContext.activityRange || 'all')
const activitySource = ref(storedContext.activitySource || 'all')
const activityCategory = ref(storedContext.activityCategory || 'all')
const activitySubCategory = ref(storedContext.activitySubCategory || 'all')
const activityReconciliation = ref(storedContext.activityReconciliation || 'all')

// La sous-catégorie dépend de la catégorie choisie — si elle change, l'ancien
// choix de sous-catégorie n'a plus forcément de sens.
watch(activityCategory, () => {
  activitySubCategory.value = 'all'
})
const checkedIncomeIds = reactive(new Set())
const isBulkDeleting = ref(false)
const isBulkDeleteConfirmOpen = ref(false)

const DEFAULT_VISIBLE_ITEMS = 12

const PAGE_SIZE_OPTIONS = ['25', '50', '100', '500', '1000']
const pageSizeSelection = ref(storedContext.pageSizeSelection || '25')
const customPageSize = ref(storedContext.customPageSize || 25)
const currentPage = ref(storedContext.currentPage || 1)

watch(
  [
    sortBy,
    sortDirection,
    displayMode,
    activityRange,
    activitySource,
    activityCategory,
    activitySubCategory,
    activityReconciliation,
    pageSizeSelection,
    customPageSize,
    currentPage,
  ],
  () => {
    localStorage.setItem(
      CONTEXT_STORAGE_KEY,
      JSON.stringify({
        sortBy: sortBy.value,
        sortDirection: sortDirection.value,
        displayMode: displayMode.value,
        activityRange: activityRange.value,
        activitySource: activitySource.value,
        activityCategory: activityCategory.value,
        activitySubCategory: activitySubCategory.value,
        activityReconciliation: activityReconciliation.value,
        pageSizeSelection: pageSizeSelection.value,
        customPageSize: customPageSize.value,
        currentPage: currentPage.value,
      })
    )
  }
)

const effectivePageSize = computed(() => {
  if (pageSizeSelection.value === 'all') return Infinity
  if (pageSizeSelection.value === 'custom') {
    return customPageSize.value > 0 ? customPageSize.value : DEFAULT_VISIBLE_ITEMS
  }
  return Number(pageSizeSelection.value)
})

function handlePageSizeChange() {
  visibleCountBySource.value = {}
  currentPage.value = 1
}

const isListCollapsed = ref(localStorage.getItem('incomesListCollapsed') === '1')

function toggleListCollapsed() {
  isListCollapsed.value = !isListCollapsed.value
  localStorage.setItem('incomesListCollapsed', isListCollapsed.value ? '1' : '0')
}

const periodOptions = [
  { value: '30d', label: '30 jours' },
  { value: '90d', label: '90 jours' },
  { value: '12m', label: '12 mois' },
  { value: 'ytd', label: 'Cette année' },
  { value: 'all', label: 'Tout' },
]

const sourceFilterOptions = computed(() => {
  const values = new Set()
  for (const income of incomes.value) values.add(income?.source?.trim() || 'Sans source')

  return [
    { value: 'all', label: 'Toutes les sources' },
    ...Array.from(values)
      .sort((a, b) => a.localeCompare(b, 'fr', { sensitivity: 'base' }))
      .map((source) => ({ value: source, label: source })),
  ]
})

const categoryFilterOptions = computed(() => {
  const values = new Set()
  for (const income of incomes.value) values.add(income?.category?.trim() || 'Sans catégorie')

  return [
    { value: 'all', label: 'Toutes les catégories' },
    ...Array.from(values)
      .sort((a, b) => a.localeCompare(b, 'fr', { sensitivity: 'base' }))
      .map((category) => ({ value: category, label: category })),
  ]
})

// Ne propose que les sous-catégories qui apparaissent réellement parmi les
// revenus de la catégorie actuellement filtrée.
const subCategoryFilterOptions = computed(() => {
  if (activityCategory.value === 'all') return []

  const values = new Set()
  for (const income of incomes.value) {
    const category = income?.category?.trim() || 'Sans catégorie'
    if (category !== activityCategory.value) continue
    values.add(income?.subCategory?.trim() || 'Sans sous-catégorie')
  }

  return [
    { value: 'all', label: 'Toutes les sous-catégories' },
    ...Array.from(values)
      .sort((a, b) => a.localeCompare(b, 'fr', { sensitivity: 'base' }))
      .map((subCategory) => ({ value: subCategory, label: subCategory })),
  ]
})

const specificMonthOptions = computed(() => {
  const months = new Map()

  for (const income of incomes.value) {
    if (!income.income_date) continue
    const date = new Date(income.income_date)
    if (Number.isNaN(date.getTime())) continue

    const key = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`
    if (months.has(key)) continue

    const formatted = new Intl.DateTimeFormat('fr-FR', {
      month: 'long',
      year: 'numeric',
    }).format(date)

    months.set(key, formatted.charAt(0).toUpperCase() + formatted.slice(1))
  }

  return Array.from(months.entries())
    .sort((a, b) => b[0].localeCompare(a[0]))
    .map(([key, label]) => ({ value: `month:${key}`, label }))
})

function isWithinRange(dateValue, range) {
  if (range === 'all') return true

  const date = new Date(dateValue)
  if (Number.isNaN(date.getTime())) return false

  if (range.startsWith('month:')) {
    const key = range.slice('month:'.length)
    const dateKey = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`
    return dateKey === key
  }

  const now = new Date()
  const startDate = new Date(now)

  if (range === '30d') startDate.setDate(now.getDate() - 30)
  else if (range === '90d') startDate.setDate(now.getDate() - 90)
  else if (range === '12m') startDate.setMonth(now.getMonth() - 12)
  else if (range === 'ytd') {
    startDate.setMonth(0, 1)
    startDate.setHours(0, 0, 0, 0)
  }

  return date >= startDate && date <= now
}

const activityIncomes = computed(() => {
  return incomes.value.filter((income) => {
    if (!isWithinRange(income.income_date, activityRange.value)) return false

    if (activitySource.value !== 'all') {
      const source = income.source?.trim() || 'Sans source'
      if (source !== activitySource.value) return false
    }

    if (activityCategory.value !== 'all') {
      const category = income.category?.trim() || 'Sans catégorie'
      if (category !== activityCategory.value) return false
    }

    if (activitySubCategory.value !== 'all') {
      const subCategory = income.subCategory?.trim() || 'Sans sous-catégorie'
      if (subCategory !== activitySubCategory.value) return false
    }

    if (activityReconciliation.value === 'reconciled' && !income.isReconciled) return false
    if (activityReconciliation.value === 'unreconciled' && income.isReconciled) return false

    return true
  })
})

const hasActivityFilters = computed(
  () =>
    activityRange.value !== 'all' ||
    activitySource.value !== 'all' ||
    activityCategory.value !== 'all' ||
    activitySubCategory.value !== 'all' ||
    activityReconciliation.value !== 'all'
)

// Les cartes de résumé reflètent les revenus filtrés (et non plus toujours
// l'ensemble des données) — sinon elles ne correspondaient plus à ce qui
// est réellement affiché dans la liste dès qu'un filtre était actif.
const filteredTotalAmount = computed(() =>
  activityIncomes.value.reduce((sum, income) => sum + Number(income.amount || 0), 0)
)
const filteredIncomeCount = computed(() => activityIncomes.value.length)
const filteredAverageAmount = computed(() =>
  filteredIncomeCount.value > 0 ? filteredTotalAmount.value / filteredIncomeCount.value : 0
)

function resetActivityFilters() {
  activityRange.value = 'all'
  activitySource.value = 'all'
  activityCategory.value = 'all'
  activitySubCategory.value = 'all'
  activityReconciliation.value = 'all'
}

function formatSourceLabel(value) {
  return value?.trim() || 'Sans source'
}

function formatTableDate(value) {
  if (!value) return ''

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''

  const day = String(date.getDate()).padStart(2, '0')
  const month = String(date.getMonth() + 1).padStart(2, '0')

  return `${day}/${month}/${date.getFullYear()}`
}

function compareIncomeItems(a, b) {
  if (sortBy.value === 'amount') {
    return Number(a.amount || 0) - Number(b.amount || 0)
  }

  if (sortBy.value === 'source') {
    return formatSourceLabel(a.source).localeCompare(formatSourceLabel(b.source), 'fr', {
      sensitivity: 'base',
    })
  }

  if (sortBy.value === 'category') {
    return (a.category || '').localeCompare(b.category || '', 'fr', { sensitivity: 'base' })
  }

  if (sortBy.value === 'subCategory') {
    return (a.subCategory || '').localeCompare(b.subCategory || '', 'fr', { sensitivity: 'base' })
  }

  if (sortBy.value === 'reference') {
    return (a.reference || '').localeCompare(b.reference || '', 'fr', { sensitivity: 'base' })
  }

  if (sortBy.value === 'isReconciled') {
    return Number(a.isReconciled) - Number(b.isReconciled)
  }

  if (sortBy.value === 'statementReference') {
    return (a.statementReference || '').localeCompare(b.statementReference || '', 'fr', {
      sensitivity: 'base',
    })
  }

  return new Date(a.income_date) - new Date(b.income_date)
}

function setSort(column) {
  if (sortBy.value === column) {
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortBy.value = column
    sortDirection.value = 'asc'
  }
}

function sortIndicator(column) {
  if (sortBy.value !== column) return ''
  return sortDirection.value === 'asc' ? '▲' : '▼'
}

function sortIncomeItems(items) {
  return [...items].sort((a, b) => {
    const result = compareIncomeItems(a, b)
    return sortDirection.value === 'asc' ? result : -result
  })
}

const sortedActivityIncomes = computed(() => sortIncomeItems(activityIncomes.value))

const totalPages = computed(() => {
  if (effectivePageSize.value === Infinity) return 1
  return Math.max(1, Math.ceil(sortedActivityIncomes.value.length / effectivePageSize.value))
})

const paginatedIncomes = computed(() => {
  if (effectivePageSize.value === Infinity) return sortedActivityIncomes.value

  const start = (currentPage.value - 1) * effectivePageSize.value
  return sortedActivityIncomes.value.slice(start, start + effectivePageSize.value)
})

const canGoPreviousPage = computed(() => currentPage.value > 1)
const canGoNextPage = computed(() => currentPage.value < totalPages.value)

function goToPreviousPage() {
  if (canGoPreviousPage.value) currentPage.value -= 1
}

function goToNextPage() {
  if (canGoNextPage.value) currentPage.value += 1
}

watch([activityRange, activitySource, activityReconciliation, sortBy, sortDirection], () => {
  currentPage.value = 1
})

watch(totalPages, (pages) => {
  if (currentPage.value > pages) currentPage.value = pages
})

const groupedIncomes = computed(() => {
  const monthGroups = new Map()

  for (const income of activityIncomes.value) {
    if (!income.income_date) continue

    const date = new Date(income.income_date)
    const monthKey = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`

    if (!monthGroups.has(monthKey)) {
      const formatted = new Intl.DateTimeFormat('fr-FR', {
        month: 'long',
        year: 'numeric',
      }).format(date)

      monthGroups.set(monthKey, {
        key: monthKey,
        label: formatted.charAt(0).toUpperCase() + formatted.slice(1),
        total: 0,
        sourcesMap: new Map(),
      })
    }

    const monthGroup = monthGroups.get(monthKey)
    const sourceKey = income.source?.trim() || 'Sans source'
    const groupSourceKey = `${monthKey}-${sourceKey}`

    if (!monthGroup.sourcesMap.has(groupSourceKey)) {
      monthGroup.sourcesMap.set(groupSourceKey, {
        key: groupSourceKey,
        label: sourceKey,
        total: 0,
        items: [],
      })
    }

    const sourceGroup = monthGroup.sourcesMap.get(groupSourceKey)
    sourceGroup.items.push(income)
    sourceGroup.total += Number(income.amount || 0)
    monthGroup.total += Number(income.amount || 0)
  }

  return Array.from(monthGroups.values())
    .sort((a, b) => b.key.localeCompare(a.key))
    .map((monthGroup) => {
      const sources = Array.from(monthGroup.sourcesMap.values())
        .sort((a, b) => a.label.localeCompare(b.label, 'fr', { sensitivity: 'base' }))
        .map((source) => {
          const sortedItems = sortIncomeItems(source.items)
          const baseCount =
            effectivePageSize.value === Infinity ? sortedItems.length : effectivePageSize.value
          const visibleCount = visibleCountBySource.value[source.key] ?? baseCount

          return {
            ...source,
            items: sortedItems,
            visibleItems: sortedItems.slice(0, visibleCount),
            visibleCount,
            hasMore: sortedItems.length > visibleCount,
            canCollapse: visibleCount > baseCount,
          }
        })

      return {
        key: monthGroup.key,
        label: monthGroup.label,
        total: monthGroup.total,
        sources,
      }
    })
})

const hasIncomeData = computed(() => incomes.value.length > 0)

const hasVisibleActivity = computed(() =>
  isGrouped.value ? groupedIncomes.value.length > 0 : sortedActivityIncomes.value.length > 0
)

function formatCurrency(value) {
  return new Intl.NumberFormat('fr-FR', {
    style: 'currency',
    currency: 'EUR',
    minimumFractionDigits: 2,
  }).format(Number(value || 0))
}

function selectIncome(income) {
  selectedIncomeId.value = income.id
}

function isChecked(id) {
  return checkedIncomeIds.has(id)
}

function toggleChecked(income) {
  if (checkedIncomeIds.has(income.id)) checkedIncomeIds.delete(income.id)
  else checkedIncomeIds.add(income.id)
}

const hasSelection = computed(() => checkedIncomeIds.size > 0)

const isAllSelected = computed(() => {
  return (
    activityIncomes.value.length > 0 &&
    activityIncomes.value.every((income) => checkedIncomeIds.has(income.id))
  )
})

function selectAll() {
  for (const income of activityIncomes.value) checkedIncomeIds.add(income.id)
}

function clearSelection() {
  checkedIncomeIds.clear()
}

function toggleSelectAll() {
  if (isAllSelected.value) clearSelection()
  else selectAll()
}

function requestDeleteSelected() {
  if (!checkedIncomeIds.size) return
  isBulkDeleteConfirmOpen.value = true
}

async function deleteSelected() {
  const ids = Array.from(checkedIncomeIds)
  if (!ids.length) return

  isBulkDeleting.value = true

  let successCount = 0
  let failCount = 0

  for (const id of ids) {
    try {
      await store.removeIncome(id)
      checkedIncomeIds.delete(id)
      successCount += 1
    } catch {
      failCount += 1
    }
  }

  isBulkDeleting.value = false
  isBulkDeleteConfirmOpen.value = false
  emit('bulk-delete', { successCount, failCount })
}

function toggleSortDirection() {
  sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
}

function showMore(sourceKey) {
  const step = effectivePageSize.value === Infinity ? DEFAULT_VISIBLE_ITEMS : effectivePageSize.value
  const current = visibleCountBySource.value[sourceKey] ?? step

  visibleCountBySource.value = {
    ...visibleCountBySource.value,
    [sourceKey]: current + step,
  }
}

function showLess(sourceKey) {
  const step = effectivePageSize.value === Infinity ? DEFAULT_VISIBLE_ITEMS : effectivePageSize.value

  visibleCountBySource.value = {
    ...visibleCountBySource.value,
    [sourceKey]: step,
  }
}

function handleCreate() {
  emit('create')
}

function handleEdit(income) {
  emit('edit', income)
}

function handleDelete(income) {
  emit('delete', income)
}
</script>

<template>
  <main class="dashboard-content incomes-view">
    <PageHero
      eyebrow="Entrées d’argent"
      title="Revenus"
      description="Ajoute tes salaires, primes et autres entrées d’argent pour suivre ton équilibre financier global."
    >
      <template #actions>
        <button class="primary-btn" type="button" @click="handleCreate">
          + Nouveau revenu
        </button>
      </template>
    </PageHero>

    <section class="incomes-stats-grid">
      <article class="panel stat-card">
        <p class="eyebrow">Total revenus</p>
        <h2>{{ formatCurrency(filteredTotalAmount) }}</h2>
        <p>{{ hasActivityFilters ? 'Montant cumulé (filtré)' : 'Montant cumulé enregistré' }}</p>
      </article>

      <article class="panel stat-card">
        <p class="eyebrow">Nombre d’entrées</p>
        <h2>{{ filteredIncomeCount }}</h2>
        <p>{{ hasActivityFilters ? 'Revenus correspondant au filtre' : 'Revenus suivis dans l’application' }}</p>
      </article>

      <article class="panel stat-card">
        <p class="eyebrow">Montant moyen</p>
        <h2>{{ formatCurrency(filteredAverageAmount) }}</h2>
        <p>Moyenne par revenu{{ hasActivityFilters ? ' (filtré)' : ' saisi' }}</p>
      </article>
    </section>

    <IncomeCharts v-if="!props.isLoading && hasIncomeData" />

    <section v-if="props.isLoading" class="panel incomes-state">
      Chargement des revenus...
    </section>

    <section v-else-if="!hasIncomeData" class="panel empty-state">
      <div class="empty-state-icon">€</div>
      <h2>Aucun revenu pour le moment</h2>
      <p>
        Ajoute ton premier salaire ou une autre entrée d’argent pour suivre ton solde net.
      </p>
      <button class="primary-btn" type="button" @click="handleCreate">
        Ajouter un revenu
      </button>
    </section>

    <section v-else-if="!hasVisibleActivity" class="panel empty-state">
      <div class="empty-state-icon">€</div>
      <h2>Aucun résultat pour ces filtres</h2>
      <p>
        Aucun revenu ne correspond à la période et à la source sélectionnées.
      </p>
      <button class="ghost-btn" type="button" @click="resetActivityFilters">
        Réinitialiser les filtres
      </button>
    </section>

    <section v-else class="incomes-layout">
      <article class="panel incomes-list-card">
        <div class="panel-header incomes-panel-header">
          <div>
            <p class="eyebrow">Historique</p>
            <h2>
              {{
                displayMode === 'grouped'
                  ? 'Revenus par mois et par source'
                  : displayMode === 'table'
                    ? 'Tableau des revenus'
                    : 'Liste des revenus'
              }}
            </h2>
          </div>

          <button
            class="ghost-btn panel-collapse-toggle"
            type="button"
            :aria-expanded="!isListCollapsed"
            @click="toggleListCollapsed"
          >
            {{ isListCollapsed ? 'Afficher' : 'Masquer' }}
          </button>
        </div>

        <div v-if="!isListCollapsed" class="incomes-toolbar">
            <label class="sort-field">
              <span class="sort-label">Période</span>
              <select v-model="activityRange" class="sort-select">
                <option v-for="option in periodOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
                <optgroup v-if="specificMonthOptions.length" label="Mois spécifique">
                  <option v-for="month in specificMonthOptions" :key="month.value" :value="month.value">
                    {{ month.label }}
                  </option>
                </optgroup>
              </select>
            </label>

            <label class="sort-field">
              <span class="sort-label">Source</span>
              <select v-model="activitySource" class="sort-select">
                <option v-for="option in sourceFilterOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </label>

            <label class="sort-field">
              <span class="sort-label">Catégorie</span>
              <select v-model="activityCategory" class="sort-select">
                <option v-for="option in categoryFilterOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </label>

            <label v-if="activityCategory !== 'all'" class="sort-field">
              <span class="sort-label">Sous-catégorie</span>
              <select v-model="activitySubCategory" class="sort-select">
                <option v-for="option in subCategoryFilterOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </label>

            <label class="sort-field">
              <span class="sort-label">Pointage</span>
              <select v-model="activityReconciliation" class="sort-select">
                <option value="all">Tous</option>
                <option value="reconciled">Pointés</option>
                <option value="unreconciled">Non pointés</option>
              </select>
            </label>

            <label class="sort-field">
              <span class="sort-label">Trier les lignes par</span>
              <select v-model="sortBy" class="sort-select">
                <option value="date">Date</option>
                <option value="amount">Montant</option>
                <option value="source">Source</option>
                <option value="category">Catégorie</option>
                <option value="subCategory">Sous-catégorie</option>
                <option value="reference">Référence</option>
                <option value="isReconciled">Pointé</option>
                <option value="statementReference">Relevé</option>
              </select>
            </label>

            <button
              class="sort-direction-btn sort-arrow-btn"
              type="button"
              :aria-label="sortDirection === 'asc' ? 'Tri croissant' : 'Tri décroissant'"
              :title="sortDirection === 'asc' ? 'Tri croissant' : 'Tri décroissant'"
              @click="toggleSortDirection"
            >
              {{ sortDirection === 'asc' ? '↑' : '↓' }}
            </button>

            <label class="sort-field">
              <span class="sort-label">Affichage</span>
              <select v-model="displayMode" class="sort-select">
                <option value="list">Liste simple</option>
                <option value="grouped">Regroupé</option>
                <option value="table">Tableau</option>
              </select>
            </label>

            <ColumnVisibilityMenu
              v-if="displayMode === 'table'"
              :columns="INCOME_TABLE_COLUMNS"
              :is-visible="isColumnVisible"
              :is-frozen="isColumnFrozen"
              @toggle-visible="toggleColumnVisible"
              @toggle-frozen="toggleColumnFrozen"
            />

            <label class="sort-field">
              <span class="sort-label">Éléments par page</span>
              <select
                v-model="pageSizeSelection"
                class="sort-select"
                @change="handlePageSizeChange"
              >
                <option v-for="option in PAGE_SIZE_OPTIONS" :key="option" :value="option">
                  {{ option }}
                </option>
                <option value="all">Tout</option>
                <option value="custom">Personnalisé…</option>
              </select>
            </label>

            <input
              v-if="pageSizeSelection === 'custom'"
              type="number"
              min="1"
              v-model.number="customPageSize"
              class="sort-select page-size-custom-input"
              placeholder="Nombre"
              @change="handlePageSizeChange"
            />

            <button
              v-if="hasActivityFilters"
              class="ghost-btn reset-filters-btn"
              type="button"
              @click="resetActivityFilters"
            >
              Réinitialiser
            </button>
          </div>

        <div v-if="!isListCollapsed" class="incomes-selection-bar">
          <button class="ghost-btn" type="button" @click="toggleSelectAll">
            {{ isAllSelected ? 'Tout désélectionner' : 'Tout sélectionner' }}
          </button>

          <span class="selection-count">
            {{ checkedIncomeIds.size }} / {{ filteredIncomeCount }} sélectionné{{ checkedIncomeIds.size > 1 ? 's' : '' }}
          </span>

          <button
            v-if="hasSelection"
            class="danger-btn"
            type="button"
            :disabled="isBulkDeleting"
            @click="requestDeleteSelected"
          >
            {{ isBulkDeleting ? 'Suppression…' : 'Supprimer la sélection' }}
          </button>
        </div>

        <div v-if="!isListCollapsed" class="incomes-list">
          <template v-if="isGrouped">
            <section
              v-for="group in groupedIncomes"
              :key="group.key"
              class="income-month-group"
            >
              <div class="income-month-header">
                <div class="income-month-heading">
                  <p class="eyebrow">Période</p>
                  <h3>{{ group.label }}</h3>
                </div>

                <div class="income-amount income-month-total">
                  {{ formatCurrency(group.total) }}
                </div>
              </div>

              <section
                v-for="source in group.sources"
                :key="source.key"
                class="income-source-group"
              >
                <div class="income-source-header">
                  <div class="income-source-heading">
                    <p class="eyebrow">Source</p>
                    <h4>{{ source.label }}</h4>
                  </div>

                  <div class="income-amount income-source-total">
                    {{ formatCurrency(source.total) }}
                  </div>
                </div>

                <div class="income-source-list">
                  <IncomeRowCard
                    v-for="income in source.visibleItems"
                    :key="income.id"
                    :income="income"
                    :selected="selectedIncomeId === income.id"
                    :checked="isChecked(income.id)"
                    @select="selectIncome"
                    @edit="handleEdit"
                    @delete="handleDelete"
                    @toggle-check="toggleChecked"
                  />
                </div>

                <div
                  v-if="source.hasMore || source.canCollapse"
                  class="income-source-footer"
                >
                  <button
                    v-if="source.hasMore"
                    class="secondary-btn income-more-btn"
                    type="button"
                    @click="showMore(source.key)"
                  >
                    Voir plus
                  </button>

                  <button
                    v-if="source.canCollapse"
                    class="ghost-btn income-less-btn"
                    type="button"
                    @click="showLess(source.key)"
                  >
                    Voir moins
                  </button>
                </div>
              </section>
            </section>
          </template>

          <template v-else-if="displayMode === 'table'">
            <div class="income-table-wrap">
              <table class="income-table">
                <colgroup>
                  <col class="col-checkbox" />
                  <col v-if="isColumnVisible('date')" style="width: 100px" />
                  <col v-if="isColumnVisible('source')" class="col-flex" />
                  <col v-if="isColumnVisible('category')" style="width: 160px" />
                  <col v-if="isColumnVisible('subCategory')" style="width: 160px" />
                  <col v-if="isColumnVisible('reference')" style="width: 140px" />
                  <col v-if="isColumnVisible('statementReference')" style="width: 120px" />
                  <col v-if="isColumnVisible('isReconciled')" style="width: 90px" />
                  <col v-if="isColumnVisible('amount')" style="width: 110px" />
                  <col class="col-actions" />
                </colgroup>
                <thead>
                  <tr>
                    <th class="checkbox-col"></th>
                    <th
                      v-if="isColumnVisible('date')"
                      class="sortable"
                      :class="{ 'is-frozen': isColumnFrozen('date') }"
                      :style="columnStyle('date')"
                      @click="setSort('date')"
                    >
                      Date <span class="sort-indicator">{{ sortIndicator('date') }}</span>
                    </th>
                    <th
                      v-if="isColumnVisible('source')"
                      class="sortable"
                      :class="{ 'is-frozen': isColumnFrozen('source') }"
                      :style="columnStyle('source')"
                      @click="setSort('source')"
                    >
                      Source <span class="sort-indicator">{{ sortIndicator('source') }}</span>
                    </th>
                    <th
                      v-if="isColumnVisible('category')"
                      class="sortable"
                      :class="{ 'is-frozen': isColumnFrozen('category') }"
                      :style="columnStyle('category')"
                      @click="setSort('category')"
                    >
                      Catégorie <span class="sort-indicator">{{ sortIndicator('category') }}</span>
                    </th>
                    <th
                      v-if="isColumnVisible('subCategory')"
                      class="sortable"
                      :class="{ 'is-frozen': isColumnFrozen('subCategory') }"
                      :style="columnStyle('subCategory')"
                      @click="setSort('subCategory')"
                    >
                      Sous-catégorie <span class="sort-indicator">{{ sortIndicator('subCategory') }}</span>
                    </th>
                    <th
                      v-if="isColumnVisible('reference')"
                      class="sortable"
                      :class="{ 'is-frozen': isColumnFrozen('reference') }"
                      :style="columnStyle('reference')"
                      @click="setSort('reference')"
                    >
                      Référence <span class="sort-indicator">{{ sortIndicator('reference') }}</span>
                    </th>
                    <th
                      v-if="isColumnVisible('statementReference')"
                      class="sortable"
                      :class="{ 'is-frozen': isColumnFrozen('statementReference') }"
                      :style="columnStyle('statementReference')"
                      @click="setSort('statementReference')"
                    >
                      Relevé <span class="sort-indicator">{{ sortIndicator('statementReference') }}</span>
                    </th>
                    <th
                      v-if="isColumnVisible('isReconciled')"
                      class="sortable"
                      :class="{ 'is-frozen': isColumnFrozen('isReconciled') }"
                      :style="columnStyle('isReconciled')"
                      @click="setSort('isReconciled')"
                    >
                      Pointé <span class="sort-indicator">{{ sortIndicator('isReconciled') }}</span>
                    </th>
                    <th
                      v-if="isColumnVisible('amount')"
                      class="sortable"
                      :class="{ 'is-frozen': isColumnFrozen('amount') }"
                      :style="columnStyle('amount')"
                      @click="setSort('amount')"
                    >
                      Montant <span class="sort-indicator">{{ sortIndicator('amount') }}</span>
                    </th>
                    <th class="actions-col">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="income in paginatedIncomes"
                    :key="income.id"
                    :class="{ 'is-selected-row': selectedIncomeId === income.id }"
                    @dblclick="handleEdit(income)"
                  >
                    <td class="checkbox-col">
                      <input
                        type="checkbox"
                        :checked="isChecked(income.id)"
                        @click.stop
                        @change="toggleChecked(income)"
                      />
                    </td>
                    <td v-if="isColumnVisible('date')" :class="{ 'is-frozen': isColumnFrozen('date') }" :style="columnStyle('date')">
                      {{ formatTableDate(income.income_date) }}
                    </td>
                    <td
                      v-if="isColumnVisible('source')"
                      class="label-cell"
                      :class="{ 'is-frozen': isColumnFrozen('source') }"
                      :style="columnStyle('source')"
                    >
                      {{ formatSourceLabel(income.source) }}
                    </td>
                    <td v-if="isColumnVisible('category')" :class="{ 'is-frozen': isColumnFrozen('category') }" :style="columnStyle('category')">
                      {{ income.category || '—' }}
                    </td>
                    <td v-if="isColumnVisible('subCategory')" :class="{ 'is-frozen': isColumnFrozen('subCategory') }" :style="columnStyle('subCategory')">
                      {{ income.subCategory || '—' }}
                    </td>
                    <td v-if="isColumnVisible('reference')" :class="{ 'is-frozen': isColumnFrozen('reference') }" :style="columnStyle('reference')">
                      {{ income.reference || '—' }}
                    </td>
                    <td v-if="isColumnVisible('statementReference')" :class="{ 'is-frozen': isColumnFrozen('statementReference') }" :style="columnStyle('statementReference')">
                      {{ income.statementReference || '—' }}
                    </td>
                    <td v-if="isColumnVisible('isReconciled')" :class="{ 'is-frozen': isColumnFrozen('isReconciled') }" :style="columnStyle('isReconciled')">
                      <span class="reconciled-badge" :class="income.isReconciled ? 'is-reconciled' : 'is-pending'">
                        {{ income.isReconciled ? '✓ Pointé' : 'Non pointé' }}
                      </span>
                    </td>
                    <td
                      v-if="isColumnVisible('amount')"
                      class="amount-positive"
                      :class="{ 'is-frozen': isColumnFrozen('amount') }"
                      :style="columnStyle('amount')"
                    >
                      +{{ formatCurrency(income.amount) }}
                    </td>
                    <td class="actions-col">
                      <div class="actions-col-inner">
                        <button class="icon-btn" type="button" title="Modifier" @click="handleEdit(income)">✏️</button>
                        <button class="icon-btn icon-btn-danger" type="button" title="Supprimer" @click="handleDelete(income)">🗑️</button>
                      </div>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <div v-if="totalPages > 1" class="pagination-bar">
              <button
                class="ghost-btn"
                type="button"
                :disabled="!canGoPreviousPage"
                @click="goToPreviousPage"
              >
                Précédent
              </button>

              <span class="pagination-status">Page {{ currentPage }} sur {{ totalPages }}</span>

              <button
                class="ghost-btn"
                type="button"
                :disabled="!canGoNextPage"
                @click="goToNextPage"
              >
                Suivant
              </button>
            </div>
          </template>

          <template v-else>
            <div class="income-source-list">
              <IncomeRowCard
                v-for="income in paginatedIncomes"
                :key="income.id"
                :income="income"
                :selected="selectedIncomeId === income.id"
                :checked="isChecked(income.id)"
                @select="selectIncome"
                @edit="handleEdit"
                @delete="handleDelete"
                @toggle-check="toggleChecked"
              />
            </div>

            <div v-if="totalPages > 1" class="pagination-bar">
              <button
                class="ghost-btn"
                type="button"
                :disabled="!canGoPreviousPage"
                @click="goToPreviousPage"
              >
                Précédent
              </button>

              <span class="pagination-status">Page {{ currentPage }} sur {{ totalPages }}</span>

              <button
                class="ghost-btn"
                type="button"
                :disabled="!canGoNextPage"
                @click="goToNextPage"
              >
                Suivant
              </button>
            </div>
          </template>
        </div>
      </article>
    </section>

    <ConfirmModal
      v-model="isBulkDeleteConfirmOpen"
      :title="`Supprimer ${checkedIncomeIds.size} revenu${checkedIncomeIds.size > 1 ? 's' : ''} ?`"
      :message="`Tu es sur le point de supprimer définitivement ${checkedIncomeIds.size} revenu${checkedIncomeIds.size > 1 ? 's' : ''} sélectionné${checkedIncomeIds.size > 1 ? 's' : ''}.`"
      note="Vérifie ta sélection avant de confirmer : cette action est irréversible."
      confirm-label="Supprimer la sélection"
      :is-processing="isBulkDeleting"
      @confirm="deleteSelected"
    />
  </main>
</template>

<style scoped>
.incomes-view {
  display: grid;
  gap: 0.9rem;
}

.incomes-stats-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.7rem;
}

.stat-card {
  padding: 0.85rem 0.95rem;
}

.stat-card h2 {
  margin: 0.22rem 0 0.15rem;
}

.stat-card p:last-child {
  color: var(--text-muted, #94a3b8);
}

.incomes-layout {
  display: grid;
  grid-template-columns: 1fr;
}

.incomes-list-card {
  min-height: 100%;
  padding: 0.95rem;
}

.incomes-panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.panel-collapse-toggle {
  flex-shrink: 0;
  height: 32px;
  padding: 0 0.8rem;
  border-radius: 10px;
  font-size: 0.8rem;
}

.incomes-toolbar {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  flex-wrap: wrap;
}

.incomes-selection-bar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-top: 0.75rem;
  padding-top: 0.75rem;
  border-top: 1px solid rgba(var(--tint-rgb), 0.05);
}

.selection-count {
  color: var(--text-soft, #b3bbc4);
  font-size: 0.84rem;
}

.danger-btn {
  height: 38px;
  padding: 0 0.9rem;
  border-radius: 10px;
  font-size: 0.84rem;
  font-weight: 600;
  cursor: pointer;
  background: rgba(220, 38, 38, 0.14);
  color: var(--negative-text);
  border: 1px solid rgba(220, 38, 38, 0.22);
  transition: background 140ms ease;
}

.danger-btn:hover {
  background: rgba(220, 38, 38, 0.2);
}

.danger-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.sort-field {
  display: flex;
  align-items: center;
  gap: 0.55rem;
}

.sort-label {
  color: var(--text-dim, #8a939d);
  font-size: 0.76rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.sort-select,
.sort-direction-btn {
  height: 32px;
  border-radius: 10px;
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--text, #eef1f3);
  padding: 0 0.7rem;
  font-size: 0.8rem;
}

.sort-select {
  min-width: 168px;
  outline: none;
}

.sort-select:focus,
.sort-direction-btn:focus-visible {
  border-color: rgba(143, 168, 160, 0.34);
  box-shadow: 0 0 0 3px rgba(143, 168, 160, 0.12);
}

.sort-direction-btn {
  cursor: pointer;
  transition:
    background 140ms ease,
    border-color 140ms ease,
    transform 140ms ease;
}

.sort-direction-btn:hover {
  background: rgba(var(--tint-rgb), 0.07);
  border-color: rgba(var(--tint-rgb), 0.14);
  transform: translateY(-1px);
}

.sort-arrow-btn {
  min-width: 32px;
  padding: 0;
  text-align: center;
  font-size: 0.95rem;
  line-height: 1;
}

.reset-filters-btn {
  height: 32px;
  min-height: 32px;
  padding: 0 0.7rem;
  border-radius: 10px;
  font-size: 0.8rem;
}

.page-size-custom-input {
  width: 90px;
  min-width: 0;
}

.pagination-bar {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  padding-top: 0.4rem;
}

.pagination-status {
  color: var(--text-soft, #b3bbc4);
  font-size: 0.82rem;
}

.incomes-list {
  display: grid;
  gap: 0.85rem;
  margin-top: 0.75rem;
}

.income-month-group {
  display: grid;
  gap: 0.65rem;
}

.income-month-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.15rem 0.15rem 0.05rem;
}

.income-month-heading h3 {
  margin: 0.2rem 0 0;
  color: var(--text, #eef1f3);
  font-size: 1rem;
  line-height: 1.15;
}

.income-month-total {
  flex-shrink: 0;
  background: rgba(143, 168, 160, 0.18);
  border-color: rgba(143, 168, 160, 0.24);
}

.income-source-group {
  display: grid;
  gap: 0.55rem;
  padding: 0.7rem;
  border-radius: 16px;
  background: rgba(var(--tint-rgb), 0.022);
  border: 1px solid rgba(var(--tint-rgb), 0.055);
}

.income-source-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding-bottom: 0.1rem;
  border-bottom: 1px solid rgba(var(--tint-rgb), 0.035);
}

.income-source-heading h4 {
  margin: 0.2rem 0 0;
  color: var(--text, #eef1f3);
  font-size: 0.9rem;
  line-height: 1.15;
}

.income-source-total {
  flex-shrink: 0;
  background: rgba(var(--tint-rgb), 0.055);
  border-color: rgba(var(--tint-rgb), 0.08);
}

.income-source-list {
  display: grid;
  gap: 0.5rem;
}

.income-table-wrap {
  max-height: 560px;
  overflow: auto;
  border-radius: 14px;
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.06));
}

.income-table {
  display: table;
  width: 100%;
  table-layout: fixed;
  border-collapse: collapse;
  font-size: 0.85rem;
}

.income-table col.col-checkbox {
  width: 36px;
}

.income-table col.col-actions {
  width: 78px;
}

.income-table col.col-flex {
  width: auto;
}

.income-table td {
  overflow: hidden;
  text-overflow: ellipsis;
}

.income-table th {
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

.income-table th.sortable {
  cursor: pointer;
  user-select: none;
  transition: color 140ms ease;
}

.income-table th.sortable:hover {
  color: var(--text, #eef1f3);
}

.income-table .sort-indicator {
  display: inline-block;
  width: 0.7em;
  font-size: 0.7rem;
  color: var(--text-soft, #b3bbc4);
}

.income-table td {
  padding: 0.55rem 0.6rem;
  border-top: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
  color: var(--text, #eef1f3);
  vertical-align: middle;
  white-space: nowrap;
}

.income-table tr:hover {
  background: rgba(var(--tint-rgb), 0.03);
}

.income-table tr.is-selected-row {
  background: rgba(143, 168, 160, 0.08);
}

.income-table .checkbox-col {
  position: sticky;
  left: 0;
  z-index: 2;
  background: var(--bg-elevated, rgba(20, 23, 27, 0.98));
  width: 36px;
}

.income-table td.checkbox-col {
  background: var(--bg-soft, #262a2f);
}

.income-table .actions-col-inner {
  display: flex;
  gap: 0.4rem;
  align-items: center;
}

.income-table th.is-frozen,
.income-table td.is-frozen {
  position: sticky;
  z-index: 1;
  background: var(--bg-elevated, rgba(20, 23, 27, 0.98));
  box-shadow: 2px 0 4px rgba(0, 0, 0, 0.12);
}

.income-table td.is-frozen {
  background: var(--bg-soft, #262a2f);
}

.income-table .label-cell {
  overflow: hidden;
  text-overflow: ellipsis;
}

.income-table .icon-btn {
  width: 26px;
  height: 26px;
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  border-radius: 8px;
  background: rgba(var(--tint-rgb), 0.035);
  cursor: pointer;
  font-size: 0.78rem;
  line-height: 1;
}

.income-table .icon-btn:hover {
  background: rgba(var(--tint-rgb), 0.08);
}

.income-table .icon-btn-danger {
  background: rgba(220, 38, 38, 0.1);
  border-color: rgba(220, 38, 38, 0.14);
}

.income-table .icon-btn-danger:hover {
  background: rgba(220, 38, 38, 0.16);
}

.amount-positive {
  color: var(--positive-text, #bfe0c9);
  font-weight: 600;
}

.reconciled-badge {
  display: inline-block;
  padding: 0.2rem 0.55rem;
  border-radius: 999px;
  font-size: 0.72rem;
  font-weight: 600;
  white-space: nowrap;
}

.reconciled-badge.is-reconciled {
  background: color-mix(in srgb, var(--bg-elevated, #2c3137) 78%, #22c55e 22%);
  color: var(--positive-text, #bfe0c9);
}

.reconciled-badge.is-pending {
  color: var(--text-dim, #8a939d);
}

.income-amount {
  flex-shrink: 0;
  padding: 0.3rem 0.55rem;
  border-radius: 999px;
  background: rgba(143, 168, 160, 0.14);
  color: var(--positive-text);
  border: 1px solid rgba(143, 168, 160, 0.2);
  font-size: 0.76rem;
  font-weight: 700;
}

.income-source-footer {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding-top: 0.2rem;
}

.income-more-btn,
.income-less-btn {
  height: 30px;
  padding: 0 0.8rem;
  border-radius: 10px;
  font-size: 0.78rem;
  cursor: pointer;
}

.secondary-btn {
  border: 1px solid rgba(143, 168, 160, 0.18);
  background: rgba(143, 168, 160, 0.1);
  color: var(--positive-text);
}

.secondary-btn:hover {
  background: rgba(143, 168, 160, 0.16);
}

.ghost-btn {
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  background: rgba(var(--tint-rgb), 0.03);
  color: var(--text-soft, #b3bbc4);
}

.ghost-btn:hover {
  background: rgba(var(--tint-rgb), 0.06);
}

.ghost-btn:focus-visible,
.secondary-btn:focus-visible {
  outline: none;
  border-color: rgba(143, 168, 160, 0.42);
  box-shadow: 0 0 0 3px rgba(143, 168, 160, 0.14);
}

.incomes-state,
.empty-state {
  padding: 1.8rem 1.3rem;
  text-align: center;
}

.empty-state h2 {
  margin: 0.75rem 0 0.5rem;
  color: var(--text, #eef1f3);
}

.empty-state p,
.incomes-state {
  color: var(--text-soft, #b3bbc4);
}

.empty-state .primary-btn {
  margin-top: 0.95rem;
}

.empty-state-icon {
  width: 54px;
  height: 54px;
  margin: 0 auto;
  border-radius: 16px;
  display: grid;
  place-items: center;
  background: rgba(var(--tint-rgb), 0.06);
  color: #dbe6df;
  font-weight: 700;
  font-size: 1.05rem;
}

.eyebrow {
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.72rem;
}

@media (max-width: 1100px) {
  .incomes-stats-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .incomes-view {
    gap: 1rem;
  }

  .incomes-stats-grid {
    grid-template-columns: 1fr;
  }

  .incomes-list-card {
    padding: 1rem;
  }

  .incomes-panel-header,
  .income-month-header,
  .income-source-header,
  .sort-field,
  .income-source-footer {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
  }

  .income-source-header {
    gap: 0.6rem;
  }

  .incomes-toolbar {
    width: 100%;
    align-items: stretch;
  }

  .sort-select,
  .sort-direction-btn {
    width: 100%;
  }
}
</style>
