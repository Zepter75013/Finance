<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import PageHero from '../Common/PageHero.vue'
import ColumnVisibilityMenu from '../Common/ColumnVisibilityMenu.vue'
import ConfirmModal from '../Common/ConfirmModal.vue'
import { usePurchasesStore } from '../../stores/purchases'
import { useTableColumns } from '../../composables/useTableColumns'
import {
  fetchStatements,
  saveStatement,
  deleteStatement,
  setStatementLock,
  fetchStatementPdfs,
  uploadStatementPdf,
  viewStatementPdf,
  deleteStatementPdf,
} from '../../services/statements'
import { formatCurrency, formatDate } from '../../utils/format'

const store = usePurchasesStore()
const { purchases, incomes } = storeToRefs(store)

// Même compte actif que les écrans Achats/Revenus/Import (partagé via le
// store) — chaque compte a sa propre suite de relevés et son propre solde
// enchaîné, complètement indépendants des autres comptes. Passe par l'action
// du store pour que le changement soit bien persisté en localStorage.
const activeAccountId = computed({
  get: () => store.activeAccountId,
  set: (value) => store.setActiveAccountId(value),
})

const TABLE_COLUMNS = [
  { key: 'dateCompta', label: 'Date compta', width: 100 },
  { key: 'type', label: 'Type', width: 90 },
  { key: 'label', label: 'Libellé', width: 220 },
  { key: 'meta', label: 'Catégorie / Source', width: 170 },
  { key: 'reference', label: 'Référence', width: 140 },
  { key: 'dateOperation', label: 'Date opération', width: 120 },
  { key: 'dateValeur', label: 'Date valeur', width: 100 },
  { key: 'amount', label: 'Montant', width: 110 },
]

const { isVisible, isFrozen, toggleVisible, toggleFrozen, columnStyle } = useTableColumns(
  'reconciliationColumns',
  TABLE_COLUMNS
)

const COLLAPSE_STORAGE_KEY = 'reconciliationCollapsedSections'

function loadStoredCollapsedSections() {
  try {
    return JSON.parse(localStorage.getItem(COLLAPSE_STORAGE_KEY) || '{}')
  } catch {
    return {}
  }
}

const collapsedSections = reactive({
  settings: false,
  statements: false,
  summary: false,
  operations: false,
  ...loadStoredCollapsedSections(),
})

function toggleSection(key) {
  collapsedSections[key] = !collapsedSections[key]
  localStorage.setItem(COLLAPSE_STORAGE_KEY, JSON.stringify(collapsedSections))
}

const STORAGE_KEY = 'reconciliationSettings'

function loadStoredSettings() {
  try {
    return JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}')
  } catch {
    return {}
  }
}

function defaultStartDate() {
  const now = new Date()
  return new Date(now.getFullYear(), now.getMonth(), 1).toISOString().slice(0, 10)
}

function defaultEndDate() {
  return new Date().toISOString().slice(0, 10)
}

const stored = loadStoredSettings()

const settings = reactive({
  statementNumber: stored.statementNumber || '',
  statementDate: stored.statementDate || '',
  startBalance: stored.startBalance ?? null,
  endBalance: stored.endBalance ?? null,
  periodStart: stored.periodStart || defaultStartDate(),
  periodEnd: stored.periodEnd || defaultEndDate(),
})

function persistSettings() {
  localStorage.setItem(
    STORAGE_KEY,
    JSON.stringify({
      statementNumber: settings.statementNumber,
      statementDate: settings.statementDate,
      startBalance: settings.startBalance,
      endBalance: settings.endBalance,
      periodStart: settings.periodStart,
      periodEnd: settings.periodEnd,
    })
  )
}

// Les relevés sont enregistrés en base (et non plus seulement dans le
// localStorage) pour garder un historique consultable : pouvoir revenir sur
// un relevé passé et vérifier s'il a bien été équilibré.
const savedStatements = ref([])
const isLoadingStatements = ref(false)
const isSavingStatement = ref(false)
const statementError = ref('')
const deleteStatementTarget = ref(null)
const isDeletingStatement = ref(false)

async function loadStatements() {
  if (!activeAccountId.value) {
    savedStatements.value = []
    return
  }

  isLoadingStatements.value = true

  try {
    savedStatements.value = await fetchStatements(activeAccountId.value)
    await Promise.all(savedStatements.value.map((item) => loadPdfsForStatement(item.id)))
  } catch (err) {
    statementError.value = err instanceof Error ? err.message : 'Impossible de charger les relevés.'
  } finally {
    isLoadingStatements.value = false
  }
}

// Recharge les relevés à chaque changement de compte. Au tout premier
// chargement, les réglages en cours (potentiellement un relevé non encore
// enregistré) sont laissés tels quels, comme avant l'ajout des comptes — ils
// ne sont resynchronisés sur le dernier relevé du compte que lors d'un vrai
// changement de compte explicite.
let isInitialAccountLoad = true

watch(
  activeAccountId,
  async (accountId) => {
    if (!accountId) return

    const wasInitial = isInitialAccountLoad
    isInitialAccountLoad = false

    await loadStatements()

    if (!wasInitial) {
      if (savedStatements.value[0]) {
        loadStatementIntoSettings(savedStatements.value[0])
      } else {
        startNewStatement()
      }
    }
  },
  { immediate: true }
)

const currentSavedStatement = computed(() => {
  const reference = settings.statementNumber.trim()
  if (!reference) return null

  return savedStatements.value.find((item) => item.reference === reference) || null
})

const isCurrentStatementLocked = computed(() => Boolean(currentSavedStatement.value?.is_locked))

// Le bouton "Enregistrer" ne doit rien faire si le relevé actuel est déjà
// sauvegardé à l'identique — seule une vraie modification (ou un tout nouveau
// numéro de relevé) doit permettre d'enregistrer.
const settingsMatchSavedStatement = computed(() => {
  const saved = currentSavedStatement.value
  if (!saved) return false

  return (
    (settings.statementDate || '') === (saved.statement_date || '') &&
    (settings.periodStart || '') === (saved.period_start || '') &&
    (settings.periodEnd || '') === (saved.period_end || '') &&
    Number(settings.startBalance || 0) === Number(saved.start_balance || 0) &&
    Number(settings.endBalance || 0) === Number(saved.end_balance || 0)
  )
})

const canSaveStatement = computed(() => {
  if (!settings.statementNumber.trim()) return false
  if (isCurrentStatementLocked.value) return false

  return !settingsMatchSavedStatement.value
})

