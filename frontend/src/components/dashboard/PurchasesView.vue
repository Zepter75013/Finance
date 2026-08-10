<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'

import PageHero from '../Common/PageHero.vue'
import ConfirmModal from '../Common/ConfirmModal.vue'
import ColumnVisibilityMenu from '../Common/ColumnVisibilityMenu.vue'
import PurchaseCharts from './PurchaseCharts.vue'
import PurchaseRowCard from './PurchaseRowCard.vue'
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
const { purchases, filteredPurchases, categoriesList, subCategoriesList } = storeToRefs(store)

// Contexte de la vue (tri, filtres, pagination) conservé au fil des
// navigations — sans ça, quitter l'écran puis y revenir remettait tout à
// zéro à chaque fois (la vue est démontée/remontée, pas juste masquée).
const CONTEXT_STORAGE_KEY = 'purchasesViewContext'

function loadStoredContext() {
  try {
    return JSON.parse(localStorage.getItem(CONTEXT_STORAGE_KEY) || '{}')
  } catch {
    return {}
  }
}

const storedContext = loadStoredContext()

const selectedPurchaseId = ref(null)
const sortBy = ref(storedContext.sortBy || 'date')
const sortDirection = ref(storedContext.sortDirection || 'desc')
const displayMode = ref(storedContext.displayMode || 'list')
const isGrouped = computed(() => displayMode.value === 'grouped')
const visibleCountByCategory = ref({})

const PURCHASE_TABLE_COLUMNS = [
  { key: 'date', label: 'Date', width: 100 },
  { key: 'merchant', label: 'Commerçant', width: 180 },
  { key: 'category', label: 'Catégorie', width: 160 },
  { key: 'subCategory', label: 'Sous-catégorie', width: 160 },
  { key: 'paymentMethod', label: 'Moyen de paiement', width: 150 },
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
} = useTableColumns('purchasesTableColumns', PURCHASE_TABLE_COLUMNS)
const activityRange = ref(storedContext.activityRange || 'all')
const activityCategory = ref(storedContext.activityCategory || 'all')
const activitySubCategory = ref(storedContext.activitySubCategory || 'all')
const activityReconciliation = ref(storedContext.activityReconciliation || 'all')

// La sous-catégorie dépend de la catégorie choisie — si elle change, l'ancien
// choix de sous-catégorie n'a plus forcément de sens.
watch(activityCategory, () => {
  activitySubCategory.value = 'all'
})
const checkedPurchaseIds = reactive(new Set())
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
  visibleCountByCategory.value = {}
  currentPage.value = 1
}

const isListCollapsed = ref(localStorage.getItem('purchasesListCollapsed') === '1')

function toggleListCollapsed() {
  isListCollapsed.value = !isListCollapsed.value
  localStorage.setItem('purchasesListCollapsed', isListCollapsed.value ? '1' : '0')
}

const periodOptions = [
  { value: '30d', label: '30 jours' },
  { value: '90d', label: '90 jours' },
  { value: '12m', label: '12 mois' },
  { value: 'ytd', label: 'Cette année' },
  { value: 'all', label: 'Tout' },
]

const categoryFilterOptions = computed(() => [
  { value: 'all', label: 'Toutes les catégories' },
  ...categoriesList.value
    .map((category) => ({ value: category.name, label: category.name }))
    .sort((a, b) => a.label.localeCompare(b.label, 'fr', { sensitivity: 'base' })),
])

// Ne propose que les sous-catégories qui appartiennent à la catégorie
// actuellement filtrée — d'où le sélecteur ne s'affiche qu'une fois une
// catégorie précise choisie.
const subCategoryFilterOptions = computed(() => {
  if (activityCategory.value === 'all') return []

  const category = categoriesList.value.find((c) => c.name === activityCategory.value)
  if (!category) return []

  return [
    { value: 'all', label: 'Toutes les sous-catégories' },
    ...subCategoriesList.value
      .filter((sub) => Number(sub.categoryId) === Number(category.id))
      .map((sub) => ({ value: sub.name, label: sub.name }))
      .sort((a, b) => a.label.localeCompare(b.label, 'fr', { sensitivity: 'base' })),
  ]
})