async function toggleLock(item) {
  if (!item) return

  statementError.value = ''

  try {
    const updated = await setStatementLock(item.id, !item.is_locked)
    const index = savedStatements.value.findIndex((s) => s.id === updated.id)
    if (index !== -1) savedStatements.value.splice(index, 1, updated)
  } catch (err) {
    statementError.value = err instanceof Error ? err.message : 'Impossible de modifier le verrouillage.'
  }
}

// --- PDF du relevé scanné : upload via icône ou glisser-déposer sur la ligne.
// Un relevé peut avoir plusieurs PDF (recto/verso, pages scannées séparément…).
const pdfFileInputs = ref({})
const uploadingPdfId = ref(null)
const draggingOverId = ref(null)
const statementPdfs = reactive({})

function setPdfFileInputRef(el, itemId) {
  if (el) pdfFileInputs.value[itemId] = el
}

function triggerPdfFilePicker(item) {
  pdfFileInputs.value[item.id]?.click()
}

async function loadPdfsForStatement(statementId) {
  try {
    statementPdfs[statementId] = await fetchStatementPdfs(statementId)
  } catch {
    statementPdfs[statementId] = []
  }
}

function pdfsForStatement(item) {
  return statementPdfs[item.id] || []
}

async function uploadPdfForStatement(item, file) {
  if (!file) return

  if (file.type !== 'application/pdf' && !file.name.toLowerCase().endsWith('.pdf')) {
    statementError.value = 'Seuls les fichiers PDF sont acceptés.'
    return
  }

  statementError.value = ''
  uploadingPdfId.value = item.id

  try {
    const created = await uploadStatementPdf(item.id, file)
    statementPdfs[item.id] = [...(statementPdfs[item.id] || []), created]
  } catch (err) {
    statementError.value = err instanceof Error ? err.message : 'Impossible d’enregistrer le PDF.'
  } finally {
    uploadingPdfId.value = null
  }
}

function handlePdfFileInputChange(item, event) {
  const file = event.target.files?.[0]
  uploadPdfForStatement(item, file)
  event.target.value = ''
}

function handleStatementRowDrop(item, event) {
  draggingOverId.value = null
  const file = event.dataTransfer?.files?.[0]
  uploadPdfForStatement(item, file)
}

async function openStatementPdf(item, pdf) {
  statementError.value = ''

  try {
    await viewStatementPdf(item.id, pdf.id)
  } catch (err) {
    statementError.value = err instanceof Error ? err.message : 'Impossible d’ouvrir le PDF.'
  }
}

async function removeStatementPdf(item, pdf) {
  statementError.value = ''

  try {
    await deleteStatementPdf(item.id, pdf.id)
    statementPdfs[item.id] = (statementPdfs[item.id] || []).filter((p) => p.id !== pdf.id)
  } catch (err) {
    statementError.value = err instanceof Error ? err.message : 'Impossible de supprimer le PDF.'
  }
}

function loadStatementIntoSettings(item) {
  settings.statementNumber = item.reference
  settings.statementDate = item.statement_date || ''
  settings.periodStart = item.period_start || settings.periodStart
  settings.periodEnd = item.period_end || settings.periodEnd
  settings.startBalance = item.start_balance
  settings.endBalance = item.end_balance
  persistSettings()
}

function addDays(dateStr, days) {
  if (!dateStr) return ''

  const date = new Date(dateStr)
  if (Number.isNaN(date.getTime())) return ''

  date.setDate(date.getDate() + days)
  return date.toISOString().slice(0, 10)
}

// Prépare un relevé vierge plutôt que de laisser l'utilisateur écraser
// manuellement le numéro du relevé courant — enchaîne sur le précédent
// (solde de départ = solde de fin du dernier relevé, période qui continue).
function startNewStatement() {
  statementError.value = ''

  const previous = savedStatements.value[0] || null

  settings.statementNumber = ''
  settings.statementDate = ''
  settings.startBalance = previous ? previous.end_balance : null
  settings.endBalance = null
  settings.periodStart = previous?.period_end ? addDays(previous.period_end, 1) : defaultStartDate()
  settings.periodEnd = defaultEndDate()

  persistSettings()
}

async function saveCurrentStatement() {
  statementError.value = ''
  const reference = settings.statementNumber.trim()

  if (!reference) {
    statementError.value = 'Renseigne un numéro de relevé avant de l’enregistrer.'
    return
  }

  isSavingStatement.value = true

  try {
    const saved = await saveStatement({
      account_id: activeAccountId.value,
      reference,
      statement_date: settings.statementDate || '',
      period_start: settings.periodStart || '',
      period_end: settings.periodEnd || '',
      start_balance: Number(settings.startBalance || 0),
      end_balance: Number(settings.endBalance || 0),
    })

    const index = savedStatements.value.findIndex((item) => item.id === saved.id)
    if (index === -1) savedStatements.value = [saved, ...savedStatements.value]
    else savedStatements.value.splice(index, 1, saved)
  } catch (err) {
    statementError.value = err instanceof Error ? err.message : 'Impossible d’enregistrer le relevé.'
  } finally {
    isSavingStatement.value = false
  }
}

function requestDeleteStatement(item) {
  if (item.is_locked) return

  statementError.value = ''
  deleteStatementTarget.value = item
}

async function confirmDeleteStatement() {
  if (!deleteStatementTarget.value) return

  isDeletingStatement.value = true

  try {
    await deleteStatement(deleteStatementTarget.value.id)
    savedStatements.value = savedStatements.value.filter(
      (item) => item.id !== deleteStatementTarget.value.id
    )
    deleteStatementTarget.value = null
  } catch (err) {
    statementError.value = err instanceof Error ? err.message : 'Impossible de supprimer le relevé.'
  } finally {
    isDeletingStatement.value = false
  }
}

// Contexte de la vue (filtre, recherche, tri) conservé au fil des
// navigations — sans ça, quitter l'écran puis y revenir remettait tout à
// zéro à chaque fois (la vue est démontée/remontée, pas juste masquée).
const VIEW_CONTEXT_STORAGE_KEY = 'reconciliationViewContext'

function loadStoredViewContext() {
  try {
    return JSON.parse(localStorage.getItem(VIEW_CONTEXT_STORAGE_KEY) || '{}')
  } catch {
    return {}
  }
}

const storedViewContext = loadStoredViewContext()

const visibilityFilter = ref(storedViewContext.visibilityFilter || 'all')
const isSavingId = ref(null)
const searchQuery = ref(storedViewContext.searchQuery || '')
const searchColumn = ref(storedViewContext.searchColumn || 'all')
const deleteTarget = ref(null)
const isDeleting = ref(false)
const deleteError = ref('')

const SEARCH_COLUMNS = [
  { value: 'all', label: 'Toutes les colonnes' },
  { value: 'dateCompta', label: 'Date compta' },
  { value: 'type', label: 'Type' },
  { value: 'label', label: 'Libell\u00e9' },
  { value: 'meta', label: 'Cat\u00e9gorie / Source' },
  { value: 'reference', label: 'R\u00e9f\u00e9rence' },
  { value: 'dateOperation', label: 'Date op\u00e9ration' },
  { value: 'dateValeur', label: 'Date valeur' },
  { value: 'amount', label: 'Montant' },
]

function normalizeSearchValue(value) {
  return String(value ?? '')
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .trim()
}



function truncatePdfName(name, maxLength = 16) {
  const trimmed = String(name ?? '').trim()
  return trimmed.length > maxLength ? `${trimmed.slice(0, maxLength - 1)}…` : trimmed
}

function getSearchableValue(row, column) {
  switch (column) {
    case 'dateCompta':
    case 'dateOperation':
    case 'dateValeur':
      return formatDate(row[column])
    case 'type':
      return row.type === 'achat' ? 'achat' : 'revenu'
    default:
      return row[column]
  }
}

const SORT_ACCESSORS = {
  dateCompta: (row) => row.dateCompta,
  dateOperation: (row) => row.dateOperation,
  dateValeur: (row) => row.dateValeur,
  type: (row) => row.type,
  label: (row) => row.label,
  reference: (row) => row.reference,
  meta: (row) => row.meta,
  amount: (row) => row.amount,
}

const sortColumn = ref(storedViewContext.sortColumn || 'dateOperation')
const sortDirection = ref(storedViewContext.sortDirection || 'asc')

function setSort(column) {
  if (!SORT_ACCESSORS[column]) return

  if (sortColumn.value === column) {
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortColumn.value = column
    sortDirection.value = 'asc'
  }
}

watch([visibilityFilter, searchQuery, searchColumn, sortColumn, sortDirection], () => {
  localStorage.setItem(
    VIEW_CONTEXT_STORAGE_KEY,
    JSON.stringify({
      visibilityFilter: visibilityFilter.value,
      searchQuery: searchQuery.value,
      searchColumn: searchColumn.value,
      sortColumn: sortColumn.value,
      sortDirection: sortDirection.value,
    })
  )
})

function sortIndicator(column) {
  if (sortColumn.value !== column) return ''
  return sortDirection.value === 'asc' ? '▲' : '▼'
}

const periodTransactions = computed(() => {
  const start = settings.periodStart
  const end = settings.periodEnd

  const achatRows = purchases.value
    .map((purchase) => {
      const dateCompta = purchase.date
      const dateOperation = purchase.operationDate || purchase.date
      const dateValeur = purchase.valueDate || ''

      return {
        id: `achat-${purchase.id}`,
        sourceId: purchase.id,
        type: 'achat',
        dateCompta,
        dateOperation,
        dateValeur,
        // Carte à débit différé : l'achat est daté du jour d'achat mais ne
        // débite le compte qu'à la date de valeur — c'est donc elle qui doit
        // déterminer à quel relevé (période) l'opération appartient, quand
        // la banque la fournit.
        datePeriode: dateValeur || dateOperation,
        label: purchase.merchant,
        reference: purchase.reference || '',
        meta: purchase.category,
        amount: -Math.abs(Number(purchase.amount || 0)),
        isReconciled: Boolean(purchase.isReconciled),
        statementReference: purchase.statementReference || '',
        raw: purchase,
      }
    })
    .filter((row) => (!start || row.datePeriode >= start) && (!end || row.datePeriode <= end))

  const revenuRows = incomes.value
    .map((income) => {
      const dateCompta = income.income_date
      const dateOperation = income.operationDate || income.income_date
      const dateValeur = income.valueDate || ''

      return {
        id: `revenu-${income.id}`,
        sourceId: income.id,
        type: 'revenu',
        dateCompta,
        dateOperation,
        dateValeur,
        datePeriode: dateValeur || dateOperation,
        label: income.source,
        reference: income.reference || '',
        meta: income.category,
        amount: Math.abs(Number(income.amount || 0)),
        isReconciled: Boolean(income.isReconciled),
        statementReference: income.statementReference || '',
        raw: income,
      }
    })
    .filter((row) => (!start || row.datePeriode >= start) && (!end || row.datePeriode <= end))

  const currentStatementNumber = settings.statementNumber.trim()

  // Une opération déjà pointée sur un AUTRE relevé (typiquement un débit
  // différé déjà rattaché au relevé précédent) ne doit pas réapparaître ici
  // sous prétexte que sa date de valeur tombe aussi dans cette période —
  // elle est déjà comptée ailleurs.
  const belongsToCurrentStatement = (row) =>
    !row.statementReference || row.statementReference === currentStatementNumber

  return [...achatRows, ...revenuRows].filter(belongsToCurrentStatement)
})

const visibleTransactions = computed(() => {
  let rows = periodTransactions.value

  if (visibilityFilter.value === 'reconciled') {
    rows = rows.filter((row) => row.isReconciled)
  } else if (visibilityFilter.value === 'unreconciled') {
    rows = rows.filter((row) => !row.isReconciled)
  }

  const normalizedQuery = normalizeSearchValue(searchQuery.value)

  if (normalizedQuery) {
    rows = rows.filter((row) => {
      if (searchColumn.value === 'all') {
        const searchableValues = [
          row.label,
          row.meta,
          row.reference,
          row.amount,
          row.type,
          formatDate(row.dateCompta),
          formatDate(row.dateOperation),
          formatDate(row.dateValeur),
        ]
        return normalizeSearchValue(searchableValues.join(' ')).includes(normalizedQuery)
      }

      return normalizeSearchValue(getSearchableValue(row, searchColumn.value)).includes(normalizedQuery)
    })
  }

  const getValue = SORT_ACCESSORS[sortColumn.value] || SORT_ACCESSORS.dateOperation
  const direction = sortDirection.value === 'desc' ? -1 : 1

  return [...rows].sort((a, b) => {
    const valueA = getValue(a)
    const valueB = getValue(b)

    if (valueA < valueB) return -1 * direction
    if (valueA > valueB) return 1 * direction
    return 0
  })
})

const reconciledCount = computed(
  () => periodTransactions.value.filter((row) => row.isReconciled).length
)