const specificMonthOptions = computed(() => {
  const months = new Map()

  for (const purchase of filteredPurchases.value) {
    if (!purchase.date) continue
    const date = new Date(purchase.date)
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

const activityPurchases = computed(() => {
  return filteredPurchases.value.filter((purchase) => {
    if (!isWithinRange(purchase.date, activityRange.value)) return false

    if (activityCategory.value !== 'all') {
      const category = purchase.category?.trim() || 'Sans catégorie'
      if (category !== activityCategory.value) return false
    }

    if (activitySubCategory.value !== 'all') {
      const subCategory = purchase.subCategory?.trim() || 'Sans sous-catégorie'
      if (subCategory !== activitySubCategory.value) return false
    }

    if (activityReconciliation.value === 'reconciled' && !purchase.isReconciled) return false
    if (activityReconciliation.value === 'unreconciled' && purchase.isReconciled) return false

    return true
  })
})

const hasActivityFilters = computed(
  () =>
    activityRange.value !== 'all' ||
    activityCategory.value !== 'all' ||
    activitySubCategory.value !== 'all' ||
    activityReconciliation.value !== 'all'
)

// Les cartes de résumé reflètent les achats filtrés (et non plus toujours
// l'ensemble des données) — sinon elles ne correspondaient plus à ce qui
// est réellement affiché dans la liste dès qu'un filtre était actif.
const filteredTotalAmount = computed(() =>
  activityPurchases.value.reduce((sum, purchase) => sum + Number(purchase.amount || 0), 0)
)
const filteredPurchaseCount = computed(() => activityPurchases.value.length)
const filteredAverageAmount = computed(() =>
  filteredPurchaseCount.value > 0 ? filteredTotalAmount.value / filteredPurchaseCount.value : 0
)

function resetActivityFilters() {
  activityRange.value = 'all'
  activityCategory.value = 'all'
  activitySubCategory.value = 'all'
  activityReconciliation.value = 'all'
}

function formatMerchantLabel(value) {
  return value?.trim() || 'Sans commerçant'
}

function formatTableDate(value) {
  if (!value) return ''

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''

  const day = String(date.getDate()).padStart(2, '0')
  const month = String(date.getMonth() + 1).padStart(2, '0')

  return `${day}/${month}/${date.getFullYear()}`
}

function comparePurchaseItems(a, b) {
  if (sortBy.value === 'amount') {
    return Number(a.amount || 0) - Number(b.amount || 0)
  }

  if (sortBy.value === 'merchant') {
    return formatMerchantLabel(a.merchant).localeCompare(formatMerchantLabel(b.merchant), 'fr', {
      sensitivity: 'base',
    })
  }

  if (sortBy.value === 'category') {
    return (a.category || '').localeCompare(b.category || '', 'fr', { sensitivity: 'base' })
  }

  if (sortBy.value === 'subCategory') {
    return (a.subCategory || '').localeCompare(b.subCategory || '', 'fr', { sensitivity: 'base' })
  }

  if (sortBy.value === 'paymentMethod') {
    return (a.paymentMethod || '').localeCompare(b.paymentMethod || '', 'fr', { sensitivity: 'base' })
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

  return new Date(a.date) - new Date(b.date)
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

function sortPurchaseItems(items) {
  return [...items].sort((a, b) => {
    const result = comparePurchaseItems(a, b)
    return sortDirection.value === 'asc' ? result : -result
  })
}

const sortedActivityPurchases = computed(() => sortPurchaseItems(activityPurchases.value))

const totalPages = computed(() => {
  if (effectivePageSize.value === Infinity) return 1
  return Math.max(1, Math.ceil(sortedActivityPurchases.value.length / effectivePageSize.value))
})

const paginatedPurchases = computed(() => {
  if (effectivePageSize.value === Infinity) return sortedActivityPurchases.value

  const start = (currentPage.value - 1) * effectivePageSize.value
  return sortedActivityPurchases.value.slice(start, start + effectivePageSize.value)
})

const canGoPreviousPage = computed(() => currentPage.value > 1)
const canGoNextPage = computed(() => currentPage.value < totalPages.value)

function goToPreviousPage() {
  if (canGoPreviousPage.value) currentPage.value -= 1
}

function goToNextPage() {
  if (canGoNextPage.value) currentPage.value += 1
}

watch([activityRange, activityCategory, activityReconciliation, sortBy, sortDirection], () => {
  currentPage.value = 1
})

watch(totalPages, (pages) => {
  if (currentPage.value > pages) currentPage.value = pages
})

const groupedPurchases = computed(() => {
  const monthGroups = new Map()

  for (const purchase of activityPurchases.value) {
    if (!purchase.date) continue

    const date = new Date(purchase.date)
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
        categoriesMap: new Map(),
      })
    }

    const monthGroup = monthGroups.get(monthKey)
    const categoryKey = purchase.category?.trim() || 'Sans catégorie'
    const groupCategoryKey = `${monthKey}-${categoryKey}`

    if (!monthGroup.categoriesMap.has(groupCategoryKey)) {
      monthGroup.categoriesMap.set(groupCategoryKey, {
        key: groupCategoryKey,
        label: categoryKey,
        total: 0,
        items: [],
      })
    }

    const categoryGroup = monthGroup.categoriesMap.get(groupCategoryKey)
    categoryGroup.items.push(purchase)
    categoryGroup.total += Number(purchase.amount || 0)
    monthGroup.total += Number(purchase.amount || 0)
  }

  return Array.from(monthGroups.values())
    .sort((a, b) => b.key.localeCompare(a.key))
    .map((monthGroup) => {
      const categories = Array.from(monthGroup.categoriesMap.values())
        .sort((a, b) => a.label.localeCompare(b.label, 'fr', { sensitivity: 'base' }))
        .map((category) => {
          const sortedItems = sortPurchaseItems(category.items)
          const baseCount =
            effectivePageSize.value === Infinity ? sortedItems.length : effectivePageSize.value
          const visibleCount = visibleCountByCategory.value[category.key] ?? baseCount

          return {
            ...category,
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
        categories,
      }
    })
})

// Basé sur l'ensemble des achats (pas la liste déjà filtrée par catégorie
// active) — comme pour les revenus, sinon un filtre de catégorie resté actif
// (ex: venu de Catégories) masquait ce bloc même quand il y a bien des achats.
const hasPurchaseData = computed(() => purchases.value.length > 0)

const hasVisibleActivity = computed(() =>
  isGrouped.value ? groupedPurchases.value.length > 0 : sortedActivityPurchases.value.length > 0
)

function formatCurrency(value) {
  return new Intl.NumberFormat('fr-FR', {
    style: 'currency',
    currency: 'EUR',
    minimumFractionDigits: 2,
  }).format(Number(value || 0))
}

function selectPurchase(purchase) {
  selectedPurchaseId.value = purchase.id
}

function isChecked(id) {
  return checkedPurchaseIds.has(id)
}

function toggleChecked(purchase) {
  if (checkedPurchaseIds.has(purchase.id)) checkedPurchaseIds.delete(purchase.id)
  else checkedPurchaseIds.add(purchase.id)
}

const hasSelection = computed(() => checkedPurchaseIds.size > 0)

const isAllSelected = computed(() => {
  return (
    activityPurchases.value.length > 0 &&
    activityPurchases.value.every((purchase) => checkedPurchaseIds.has(purchase.id))
  )
})

function selectAll() {
  for (const purchase of activityPurchases.value) checkedPurchaseIds.add(purchase.id)
}

function clearSelection() {
  checkedPurchaseIds.clear()
}

function toggleSelectAll() {
  if (isAllSelected.value) clearSelection()
  else selectAll()
}

function requestDeleteSelected() {
  if (!checkedPurchaseIds.size) return
  isBulkDeleteConfirmOpen.value = true
}

async function deleteSelected() {
  const ids = Array.from(checkedPurchaseIds)
  if (!ids.length) return

  isBulkDeleting.value = true

  let successCount = 0
  let failCount = 0

  for (const id of ids) {
    try {
      await store.removePurchase(id)
      checkedPurchaseIds.delete(id)
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

function showMore(categoryKey) {
  const step = effectivePageSize.value === Infinity ? DEFAULT_VISIBLE_ITEMS : effectivePageSize.value
  const current = visibleCountByCategory.value[categoryKey] ?? step

  visibleCountByCategory.value = {
    ...visibleCountByCategory.value,
    [categoryKey]: current + step,
  }
}

function showLess(categoryKey) {
  const step = effectivePageSize.value === Infinity ? DEFAULT_VISIBLE_ITEMS : effectivePageSize.value

  visibleCountByCategory.value = {
    ...visibleCountByCategory.value,
    [categoryKey]: step,
  }
}

function handleCreate() {
  emit('create')
}

function handleEdit(purchase) {
  emit('edit', purchase)
}

function handleDelete(purchase) {
  emit('delete', purchase)
}
</script>

<template>
  <main class="dashboard-content purchases-view">
    <PageHero
      eyebrow="Sorties d’argent"
      title="Achats"
      description="Enregistre tes dépenses au quotidien pour suivre où part ton argent, catégorie par catégorie."
    >
      <template #actions>
        <button class="primary-btn" type="button" @click="handleCreate">
          + Nouvel achat
        </button>
      </template>
    </PageHero>

    <section class="purchases-stats-grid">
      <article class="panel stat-card">
        <p class="eyebrow">Total dépensé</p>
        <h2>{{ formatCurrency(filteredTotalAmount) }}</h2>
        <p>{{ hasActivityFilters ? 'Montant cumulé (filtré)' : 'Montant cumulé enregistré' }}</p>
      </article>

      <article class="panel stat-card">
        <p class="eyebrow">Nombre d’achats</p>
        <h2>{{ filteredPurchaseCount }}</h2>
        <p>{{ hasActivityFilters ? 'Achats correspondant au filtre' : 'Achats suivis dans l’application' }}</p>
      </article>

      <article class="panel stat-card">
        <p class="eyebrow">Montant moyen</p>
        <h2>{{ formatCurrency(filteredAverageAmount) }}</h2>
        <p>Moyenne par achat{{ hasActivityFilters ? ' (filtré)' : ' saisi' }}</p>
      </article>
    </section>

    <PurchaseCharts v-if="!props.isLoading && hasPurchaseData" />

    <section v-if="props.isLoading" class="panel purchases-state">
      Chargement des achats...
    </section>

    <section v-else-if="!hasPurchaseData" class="panel empty-state">
      <div class="empty-state-icon">€</div>
      <h2>Aucun achat pour le moment</h2>
      <p>
        Ajoute ton premier achat pour commencer à suivre tes dépenses par catégorie.
      </p>
      <button class="primary-btn" type="button" @click="handleCreate">
        Ajouter un achat
      </button>
    </section>

    <section v-else-if="!hasVisibleActivity" class="panel empty-state">
      <div class="empty-state-icon">€</div>
      <h2>Aucun résultat pour ces filtres</h2>
      <p>
        Aucun achat ne correspond à la période et à la catégorie sélectionnées.
      </p>
      <button class="ghost-btn" type="button" @click="resetActivityFilters">
        Réinitialiser les filtres
      </button>
    </section>

    <section v-else class="purchases-layout">
      <article class="panel purchases-list-card">
        <div class="panel-header purchases-panel-header">
          <div>
            <p class="eyebrow">Activité récente</p>
            <h2>
              {{
                displayMode === 'grouped'
                  ? 'Achats par mois et par catégorie'
                  : displayMode === 'table'
                    ? 'Tableau des achats'
                    : 'Liste des achats'
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

        <div v-if="!isListCollapsed" class="purchases-toolbar">
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
                <option value="merchant">Commerçant</option>
                <option value="category">Catégorie</option>
                <option value="subCategory">Sous-catégorie</option>
                <option value="paymentMethod">Moyen de paiement</option>
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
              :columns="PURCHASE_TABLE_COLUMNS"
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

        <div v-if="!isListCollapsed" class="purchases-selection-bar">
          <button class="ghost-btn" type="button" @click="toggleSelectAll">
            {{ isAllSelected ? 'Tout désélectionner' : 'Tout sélectionner' }}
          </button>

          <span class="selection-count">
            {{ checkedPurchaseIds.size }} / {{ filteredPurchaseCount }} sélectionné{{ checkedPurchaseIds.size > 1 ? 's' : '' }}
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

        <div v-if="!isListCollapsed" class="purchases-list">
          <template v-if="isGrouped">
            <section
              v-for="group in groupedPurchases"
              :key="group.key"
              class="purchase-month-group"
            >
              <div class="purchase-month-header">
                <div class="purchase-month-heading">
                  <p class="eyebrow">Période</p>
                  <h3>{{ group.label }}</h3>
                </div>

                <div class="purchase-amount purchase-month-total">
                  {{ formatCurrency(group.total) }}
                </div>
              </div>

              <section
                v-for="category in group.categories"
                :key="category.key"
                class="purchase-category-group"
              >
                <div class="purchase-category-header">
                  <div class="purchase-category-heading">
                    <p class="eyebrow">Catégorie</p>
                    <h4>{{ category.label }}</h4>
                  </div>

                  <div class="purchase-amount purchase-category-total">
                    {{ formatCurrency(category.total) }}
                  </div>
                </div>

                <div class="purchase-category-list">
                  <PurchaseRowCard
                    v-for="purchase in category.visibleItems"
                    :key="purchase.id"
                    :purchase="purchase"
                    :selected="selectedPurchaseId === purchase.id"
                    :checked="isChecked(purchase.id)"
                    @select="selectPurchase"
                    @edit="handleEdit"
                    @delete="handleDelete"
                    @toggle-check="toggleChecked"
                  />
                </div>

                <div
                  v-if="category.hasMore || category.canCollapse"
                  class="purchase-category-footer"
                >
                  <button
                    v-if="category.hasMore"
                    class="secondary-btn purchase-more-btn"
                    type="button"
                    @click="showMore(category.key)"
                  >
                    Voir plus
                  </button>

                  <button
                    v-if="category.canCollapse"
                    class="ghost-btn purchase-less-btn"
                    type="button"
                    @click="showLess(category.key)"
                  >
                    Voir moins
                  </button>
                </div>
              </section>
            </section>
          </template>

          <template v-else-if="displayMode === 'table'">
            <div class="purchase-table-wrap">
              <table class="purchase-table">
                <colgroup>
                  <col class="col-checkbox" />
                  <col v-if="isColumnVisible('date')" style="width: 100px" />
                  <col v-if="isColumnVisible('merchant')" class="col-flex" />
                  <col v-if="isColumnVisible('category')" style="width: 160px" />
                  <col v-if="isColumnVisible('subCategory')" style="width: 160px" />
                  <col v-if="isColumnVisible('paymentMethod')" style="width: 150px" />
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
                      v-if="isColumnVisible('merchant')"
                      class="sortable"
                      :class="{ 'is-frozen': isColumnFrozen('merchant') }"
                      :style="columnStyle('merchant')"
                      @click="setSort('merchant')"
                    >
                      Commerçant <span class="sort-indicator">{{ sortIndicator('merchant') }}</span>
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
                      v-if="isColumnVisible('paymentMethod')"
                      class="sortable"
                      :class="{ 'is-frozen': isColumnFrozen('paymentMethod') }"
                      :style="columnStyle('paymentMethod')"
                      @click="setSort('paymentMethod')"
                    >
                      Moyen de paiement <span class="sort-indicator">{{ sortIndicator('paymentMethod') }}</span>
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
                    v-for="purchase in paginatedPurchases"
                    :key="purchase.id"
                    :class="{ 'is-selected-row': selectedPurchaseId === purchase.id }"
                    @dblclick="handleEdit(purchase)"
                  >
                    <td class="checkbox-col">
                      <input
                        type="checkbox"
                        :checked="isChecked(purchase.id)"
                        @click.stop
                        @change="toggleChecked(purchase)"
                      />
                    </td>
                    <td v-if="isColumnVisible('date')" :class="{ 'is-frozen': isColumnFrozen('date') }" :style="columnStyle('date')">
                      {{ formatTableDate(purchase.date) }}
                    </td>
                    <td
                      v-if="isColumnVisible('merchant')"
                      class="label-cell"
                      :class="{ 'is-frozen': isColumnFrozen('merchant') }"
                      :style="columnStyle('merchant')"
                    >
                      {{ formatMerchantLabel(purchase.merchant) }}
                    </td>
                    <td v-if="isColumnVisible('category')" :class="{ 'is-frozen': isColumnFrozen('category') }" :style="columnStyle('category')">
                      {{ purchase.category || '—' }}
                    </td>
                    <td v-if="isColumnVisible('subCategory')" :class="{ 'is-frozen': isColumnFrozen('subCategory') }" :style="columnStyle('subCategory')">
                      {{ purchase.subCategory || '—' }}
                    </td>
                    <td v-if="isColumnVisible('paymentMethod')" :class="{ 'is-frozen': isColumnFrozen('paymentMethod') }" :style="columnStyle('paymentMethod')">
                      {{ purchase.paymentMethod || '—' }}
                    </td>
                    <td v-if="isColumnVisible('reference')" :class="{ 'is-frozen': isColumnFrozen('reference') }" :style="columnStyle('reference')">
                      {{ purchase.reference || '—' }}
                    </td>
                    <td v-if="isColumnVisible('statementReference')" :class="{ 'is-frozen': isColumnFrozen('statementReference') }" :style="columnStyle('statementReference')">
                      {{ purchase.statementReference || '—' }}
                    </td>
                    <td v-if="isColumnVisible('isReconciled')" :class="{ 'is-frozen': isColumnFrozen('isReconciled') }" :style="columnStyle('isReconciled')">
                      <span class="reconciled-badge" :class="purchase.isReconciled ? 'is-reconciled' : 'is-pending'">
                        {{ purchase.isReconciled ? '✓ Pointé' : 'Non pointé' }}
                      </span>
                    </td>
                    <td
                      v-if="isColumnVisible('amount')"
                      class="amount-negative"
                      :class="{ 'is-frozen': isColumnFrozen('amount') }"
                      :style="columnStyle('amount')"
                    >
                      −{{ formatCurrency(purchase.amount) }}
                    </td>
                    <td class="actions-col">
                      <div class="actions-col-inner">
                        <button class="icon-btn" type="button" title="Modifier" @click="handleEdit(purchase)">✏️</button>
                        <button class="icon-btn icon-btn-danger" type="button" title="Supprimer" @click="handleDelete(purchase)">🗑️</button>
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
            <div class="purchase-category-list">
              <PurchaseRowCard
                v-for="purchase in paginatedPurchases"
                :key="purchase.id"
                :purchase="purchase"
                :selected="selectedPurchaseId === purchase.id"
                :checked="isChecked(purchase.id)"
                @select="selectPurchase"
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
      :title="`Supprimer ${checkedPurchaseIds.size} achat${checkedPurchaseIds.size > 1 ? 's' : ''} ?`"
      :message="`Tu es sur le point de supprimer définitivement ${checkedPurchaseIds.size} achat${checkedPurchaseIds.size > 1 ? 's' : ''} sélectionné${checkedPurchaseIds.size > 1 ? 's' : ''}.`"
      note="Vérifie ta sélection avant de confirmer : cette action est irréversible."
      confirm-label="Supprimer la sélection"
      :is-processing="isBulkDeleting"
      @confirm="deleteSelected"
    />
  </main>
</template>

<style scoped>
.purchases-view {
  display: grid;
  gap: 0.9rem;
}

.purchases-stats-grid {
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

.purchases-layout {
  display: grid;
  grid-template-columns: 1fr;
}

.purchases-list-card {
  min-height: 100%;
  padding: 0.95rem;
}

.purchases-panel-header {
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

.purchases-toolbar {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  flex-wrap: wrap;
}

.purchases-selection-bar {
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
  transition: background 140ms ease, border-color 140ms ease, transform 140ms ease;
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

.purchases-list {
  display: grid;
  gap: 0.85rem;
  margin-top: 0.75rem;
}

.purchase-month-group {
  display: grid;
  gap: 0.65rem;
}

.purchase-month-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.15rem 0.15rem 0.05rem;
}

.purchase-month-heading h3 {
  margin: 0.2rem 0 0;
  color: var(--text, #eef1f3);
  font-size: 1rem;
  line-height: 1.15;
}

.purchase-month-total {
  flex-shrink: 0;
  background: rgba(143, 168, 160, 0.18);
  border-color: rgba(143, 168, 160, 0.24);
}

.purchase-category-group {
  display: grid;
  gap: 0.55rem;
  padding: 0.7rem;
  border-radius: 16px;
  background: rgba(var(--tint-rgb), 0.022);
  border: 1px solid rgba(var(--tint-rgb), 0.055);
}

.purchase-category-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding-bottom: 0.1rem;
  border-bottom: 1px solid rgba(var(--tint-rgb), 0.035);
}

.purchase-category-heading h4 {
  margin: 0.2rem 0 0;
  color: var(--text, #eef1f3);
  font-size: 0.9rem;
  line-height: 1.15;
}

.purchase-category-total {
  flex-shrink: 0;
  background: rgba(var(--tint-rgb), 0.055);
  border-color: rgba(var(--tint-rgb), 0.08);
}

.purchase-category-list {
  display: grid;
  gap: 0.5rem;
}

.purchase-table-wrap {
  max-height: 560px;
  overflow: auto;
  border-radius: 14px;
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.06));
}

.purchase-table {
  display: table;
  width: 100%;
  table-layout: fixed;
  border-collapse: collapse;
  font-size: 0.85rem;
}

.purchase-table col.col-checkbox {
  width: 36px;
}

.purchase-table col.col-actions {
  width: 78px;
}

.purchase-table col.col-flex {
  width: auto;
}

.purchase-table td {
  overflow: hidden;
  text-overflow: ellipsis;
}

.purchase-table th {
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

.purchase-table th.sortable {
  cursor: pointer;
  user-select: none;
  transition: color 140ms ease;
}

.purchase-table th.sortable:hover {
  color: var(--text, #eef1f3);
}

.purchase-table .sort-indicator {
  display: inline-block;
  width: 0.7em;
  font-size: 0.7rem;
  color: var(--text-soft, #b3bbc4);
}

.purchase-table td {
  padding: 0.55rem 0.6rem;
  border-top: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
  color: var(--text, #eef1f3);
  vertical-align: middle;
  white-space: nowrap;
}

.purchase-table tr:hover {
  background: rgba(var(--tint-rgb), 0.03);
}

.purchase-table tr.is-selected-row {
  background: rgba(143, 168, 160, 0.08);
}

.purchase-table .checkbox-col {
  position: sticky;
  left: 0;
  z-index: 2;
  background: var(--bg-elevated, rgba(20, 23, 27, 0.98));
  width: 36px;
}

.purchase-table td.checkbox-col {
  background: var(--bg-soft, #262a2f);
}

.purchase-table .actions-col-inner {
  display: flex;
  gap: 0.4rem;
  align-items: center;
}

.purchase-table th.is-frozen,
.purchase-table td.is-frozen {
  position: sticky;
  z-index: 1;
  background: var(--bg-elevated, rgba(20, 23, 27, 0.98));
  box-shadow: 2px 0 4px rgba(0, 0, 0, 0.12);
}

.purchase-table td.is-frozen {
  background: var(--bg-soft, #262a2f);
}

.purchase-table .label-cell {
  overflow: hidden;
  text-overflow: ellipsis;
}

.purchase-table .icon-btn {
  width: 26px;
  height: 26px;
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  border-radius: 8px;
  background: rgba(var(--tint-rgb), 0.035);
  cursor: pointer;
  font-size: 0.78rem;
  line-height: 1;
}

.purchase-table .icon-btn:hover {
  background: rgba(var(--tint-rgb), 0.08);
}

.purchase-table .icon-btn-danger {
  background: rgba(220, 38, 38, 0.1);
  border-color: rgba(220, 38, 38, 0.14);
}

.purchase-table .icon-btn-danger:hover {
  background: rgba(220, 38, 38, 0.16);
}

.amount-negative {
  color: var(--negative-text, #e1b4b4);
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

.purchase-amount {
  flex-shrink: 0;
  padding: 0.3rem 0.55rem;
  border-radius: 999px;
  background: rgba(143, 168, 160, 0.14);
  color: var(--positive-text);
  border: 1px solid rgba(143, 168, 160, 0.2);
  font-size: 0.76rem;
  font-weight: 700;
}

.purchase-category-footer {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding-top: 0.2rem;
}

.purchase-more-btn,
.purchase-less-btn {
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

.purchases-state,
.empty-state {
  padding: 1.8rem 1.3rem;
  text-align: center;
}

.empty-state h2 {
  margin: 0.75rem 0 0.5rem;
  color: var(--text, #eef1f3);
}

.empty-state p,
.purchases-state {
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
  .purchases-stats-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .purchases-view {
    gap: 1rem;
  }

  .purchases-stats-grid {
    grid-template-columns: 1fr;
  }

  .purchases-list-card {
    padding: 1rem;
  }

  .purchases-panel-header,
  .purchase-month-header,
  .purchase-category-header,
  .sort-field,
  .purchase-category-footer {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
  }

  .purchase-category-header {
    gap: 0.6rem;
  }

  .purchases-toolbar {
    width: 100%;
    align-items: stretch;
  }

  .sort-select,
  .sort-direction-btn {
    width: 100%;
  }
}
</style>