const reconciledTotal = computed(() => {
  return periodTransactions.value
    .filter((row) => row.isReconciled)
    .reduce((sum, row) => sum + row.amount, 0)
})

const computedBalance = computed(() => {
  return Number(settings.startBalance || 0) + reconciledTotal.value
})

const balanceGap = computed(() => computedBalance.value - Number(settings.endBalance || 0))

const isBalanced = computed(() => Math.abs(balanceGap.value) < 0.005)

async function toggleReconciled(row) {
  if (isCurrentStatementLocked.value) return

  isSavingId.value = row.id

  const nextReconciled = !row.isReconciled
  const patch = {
    isReconciled: nextReconciled,
    statementReference: nextReconciled ? settings.statementNumber.trim() : '',
  }

  try {
    if (row.type === 'achat') {
      await store.editPurchase(row.sourceId, { ...row.raw, ...patch })
    } else {
      await store.editIncome(row.sourceId, { ...row.raw, ...patch })
    }
  } catch {
    // L'état visuel restera synchronisé avec le store même en cas d'échec réseau ponctuel.
  } finally {
    isSavingId.value = null
  }
}

async function setAllVisible(reconciled) {
  if (isCurrentStatementLocked.value) return

  const rows = visibleTransactions.value.filter((row) => row.isReconciled !== reconciled)

  for (const row of rows) {
    await toggleReconciled(row)
  }
}

// Des opérations pointées avant l'ajout du numéro de relevé (ou pointées via
// "Tout pointer" sans numéro renseigné) n'ont pas ce champ rempli. Ce bouton
// permet de le renseigner rétroactivement sans avoir à dépointer/repointer
// chaque ligne.
const missingStatementCount = computed(
  () => visibleTransactions.value.filter((row) => row.isReconciled && !row.statementReference).length
)

async function applyStatementToReconciled() {
  if (isCurrentStatementLocked.value) return

  const statementNumber = settings.statementNumber.trim()
  if (!statementNumber) return

  const rows = visibleTransactions.value.filter((row) => row.isReconciled && !row.statementReference)

  for (const row of rows) {
    isSavingId.value = row.id

    try {
      const patch = { isReconciled: true, statementReference: statementNumber }

      if (row.type === 'achat') {
        await store.editPurchase(row.sourceId, { ...row.raw, ...patch })
      } else {
        await store.editIncome(row.sourceId, { ...row.raw, ...patch })
      }
    } catch {
      // On continue avec les lignes suivantes même en cas d'échec ponctuel.
    } finally {
      isSavingId.value = null
    }
  }
}

function requestDelete(row) {
  deleteError.value = ''
  deleteTarget.value = row
}

async function confirmDelete() {
  if (!deleteTarget.value) return

  isDeleting.value = true
  deleteError.value = ''

  try {
    if (deleteTarget.value.type === 'achat') {
      await store.removePurchase(deleteTarget.value.sourceId)
    } else {
      await store.removeIncome(deleteTarget.value.sourceId)
    }

    deleteTarget.value = null
  } catch (err) {
    deleteError.value = err instanceof Error ? err.message : 'Impossible de supprimer cette opération.'
  } finally {
    isDeleting.value = false
  }
}
</script>

<template>
  <main class="dashboard-content reconciliation-view">
    <PageHero
      eyebrow="Vérification"
      title="Pointage bancaire"
      description="Saisis les soldes de ton relevé et pointe tes opérations pour vérifier qu'elles correspondent."
    >
    </PageHero>

    <section class="panel reconciliation-card">
      <div class="panel-header statement-panel-header">
        <div>
          <p class="eyebrow">Relevé</p>
          <h2>Paramètres du relevé</h2>
        </div>

        <div class="statement-header-actions">
          <span v-if="currentSavedStatement" class="statement-saved-badge">✓ Relevé enregistré</span>
          <span v-if="isCurrentStatementLocked" class="statement-locked-badge">🔒 Verrouillé</span>
          <button class="ghost-btn" type="button" @click="startNewStatement">
            + Nouveau relevé
          </button>
          <button
            class="primary-btn"
            type="button"
            :disabled="!canSaveStatement || isSavingStatement"
            :title="isCurrentStatementLocked ? 'Déverrouille le relevé avant de le modifier' : ''"
            @click="saveCurrentStatement"
          >
            {{ isSavingStatement ? 'Enregistrement...' : 'Enregistrer le relevé' }}
          </button>
          <button class="ghost-btn" type="button" @click="toggleSection('settings')">
            {{ collapsedSections.settings ? 'Afficher' : 'Masquer' }}
          </button>
        </div>
      </div>

      <template v-if="!collapsedSections.settings">
        <p v-if="statementError" class="form-error statement-error">{{ statementError }}</p>

        <div class="settings-grid">
          <label class="form-field">
            <span>Numéro de relevé</span>
            <input
              v-model.trim="settings.statementNumber"
              type="text"
              placeholder="Ex: 2026-01"
              @change="persistSettings"
            />
          </label>

          <label class="form-field">
            <span>Date du relevé</span>
            <input
              v-model="settings.statementDate"
              type="date"
              :disabled="isCurrentStatementLocked"
              @change="persistSettings"
            />
          </label>

          <label class="form-field">
            <span>Solde de départ</span>
            <input
              v-model.number="settings.startBalance"
              type="number"
              step="0.01"
              placeholder="0.00"
              :disabled="isCurrentStatementLocked"
              @change="persistSettings"
            />
          </label>

          <label class="form-field">
            <span>Solde de fin</span>
            <input
              v-model.number="settings.endBalance"
              type="number"
              step="0.01"
              placeholder="0.00"
              :disabled="isCurrentStatementLocked"
              @change="persistSettings"
            />
          </label>

          <label class="form-field">
            <span>Période — du</span>
            <input
              v-model="settings.periodStart"
              type="date"
              :disabled="isCurrentStatementLocked"
              @change="persistSettings"
            />
          </label>

          <label class="form-field">
            <span>Période — au</span>
            <input
              v-model="settings.periodEnd"
              type="date"
              :disabled="isCurrentStatementLocked"
              @change="persistSettings"
            />
          </label>
        </div>
      </template>
    </section>

    <section class="panel reconciliation-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Historique</p>
          <h2>Relevés enregistrés</h2>
        </div>

        <button class="ghost-btn" type="button" @click="toggleSection('statements')">
          {{ collapsedSections.statements ? 'Afficher' : 'Masquer' }}
        </button>
      </div>

      <template v-if="!collapsedSections.statements">
        <p v-if="isLoadingStatements" class="statement-list-empty">Chargement des relevés...</p>

        <p v-else-if="!savedStatements.length" class="statement-list-empty">
          Aucun relevé enregistré pour l'instant — clique sur « Enregistrer le relevé » ci-dessus une
          fois tes soldes saisis.
        </p>

        <div v-else class="statement-list">
          <article
            v-for="item in savedStatements"
            :key="item.id"
            class="statement-row"
            :class="{
              'is-current': item.reference === settings.statementNumber.trim(),
              'is-drag-over': draggingOverId === item.id,
            }"
            role="button"
            tabindex="0"
            :title="`Afficher le relevé ${item.reference}`"
            @click="loadStatementIntoSettings(item)"
            @keydown.enter="loadStatementIntoSettings(item)"
            @dragover.prevent="draggingOverId = item.id"
            @dragleave.prevent="draggingOverId = null"
            @drop.prevent="handleStatementRowDrop(item, $event)"
          >
            <div class="statement-row-main">
              <strong>
                {{ item.reference }}
                <span v-if="item.is_locked" title="Relevé verrouillé">🔒</span>
              </strong>
              <span class="statement-row-meta">
                {{ item.statement_date ? formatDate(item.statement_date) : 'Sans date' }}
                · Début {{ formatCurrency(item.start_balance) }} · Fin {{ formatCurrency(item.end_balance) }}
              </span>
              <div v-if="pdfsForStatement(item).length" class="statement-pdf-list">
                <button
                  v-for="pdf in pdfsForStatement(item)"
                  :key="pdf.id"
                  class="statement-pdf-badge"
                  type="button"
                  :title="`Ouvrir ${pdf.original_filename}`"
                  @click.stop="openStatementPdf(item, pdf)"
                >
                  📄 {{ truncatePdfName(pdf.original_filename) }}
                  <span
                    class="statement-pdf-remove"
                    title="Retirer ce PDF"
                    @click.stop="removeStatementPdf(item, pdf)"
                  >
                    ✕
                  </span>
                </button>
              </div>
            </div>

            <div class="statement-row-actions" @click.stop>
              <input
                :ref="(el) => setPdfFileInputRef(el, item.id)"
                type="file"
                accept=".pdf,application/pdf"
                class="statement-pdf-input"
                @change="handlePdfFileInputChange(item, $event)"
              />
              <button
                class="ghost-btn ghost-btn-icon"
                type="button"
                title="Ajouter un PDF au relevé (ou glisse-le sur la ligne)"
                :disabled="uploadingPdfId === item.id"
                @click="triggerPdfFilePicker(item)"
              >
                {{ uploadingPdfId === item.id ? '…' : '📎' }}
              </button>
              <button class="ghost-btn" type="button" @click="toggleLock(item)">
                {{ item.is_locked ? 'Déverrouiller' : 'Verrouiller' }}
              </button>
              <button
                class="ghost-btn ghost-btn-danger"
                type="button"
                :disabled="item.is_locked"
                :title="item.is_locked ? 'Déverrouille le relevé avant de le supprimer' : ''"
                @click="requestDeleteStatement(item)"
              >
                Supprimer
              </button>
            </div>
          </article>
        </div>
      </template>
    </section>

    <section class="panel reconciliation-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Vérification</p>
          <h2>Résumé du pointage</h2>
        </div>

        <button class="ghost-btn" type="button" @click="toggleSection('summary')">
          {{ collapsedSections.summary ? 'Afficher' : 'Masquer' }}
        </button>
      </div>

      <template v-if="!collapsedSections.summary">
        <div class="summary-grid">
          <div class="summary-row">
            <span>Solde de départ</span>
            <strong>{{ formatCurrency(settings.startBalance) }}</strong>
          </div>
          <div class="summary-row">
            <span>Opérations pointées ({{ reconciledCount }})</span>
            <strong :class="reconciledTotal >= 0 ? 'amount-positive' : 'amount-negative'">
              {{ reconciledTotal >= 0 ? '+' : '' }}{{ formatCurrency(reconciledTotal) }}
            </strong>
          </div>
          <div class="summary-row summary-row-total">
            <span>Solde calculé</span>
            <strong>{{ formatCurrency(computedBalance) }}</strong>
          </div>
          <div class="summary-row">
            <span>Solde de fin (relevé)</span>
            <strong>{{ formatCurrency(settings.endBalance) }}</strong>
          </div>
          <div class="summary-row summary-row-gap" :class="{ 'is-balanced': isBalanced, 'is-off': !isBalanced }">
            <span>Écart</span>
            <strong>{{ formatCurrency(balanceGap) }}</strong>
          </div>
        </div>

        <p class="balance-status" :class="{ 'is-balanced': isBalanced, 'is-off': !isBalanced }">
          {{ isBalanced ? '✓ Tout est pointé, la balance est à 0.' : 'Il reste un écart — vérifie les opérations non pointées ci-dessous.' }}
        </p>
      </template>
    </section>

    <section class="panel reconciliation-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Opérations</p>
          <h2>Opérations de la période</h2>
        </div>

        <button class="ghost-btn" type="button" @click="toggleSection('operations')">
          {{ collapsedSections.operations ? 'Afficher' : 'Masquer' }}
        </button>
      </div>

      <template v-if="!collapsedSections.operations">
      <div class="reconciliation-toolbar">
        <div class="filter-tabs">
          <button
            class="filter-tab"
            :class="{ 'is-active': visibilityFilter === 'all' }"
            type="button"
            @click="visibilityFilter = 'all'"
          >
            Toutes ({{ periodTransactions.length }})
          </button>
          <button
            class="filter-tab"
            :class="{ 'is-active': visibilityFilter === 'unreconciled' }"
            type="button"
            @click="visibilityFilter = 'unreconciled'"
          >
            Non pointées ({{ periodTransactions.length - reconciledCount }})
          </button>
          <button
            class="filter-tab"
            :class="{ 'is-active': visibilityFilter === 'reconciled' }"
            type="button"
            @click="visibilityFilter = 'reconciled'"
          >
            Pointées ({{ reconciledCount }})
          </button>
        </div>

        <div class="search-group">
          <select v-model="searchColumn" class="search-column-select">
            <option v-for="option in SEARCH_COLUMNS" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>

          <input
            v-model.trim="searchQuery"
            type="search"
            class="search-input"
            placeholder="Rechercher…"
          />
        </div>

        <div class="bulk-actions">
          <button
            class="ghost-btn"
            type="button"
            :disabled="isCurrentStatementLocked"
            :title="isCurrentStatementLocked ? 'Déverrouille le relevé pour pouvoir pointer' : ''"
            @click="setAllVisible(true)"
          >
            Tout pointer
          </button>
          <button
            class="ghost-btn"
            type="button"
            :disabled="isCurrentStatementLocked"
            :title="isCurrentStatementLocked ? 'Déverrouille le relevé pour pouvoir dépointer' : ''"
            @click="setAllVisible(false)"
          >
            Tout dépointer
          </button>
          <button
            v-if="missingStatementCount > 0"
            class="ghost-btn"
            type="button"
            :disabled="!settings.statementNumber.trim() || isCurrentStatementLocked"
            :title="
              isCurrentStatementLocked
                ? 'Déverrouille le relevé pour pouvoir modifier ses opérations'
                : settings.statementNumber.trim()
                ? `Renseigne « ${settings.statementNumber} » sur les ${missingStatementCount} opération(s) pointée(s) sans relevé`
                : 'Saisis un numéro de relevé ci-dessus pour pouvoir l’appliquer'
            "
            @click="applyStatementToReconciled"
          >
            Renseigner le relevé ({{ missingStatementCount }})
          </button>
          <ColumnVisibilityMenu
            :columns="TABLE_COLUMNS"
            :is-visible="isVisible"
            :is-frozen="isFrozen"
            @toggle-visible="toggleVisible"
            @toggle-frozen="toggleFrozen"
          />
        </div>
      </div>

      <div v-if="!visibleTransactions.length" class="empty-state-inline">
        {{ searchQuery ? 'Aucune opération ne correspond à ta recherche.' : 'Aucune opération sur cette période.' }}
      </div>

      <div v-else class="reconciliation-table-wrap">
        <table class="reconciliation-table">
          <thead>
            <tr>
              <th class="checkbox-col">Pointé</th>
              <th
                v-if="isVisible('dateCompta')"
                class="sortable"
                :class="{ 'is-frozen': isFrozen('dateCompta') }"
                :style="columnStyle('dateCompta')"
                @click="setSort('dateCompta')"
              >
                Date compta <span class="sort-indicator">{{ sortIndicator('dateCompta') }}</span>
              </th>
              <th
                v-if="isVisible('type')"
                class="sortable"
                :class="{ 'is-frozen': isFrozen('type') }"
                :style="columnStyle('type')"
                @click="setSort('type')"
              >
                Type <span class="sort-indicator">{{ sortIndicator('type') }}</span>
              </th>
              <th
                v-if="isVisible('label')"
                class="sortable"
                :class="{ 'is-frozen': isFrozen('label') }"
                :style="columnStyle('label')"
                @click="setSort('label')"
              >
                Libellé <span class="sort-indicator">{{ sortIndicator('label') }}</span>
              </th>
              <th
                v-if="isVisible('meta')"
                class="sortable"
                :class="{ 'is-frozen': isFrozen('meta') }"
                :style="columnStyle('meta')"
                @click="setSort('meta')"
              >
                Catégorie / Source <span class="sort-indicator">{{ sortIndicator('meta') }}</span>
              </th>
              <th
                v-if="isVisible('reference')"
                class="sortable"
                :class="{ 'is-frozen': isFrozen('reference') }"
                :style="columnStyle('reference')"
                @click="setSort('reference')"
              >
                Référence <span class="sort-indicator">{{ sortIndicator('reference') }}</span>
              </th>
              <th
                v-if="isVisible('dateOperation')"
                class="sortable"
                :class="{ 'is-frozen': isFrozen('dateOperation') }"
                :style="columnStyle('dateOperation')"
                @click="setSort('dateOperation')"
              >
                Date opération <span class="sort-indicator">{{ sortIndicator('dateOperation') }}</span>
              </th>
              <th
                v-if="isVisible('dateValeur')"
                class="sortable"
                :class="{ 'is-frozen': isFrozen('dateValeur') }"
                :style="columnStyle('dateValeur')"
                @click="setSort('dateValeur')"
              >
                Date valeur <span class="sort-indicator">{{ sortIndicator('dateValeur') }}</span>
              </th>
              <th
                v-if="isVisible('amount')"
                class="sortable"
                :class="{ 'is-frozen': isFrozen('amount') }"
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
              v-for="row in visibleTransactions"
              :key="row.id"
              :class="{ 'is-reconciled': row.isReconciled }"
            >
              <td class="checkbox-col">
                <input
                  type="checkbox"
                  :checked="row.isReconciled"
                  :disabled="isSavingId === row.id || isCurrentStatementLocked"
                  :title="isCurrentStatementLocked ? 'Déverrouille le relevé pour pouvoir pointer' : ''"
                  @change="toggleReconciled(row)"
                />
              </td>
              <td v-if="isVisible('dateCompta')" :class="{ 'is-frozen': isFrozen('dateCompta') }" :style="columnStyle('dateCompta')">
                {{ formatDate(row.dateCompta) }}
              </td>
              <td v-if="isVisible('type')" :class="{ 'is-frozen': isFrozen('type') }" :style="columnStyle('type')">
                <span class="import-type-badge" :class="row.type === 'achat' ? 'is-achat' : 'is-revenu'">
                  {{ row.type === 'achat' ? 'Achat' : 'Revenu' }}
                </span>
              </td>
              <td
                v-if="isVisible('label')"
                class="label-cell"
                :class="{ 'is-frozen': isFrozen('label') }"
                :style="columnStyle('label')"
                :title="row.label"
              >
                {{ row.label }}
              </td>
              <td v-if="isVisible('meta')" class="meta-cell" :class="{ 'is-frozen': isFrozen('meta') }" :style="columnStyle('meta')">
                {{ row.meta }}
              </td>
              <td
                v-if="isVisible('reference')"
                class="reference-cell"
                :class="{ 'is-frozen': isFrozen('reference') }"
                :style="columnStyle('reference')"
                :title="row.reference"
              >
                {{ row.reference }}
              </td>
              <td v-if="isVisible('dateOperation')" :class="{ 'is-frozen': isFrozen('dateOperation') }" :style="columnStyle('dateOperation')">
                {{ formatDate(row.dateOperation) }}
              </td>
              <td v-if="isVisible('dateValeur')" :class="{ 'is-frozen': isFrozen('dateValeur') }" :style="columnStyle('dateValeur')">
                {{ formatDate(row.dateValeur) }}
              </td>
              <td
                v-if="isVisible('amount')"
                :class="[row.amount < 0 ? 'amount-negative' : 'amount-positive', { 'is-frozen': isFrozen('amount') }]"
                :style="columnStyle('amount')"
              >
                {{ row.amount >= 0 ? '+' : '' }}{{ formatCurrency(row.amount) }}
              </td>
              <td class="actions-col">
                <button
                  class="icon-btn icon-btn-danger"
                  type="button"
                  title="Supprimer"
                  aria-label="Supprimer cette opération"
                  @click="requestDelete(row)"
                >
                  🗑️
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      </template>
    </section>

    <ConfirmModal
      :model-value="Boolean(deleteTarget)"
      title="Supprimer cette opération ?"
      :message="`${deleteTarget?.label} — ${formatCurrency(Math.abs(deleteTarget?.amount ?? 0))} ne sera plus suivi dans l'application.`"
      :note="deleteError || 'Cette action est irréversible.'"
      confirm-label="Supprimer"
      :is-processing="isDeleting"
      @update:model-value="deleteTarget = null"
      @confirm="confirmDelete"
    />

    <ConfirmModal
      :model-value="Boolean(deleteStatementTarget)"
      title="Supprimer ce relevé ?"
      :message="`Le relevé « ${deleteStatementTarget?.reference} » ne sera plus dans l'historique. Les opérations déjà pointées ne sont pas affectées.`"
      note="Cette action est irréversible."
      confirm-label="Supprimer"
      :is-processing="isDeletingStatement"
      @update:model-value="deleteStatementTarget = null"
      @confirm="confirmDeleteStatement"
    />
  </main>
</template>

<style scoped>
.reconciliation-view {
  display: grid;
  gap: 0.9rem;
}

.reconciliation-card {
  padding: 1.1rem;
}

.panel-header {
  margin-bottom: 0.9rem;
}

.eyebrow {
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.74rem;
}

.panel-header h2 {
  margin: 0.25rem 0 0;
  font-size: 1.2rem;
  color: var(--text, #eef1f3);
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.9rem;
}

.form-field {
  display: grid;
  gap: 0.45rem;
}

.form-field span {
  color: var(--text-soft, #b3bbc4);
  font-size: 0.88rem;
  font-weight: 600;
}

.form-field input {
  width: 100%;
  height: 44px;
  padding: 0 0.9rem;
  border-radius: 12px;
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--text, #eef1f3);
  outline: none;
}

.form-field input:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.summary-grid {
  display: grid;
  gap: 0.55rem;
  max-width: 460px;
}

.summary-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.5rem 0;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.92rem;
  border-bottom: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
}

.summary-row strong {
  color: var(--text, #eef1f3);
}

.summary-row-total {
  border-bottom: none;
  padding-top: 0.7rem;
  margin-top: 0.2rem;
  border-top: 1px dashed var(--line-soft, rgba(255, 255, 255, 0.08));
}

.summary-row-gap {
  border-bottom: none;
  font-weight: 700;
}

.summary-row-gap span {
  font-weight: 700;
  color: var(--text, #eef1f3);
}

.summary-row-gap.is-balanced strong {
  color: var(--positive-text, #bfe0c9);
}

.summary-row-gap.is-off strong {
  color: var(--negative-text, #f5b4b4);
}

.balance-status {
  margin: 0.9rem 0 0;
  padding: 0.75rem 0.95rem;
  border-radius: 12px;
  font-size: 0.88rem;
  font-weight: 600;
}

.balance-status.is-balanced {
  background: rgba(34, 197, 94, 0.12);
  color: var(--positive-text, #bfe0c9);
  border: 1px solid rgba(34, 197, 94, 0.22);
}

.balance-status.is-off {
  background: rgba(220, 38, 38, 0.1);
  color: var(--negative-text, #f5b4b4);
  border: 1px solid rgba(220, 38, 38, 0.2);
}

.reconciliation-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 0.7rem;
  margin-bottom: 0.9rem;
}

.filter-tabs {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.filter-tab {
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  border-radius: 999px;
  padding: 0.45rem 0.9rem;
  background: rgba(var(--tint-rgb), 0.03);
  color: var(--text-soft, #b3bbc4);
  font-size: 0.84rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 140ms ease, color 140ms ease, border-color 140ms ease;
}

.filter-tab:hover {
  color: var(--text, #eef1f3);
}

.filter-tab.is-active {
  background: rgba(219, 230, 223, 0.14);
  border-color: rgba(219, 230, 223, 0.3);
  color: var(--text, #eef1f3);
}

.search-group {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex: 1 1 360px;
}

.search-column-select {
  flex: 0 0 auto;
  height: 38px;
  padding: 0 0.7rem;
  border-radius: 999px;
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--text-soft, #b3bbc4);
  font-size: 0.82rem;
  outline: none;
}

.search-input {
  flex: 1 1 200px;
  min-width: 180px;
  max-width: 320px;
  height: 38px;
  padding: 0 0.9rem;
  border-radius: 999px;
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--text, #eef1f3);
  font-size: 0.84rem;
  outline: none;
}

.search-input::placeholder {
  color: var(--text-dim, #8a939d);
}

.search-input:focus {
  border-color: rgba(114, 137, 152, 0.55);
  box-shadow: 0 0 0 3px rgba(114, 137, 152, 0.12);
}

.bulk-actions {
  display: flex;
  gap: 0.5rem;
}

.ghost-btn {
  border: none;
  border-radius: 10px;
  padding: 0.55rem 0.85rem;
  background: rgba(var(--tint-rgb), 0.06);
  color: var(--text, #eef1f3);
  font-size: 0.84rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 140ms ease, background 140ms ease;
}

.ghost-btn:hover {
  transform: translateY(-1px);
}

.empty-state-inline {
  padding: 1.4rem;
  text-align: center;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.92rem;
}

.reconciliation-table-wrap {
  max-height: 520px;
  overflow: auto;
  border-radius: 14px;
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.06));
}

.reconciliation-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.86rem;
}

.reconciliation-table th {
  position: sticky;
  top: 0;
  background: var(--bg-elevated, rgba(20, 23, 27, 0.98));
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-size: 0.68rem;
  text-align: left;
  padding: 0.6rem 0.6rem;
  white-space: nowrap;
}

.reconciliation-table th.sortable {
  cursor: pointer;
  user-select: none;
  transition: color 140ms ease;
}

.reconciliation-table th.sortable:hover {
  color: var(--text, #eef1f3);
}

.sort-indicator {
  display: inline-block;
  width: 0.7em;
  font-size: 0.7rem;
  color: var(--text-soft, #b3bbc4);
}

.reconciliation-table td {
  padding: 0.55rem 0.6rem;
  border-top: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
  color: var(--text, #eef1f3);
  vertical-align: middle;
}

.reconciliation-table tr.is-reconciled {
  background: rgba(34, 197, 94, 0.16);
}

.checkbox-col {
  position: sticky;
  left: 0;
  z-index: 3;
  background: var(--bg-elevated, rgba(20, 23, 27, 0.98));
}

td.checkbox-col {
  z-index: 2;
  transition: background-color 160ms ease, box-shadow 160ms ease;
}

td.checkbox-col input[type='checkbox'] {
  width: 17px;
  height: 17px;
  accent-color: var(--positive-text, #22c55e);
  cursor: pointer;
}

th.is-frozen {
  z-index: 3;
  background: var(--bg-elevated, rgba(20, 23, 27, 0.98));
  box-shadow: 2px 0 4px rgba(0, 0, 0, 0.12);
}

td.is-frozen {
  background: var(--bg-soft, #262a2f);
  box-shadow: 2px 0 4px rgba(0, 0, 0, 0.12);
  transition: background-color 160ms ease;
}

.reconciliation-table tr.is-reconciled td.checkbox-col {
  background: color-mix(in srgb, var(--bg-elevated, #2c3137) 78%, #22c55e 22%);
  box-shadow: inset 3px 0 0 #22c55e;
}

.reconciliation-table tr.is-reconciled td.is-frozen {
  background: color-mix(in srgb, var(--bg-soft, #262a2f) 78%, #22c55e 22%);
}

.label-cell {
  max-width: 260px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta-cell {
  color: var(--text-soft, #b3bbc4);
  font-size: 0.82rem;
}

.reference-cell {
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-dim, #8a939d);
  font-size: 0.8rem;
}

.import-type-badge {
  display: inline-block;
  padding: 0.22rem 0.6rem;
  border-radius: 999px;
  font-size: 0.72rem;
  font-weight: 600;
  white-space: nowrap;
}

.import-type-badge.is-achat {
  background: rgba(220, 38, 38, 0.12);
  color: var(--negative-text, #e1b4b4);
}

.import-type-badge.is-revenu {
  background: rgba(34, 197, 94, 0.12);
  color: var(--positive-text, #bfe0c9);
}

.amount-negative {
  color: var(--negative-text, #e1b4b4);
  font-weight: 600;
  white-space: nowrap;
}

.amount-positive {
  color: var(--positive-text, #bfe0c9);
  font-weight: 600;
  white-space: nowrap;
}

.actions-col {
  text-align: center;
  white-space: nowrap;
}

.icon-btn {
  width: 26px;
  height: 26px;
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  border-radius: 8px;
  background: rgba(var(--tint-rgb), 0.035);
  color: var(--text);
  cursor: pointer;
  font-size: 0.78rem;
  line-height: 1;
  transition: background 140ms ease, border-color 140ms ease;
}

.icon-btn-danger {
  color: var(--negative-text);
  background: rgba(220, 38, 38, 0.1);
  border-color: rgba(220, 38, 38, 0.14);
}

.icon-btn-danger:hover {
  background: rgba(220, 38, 38, 0.16);
  border-color: rgba(220, 38, 38, 0.22);
}

.statement-panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
}

.statement-header-actions {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  flex-wrap: wrap;
}

.statement-saved-badge {
  padding: 0.3rem 0.65rem;
  border-radius: 999px;
  background: rgba(34, 197, 94, 0.12);
  color: var(--positive-text, #bfe0c9);
  border: 1px solid rgba(34, 197, 94, 0.2);
  font-size: 0.78rem;
  font-weight: 600;
  white-space: nowrap;
}

.statement-locked-badge {
  padding: 0.3rem 0.65rem;
  border-radius: 999px;
  background: rgba(239, 68, 68, 0.12);
  color: var(--negative-text, #e1b4b4);
  border: 1px solid rgba(239, 68, 68, 0.2);
  font-size: 0.78rem;
  font-weight: 600;
  white-space: nowrap;
}

.statement-error {
  margin: 0 0 0.9rem;
}

.statement-list-empty {
  margin: 0;
  padding: 1.2rem;
  text-align: center;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.88rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.025);
  border: 1px solid rgba(var(--tint-rgb), 0.05);
}

.statement-list {
  display: grid;
  gap: 0.55rem;
}

.statement-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.9rem;
  padding: 0.7rem 0.85rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.025);
  border: 1px solid rgba(var(--tint-rgb), 0.05);
  cursor: pointer;
  transition: background 140ms ease, border-color 140ms ease;
}

.statement-row:hover {
  background: rgba(var(--tint-rgb), 0.05);
  border-color: rgba(var(--tint-rgb), 0.1);
}

.statement-row:focus-visible {
  outline: 2px solid rgba(114, 137, 152, 0.6);
  outline-offset: 2px;
}

.statement-row.is-current {
  border-color: rgba(34, 197, 94, 0.28);
  background: rgba(34, 197, 94, 0.06);
}

.statement-row.is-drag-over {
  border-color: rgba(114, 137, 152, 0.6);
  background: rgba(114, 137, 152, 0.12);
}

.statement-row-main {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  min-width: 0;
}

.statement-row-main strong {
  color: var(--text, #eef1f3);
  font-size: 0.92rem;
}

.statement-row-meta {
  color: var(--text-soft, #b3bbc4);
  font-size: 0.8rem;
}

.statement-pdf-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  margin-top: 0.15rem;
}

.statement-pdf-badge {
  justify-self: start;
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.15rem 0.55rem;
  border: 1px solid rgba(114, 137, 152, 0.3);
  border-radius: 999px;
  background: rgba(114, 137, 152, 0.12);
  color: var(--text-soft, #b3bbc4);
  font-size: 0.74rem;
  font-weight: 600;
  cursor: pointer;
  width: fit-content;
}

.statement-pdf-badge:hover {
  background: rgba(114, 137, 152, 0.2);
  color: var(--text, #eef1f3);
}

.statement-pdf-remove {
  color: var(--text-dim, #8a939d);
  transition: color 140ms ease;
}

.statement-pdf-remove:hover {
  color: var(--negative-text);
}

.statement-pdf-input {
  display: none;
}

.ghost-btn-icon {
  min-width: 0;
  padding: 0 0.7rem;
  font-size: 0.95rem;
}

.statement-row-actions {
  display: flex;
  gap: 0.5rem;
  flex-shrink: 0;
}

.ghost-btn-danger {
  color: var(--negative-text);
}

@media (max-width: 900px) {
  .settings-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .settings-grid {
    grid-template-columns: 1fr;
  }

  .reconciliation-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .statement-row {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
