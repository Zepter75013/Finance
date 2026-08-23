<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import PageHero from '../Common/PageHero.vue'
import QuickCreateModal from '../Common/QuickCreateModal.vue'
import { usePurchasesStore } from '../../stores/purchases'
import { parseCsv, normalizeAmount, normalizeDate } from '../../utils/csv'
import { formatCurrency, formatDate } from '../../utils/format'

const emit = defineEmits(['imported'])

const store = usePurchasesStore()
const { categoriesList, subCategoriesList, purchases, incomes, activeAccountId, accountsList } =
  storeToRefs(store)

const otherAccountsList = computed(() =>
  accountsList.value.filter((account) => Number(account.id) !== Number(activeAccountId.value))
)

const isCategoryModalOpen = ref(false)
const categoryModalRow = ref(null)
const isCreatingCategoryInline = ref(false)
const categoryModalError = ref('')

const isSubCategoryModalOpen = ref(false)
const subCategoryModalRow = ref(null)
const isCreatingSubCategoryInline = ref(false)
const subCategoryModalError = ref('')

// Certaines banques réutilisent le même numéro de référence d'un relevé à
// l'autre (ou d'une année sur l'autre) — se fier à la seule référence pour
// détecter les doublons rejetait alors à tort de vraies nouvelles opérations.
// On exige donc en plus la même date ET le même montant.
function duplicateKey(reference, date, amount) {
  return `${reference}|${date}|${Number(amount).toFixed(2)}`
}

const existingPurchaseKeys = computed(() => {
  const keys = new Set()

  for (const purchase of purchases.value) {
    const reference = purchase.reference?.trim()
    if (!reference) continue
    keys.add(duplicateKey(reference, purchase.date, purchase.amount))
  }

  return keys
})

const existingIncomeKeys = computed(() => {
  const keys = new Set()

  for (const income of incomes.value) {
    const reference = income.reference?.trim()
    if (!reference) continue
    keys.add(duplicateKey(reference, income.income_date, income.amount))
  }

  return keys
})

const PAYMENT_METHODS = ['Carte bancaire', 'Visa', 'Mastercard', 'Espèces', 'Virement', 'Prélèvement']

const DATE_KEYWORDS = ['date']
const LABEL_KEYWORDS = ['libell', 'description', 'intitul', 'communication', 'nature', 'operation', 'opération', 'tiers']
const AMOUNT_KEYWORDS = ['montant', 'amount']
const DEBIT_KEYWORDS = ['debit', 'débit']
const CREDIT_KEYWORDS = ['credit', 'crédit']
const CATEGORY_KEYWORDS = ['categorie', 'catégorie', 'category']
const REFERENCE_KEYWORDS = ['reference', 'référence']
const OPERATION_LABEL_KEYWORDS = ['libelle operation', 'libellé opération', 'libelle opération', 'libellé operation']
const ADDITIONAL_INFO_KEYWORDS = ['information', 'complementaire', 'complémentaire']
const SUB_CATEGORY_KEYWORDS = ['sous categorie', 'sous-categorie', 'sous catégorie', 'souscategorie']
const TYPE_OPERATION_KEYWORDS = ["type d'operation", "type d'opération", 'type operation', 'type opération']
const OPERATION_DATE_KEYWORDS = ["date d'operation", "date d'opération", 'date operation', 'date opération']
const VALUE_DATE_KEYWORDS = ['date de valeur', 'date valeur']
const RECONCILED_KEYWORDS = ['pointage']

const step = ref('upload')
const fileName = ref('')
const fileError = ref('')
const isDragging = ref(false)

const csvHeaders = ref([])
const csvRows = ref([])
const skippedRowCount = ref(0)

const mapping = reactive({
  dateColumn: '',
  labelColumn: '',
  amountMode: 'single',
  amountColumn: '',
  debitColumn: '',
  creditColumn: '',
  categoryColumn: '',
  paymentMethod: 'Carte bancaire',
  referenceColumn: '',
  operationLabelColumn: '',
  additionalInfoColumn: '',
  subCategoryColumn: '',
  typeOperationColumn: '',
  operationDateColumn: '',
  valueDateColumn: '',
  reconciledColumn: '',
})

const transactions = ref([])
const duplicateTransactions = ref([])
const isImporting = ref(false)
const importSummary = ref(null)
const isCreatingCategories = ref(false)

// Sauvegarde locale de l'état d'import en cours (fichier, mapping, catégories
// déjà assignées) pour pouvoir reprendre plus tard si on n'a pas le temps de
// tout finir d'un coup — restreint à ce navigateur.
const IMPORT_DRAFT_KEY = 'importDraftV1'
const savedDraft = ref(null)
let saveDraftTimer = null

function loadSavedDraft() {
  try {
    const raw = localStorage.getItem(IMPORT_DRAFT_KEY)
    if (!raw) return null
    return JSON.parse(raw)
  } catch {
    return null
  }
}

function clearDraft() {
  savedDraft.value = null
  try {
    localStorage.removeItem(IMPORT_DRAFT_KEY)
  } catch {
    // stockage indisponible : rien à nettoyer
  }
}

function saveDraftNow() {
  if (step.value === 'upload') return

  const payload = {
    savedAt: new Date().toISOString(),
    fileName: fileName.value,
    step: step.value,
    csvHeaders: csvHeaders.value,
    csvRows: csvRows.value,
    mapping: { ...mapping },
    transactions: transactions.value,
    duplicateTransactions: duplicateTransactions.value,
    skippedRowCount: skippedRowCount.value,
  }

  try {
    localStorage.setItem(IMPORT_DRAFT_KEY, JSON.stringify(payload))
  } catch {
    // quota localStorage dépassé ou indisponible : on continue sans sauvegarder
  }
}

function scheduleDraftSave() {
  clearTimeout(saveDraftTimer)
  saveDraftTimer = setTimeout(saveDraftNow, 400)
}

function resumeDraft() {
  const draft = savedDraft.value
  if (!draft) return

  fileName.value = draft.fileName
  csvHeaders.value = draft.csvHeaders
  csvRows.value = draft.csvRows
  Object.assign(mapping, draft.mapping)
  transactions.value = draft.transactions
  duplicateTransactions.value = draft.duplicateTransactions
  skippedRowCount.value = draft.skippedRowCount
  step.value = draft.step

  savedDraft.value = null
}

function discardDraft() {
  clearDraft()
}

function normalizeHeaderName(value) {
  return String(value ?? '')
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .trim()
}

function guessColumn(headers, keywords) {
  const normalizedHeaders = headers.map(normalizeHeaderName)

  for (const keyword of keywords) {
    const index = normalizedHeaders.findIndex((header) => header.includes(keyword))
    if (index !== -1) return headers[index]
  }

  return ''
}

function resetState() {
  step.value = 'upload'
  fileName.value = ''
  fileError.value = ''
  isDragging.value = false
  csvHeaders.value = []
  csvRows.value = []
  skippedRowCount.value = 0
  mapping.dateColumn = ''
  mapping.labelColumn = ''
  mapping.amountMode = 'single'
  mapping.amountColumn = ''
  mapping.debitColumn = ''
  mapping.creditColumn = ''
  mapping.categoryColumn = ''
  mapping.paymentMethod = 'Carte bancaire'
  mapping.referenceColumn = ''
  mapping.operationLabelColumn = ''
  mapping.additionalInfoColumn = ''
  mapping.subCategoryColumn = ''
  mapping.typeOperationColumn = ''
  mapping.operationDateColumn = ''
  mapping.valueDateColumn = ''
  mapping.reconciledColumn = ''
  transactions.value = []
  duplicateTransactions.value = []
  isImporting.value = false
  importSummary.value = null
  clearDraft()
}

savedDraft.value = loadSavedDraft()

watch([transactions, mapping, step], scheduleDraftSave, { deep: true })

async function decodeFileText(file) {
  const buffer = await file.arrayBuffer()

  // Les exports bancaires français sont très souvent en Latin-1/Windows-1252
  // plutôt qu'en UTF-8 (ex: le caractère « ° » dans « N° »).
  try {
    const decoded = new TextDecoder('windows-1252', { fatal: true }).decode(buffer)
    return decoded
  } catch {
    return new TextDecoder('utf-8').decode(buffer)
  }
}

async function handleFile(file) {
  fileError.value = ''

  if (!file) return

  try {
    const text = await decodeFileText(file)
    const { headers, rows } = parseCsv(text)

    if (!headers.length || !rows.length) {
      fileError.value = 'Le fichier semble vide ou illisible.'
      return
    }

    fileName.value = file.name
    csvHeaders.value = headers
    csvRows.value = rows

    mapping.dateColumn = guessColumn(headers, DATE_KEYWORDS)
    mapping.labelColumn = guessColumn(headers, LABEL_KEYWORDS)
    mapping.amountColumn = guessColumn(headers, AMOUNT_KEYWORDS)
    mapping.debitColumn = guessColumn(headers, DEBIT_KEYWORDS)
    mapping.creditColumn = guessColumn(headers, CREDIT_KEYWORDS)
    mapping.categoryColumn = guessColumn(headers, CATEGORY_KEYWORDS)
    mapping.amountMode = mapping.amountColumn ? 'single' : 'split'
    mapping.referenceColumn = guessColumn(headers, REFERENCE_KEYWORDS)
    mapping.operationLabelColumn = guessColumn(headers, OPERATION_LABEL_KEYWORDS)
    mapping.additionalInfoColumn = guessColumn(headers, ADDITIONAL_INFO_KEYWORDS)
    mapping.subCategoryColumn = guessColumn(headers, SUB_CATEGORY_KEYWORDS)
    mapping.typeOperationColumn = guessColumn(headers, TYPE_OPERATION_KEYWORDS)
    mapping.operationDateColumn = guessColumn(headers, OPERATION_DATE_KEYWORDS)
    mapping.valueDateColumn = guessColumn(headers, VALUE_DATE_KEYWORDS)
    mapping.reconciledColumn = guessColumn(headers, RECONCILED_KEYWORDS)

    step.value = 'mapping'
  } catch (err) {
    fileError.value = err instanceof Error ? err.message : 'Impossible de lire ce fichier.'
  }
}

function handleFileInput(event) {
  const file = event.target?.files?.[0]
  handleFile(file)
}

function handleDrop(event) {
  isDragging.value = false
  const file = event.dataTransfer?.files?.[0]
  handleFile(file)
}



function formatDraftDate(value) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''

  return new Intl.DateTimeFormat('fr-FR', {
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function truncateLabel(value, maxLength = 60) {
  const trimmed = String(value ?? '').trim()
  return trimmed.length > maxLength ? `${trimmed.slice(0, maxLength - 1)}…` : trimmed
}

function findMatchingCategoryId(rawName, type) {
  const normalized = normalizeHeaderName(rawName)
  if (!normalized) return null

  const match = categoriesList.value.find(
    (category) =>
      normalizeHeaderName(category.name) === normalized && (category.type || 'achat') === type
  )

  return match ? match.id : null
}

// Apprend des achats/revenus déjà importés et classifiés : pour un même
// libellé (commerçant/source), on retient la catégorie et sous-catégorie les
// plus fréquemment utilisées, pour les proposer par défaut sur les nouvelles
// lignes — l'utilisateur garde toujours la main pour corriger.
function buildHistoricalSuggestions(records, labelField) {
  const lookup = new Map()

  for (const record of records) {
    const normalized = normalizeHeaderName(record[labelField])
    if (!normalized) continue

    if (!lookup.has(normalized)) {
      lookup.set(normalized, { categoryCounts: new Map(), subCategoryCounts: new Map() })
    }

    const entry = lookup.get(normalized)

    if (record.category) {
      entry.categoryCounts.set(record.category, (entry.categoryCounts.get(record.category) || 0) + 1)
    }

    if (record.subCategory) {
      entry.subCategoryCounts.set(
        record.subCategory,
        (entry.subCategoryCounts.get(record.subCategory) || 0) + 1
      )
    }
  }

  return lookup
}

function pickMostFrequent(countsMap) {
  let bestValue = null
  let bestCount = 0

  for (const [value, count] of countsMap) {
    if (count > bestCount) {
      bestValue = value
      bestCount = count
    }
  }

  return bestValue
}

// Distance de Levenshtein classique (nombre minimal d'insertions/suppressions/
// substitutions pour passer de a à b), utilisée pour repérer un libellé
// "quasiment identique" à un libellé déjà importé (ex: un numéro de carte ou
// une ville qui varie légèrement d'un relevé à l'autre pour le même commerçant).
function levenshteinDistance(a, b) {
  const rows = a.length + 1
  const cols = b.length + 1
  let previousRow = Array.from({ length: cols }, (_, i) => i)

  for (let i = 1; i < rows; i += 1) {
    const currentRow = [i]

    for (let j = 1; j < cols; j += 1) {
      const cost = a[i - 1] === b[j - 1] ? 0 : 1
      currentRow.push(
        Math.min(
          previousRow[j] + 1, // suppression
          currentRow[j - 1] + 1, // insertion
          previousRow[j - 1] + cost // substitution
        )
      )
    }

    previousRow = currentRow
  }

  return previousRow[cols - 1]
}

function similarityRatio(a, b) {
  const maxLength = Math.max(a.length, b.length)
  if (maxLength === 0) return 1
  return 1 - levenshteinDistance(a, b) / maxLength
}

// Seuil de ressemblance demandé par l'utilisateur pour considérer un libellé
// comme "quasiment identique" à un libellé déjà importé et classifié.
const FUZZY_MATCH_THRESHOLD = 0.7
const FUZZY_MATCH_MIN_LENGTH = 4

const historicalPurchaseSuggestions = computed(() => buildHistoricalSuggestions(purchases.value, 'merchant'))
const historicalIncomeSuggestions = computed(() => buildHistoricalSuggestions(incomes.value, 'source'))

function findBestFuzzyMatch(normalized, lookup) {
  if (normalized.length < FUZZY_MATCH_MIN_LENGTH) return null

  let bestEntry = null
  let bestRatio = FUZZY_MATCH_THRESHOLD

  for (const [key, entry] of lookup) {
    if (key.length < FUZZY_MATCH_MIN_LENGTH) continue

    const ratio = similarityRatio(normalized, key)
    if (ratio >= bestRatio) {
      bestRatio = ratio
      bestEntry = entry
    }
  }

  return bestEntry
}

function suggestCategoryFromHistory(label, type) {
  const normalized = normalizeHeaderName(label)
  if (!normalized) return null

  const lookup = type === 'achat' ? historicalPurchaseSuggestions.value : historicalIncomeSuggestions.value
  const entry = lookup.get(normalized) || findBestFuzzyMatch(normalized, lookup)
  if (!entry) return null

  const categoryName = pickMostFrequent(entry.categoryCounts)
  if (!categoryName) return null

  return { categoryName, subCategoryName: pickMostFrequent(entry.subCategoryCounts) }
}

function categoriesForRow(row) {
  return categoriesList.value
    .filter((category) => (category.type || 'achat') === row.type)
    .sort((a, b) => a.name.localeCompare(b.name, 'fr', { sensitivity: 'base' }))
}

function findMatchingSubCategoryId(categoryId, rawName) {
  const normalized = normalizeHeaderName(rawName)
  if (!normalized || !categoryId) return null

  const match = subCategoriesList.value.find(
    (sub) => Number(sub.categoryId) === Number(categoryId) && normalizeHeaderName(sub.name) === normalized
  )

  return match ? match.id : null
}

function subCategoriesForRow(row) {
  if (!row.categoryId) return []

  return subCategoriesList.value
    .filter((sub) => Number(sub.categoryId) === Number(row.categoryId))
    .sort((a, b) => a.name.localeCompare(b.name, 'fr', { sensitivity: 'base' }))
}

function subCategoryNameForRow(row) {
  if (row.subCategoryId) {
    const match = subCategoriesList.value.find((sub) => sub.id === row.subCategoryId)
    if (match) return match.name
  }

  return row.bankSubCategoryName
}

function categoryNameForRow(row) {
  if (row.categoryId) {
    const match = categoriesList.value.find((category) => category.id === row.categoryId)
    if (match) return match.name
  }

  return row.bankCategoryName
}

let nextRowId = 1

function columnValue(row, columnName) {
  const index = csvHeaders.value.indexOf(columnName)
  return index !== -1 ? (row[index] ?? '').trim() : ''
}

function isTruthyFlag(value) {
  const normalized = String(value ?? '').trim().toLowerCase()
  return normalized !== '' && normalized !== '0' && normalized !== 'false' && normalized !== 'non'
}

function buildTransactions() {
  const results = []
  const duplicates = []
  let skipped = 0
  const seenPurchaseKeys = new Set()
  const seenIncomeKeys = new Set()

  for (const row of csvRows.value) {
    const rawDate = columnValue(row, mapping.dateColumn)
    const rawLabel = columnValue(row, mapping.labelColumn)
    const rawCategory = columnValue(row, mapping.categoryColumn)

    let amount = NaN

    if (mapping.amountMode === 'single') {
      amount = normalizeAmount(columnValue(row, mapping.amountColumn))
    } else {
      const debit = normalizeAmount(columnValue(row, mapping.debitColumn))
      const credit = normalizeAmount(columnValue(row, mapping.creditColumn))

      if (!Number.isNaN(credit) && credit !== 0) amount = Math.abs(credit)
      else if (!Number.isNaN(debit) && debit !== 0) amount = -Math.abs(debit)
    }

    const date = normalizeDate(rawDate)

    if (!date || Number.isNaN(amount) || amount === 0) {
      skipped += 1
      continue
    }

    const type = amount < 0 ? 'achat' : 'revenu'
    const label = truncateLabel(rawLabel) || 'Sans libellé'
    const bankCategoryName = rawCategory

    const reference = columnValue(row, mapping.referenceColumn)
    const operationLabel = columnValue(row, mapping.operationLabelColumn)
    const additionalInfo = columnValue(row, mapping.additionalInfoColumn)
    const bankSubCategoryName = columnValue(row, mapping.subCategoryColumn)
    const typeOperation = columnValue(row, mapping.typeOperationColumn)
    const rawOperationDate = columnValue(row, mapping.operationDateColumn)
    const rawValueDate = columnValue(row, mapping.valueDateColumn)
    const isReconciled = isTruthyFlag(columnValue(row, mapping.reconciledColumn))

    let isDuplicate = false

    if (reference) {
      const key = duplicateKey(reference, date, Math.abs(amount))
      const existingKeys = type === 'achat' ? existingPurchaseKeys.value : existingIncomeKeys.value
      const seenKeys = type === 'achat' ? seenPurchaseKeys : seenIncomeKeys

      isDuplicate = existingKeys.has(key) || seenKeys.has(key)
      seenKeys.add(key)
    }

    if (isDuplicate) {
      duplicates.push({ id: nextRowId++, date, label, amount: Math.abs(amount), type, reference })
      continue
    }

    let categoryId = findMatchingCategoryId(bankCategoryName, type)
    let subCategoryId = findMatchingSubCategoryId(categoryId, bankSubCategoryName)
    let isSuggested = false

    if (!categoryId) {
      const suggestion = suggestCategoryFromHistory(label, type)
      const suggestedCategoryId = suggestion ? findMatchingCategoryId(suggestion.categoryName, type) : null

      if (suggestedCategoryId) {
        categoryId = suggestedCategoryId
        subCategoryId = findMatchingSubCategoryId(categoryId, suggestion.subCategoryName)
        isSuggested = true
      }
    }

    results.push({
      id: nextRowId++,
      date,
      label,
      amount: Math.abs(amount),
      type,
      included: true,
      categoryId,
      isSuggested,
      bankCategoryName,
      subCategoryId,
      bankSubCategoryName,
      reference,
      operationLabel,
      additionalInfo,
      operationType: typeOperation,
      paymentMethod: type === 'achat' ? typeOperation || mapping.paymentMethod : '',
      operationDate: normalizeDate(rawOperationDate) || '',
      valueDate: normalizeDate(rawValueDate) || '',
      isReconciled,
      error: '',
      // Un virement interne (vers/depuis un autre compte géré dans l'app) ne
      // doit jamais devenir un achat/revenu — voir confirmImport(). Le signe
      // du montant (déjà capturé par `type`) continue de déterminer le sens.
      isTransfer: false,
      transferAccountId: null,
    })
  }

  results.sort((a, b) => b.date.localeCompare(a.date))
  duplicates.sort((a, b) => b.date.localeCompare(a.date))

  transactions.value = results
  duplicateTransactions.value = duplicates
  skippedRowCount.value = skipped
}

function goToPreview() {
  buildTransactions()
  step.value = 'preview'
}

function backToMapping() {
  step.value = 'mapping'
}

const SORT_ACCESSORS = {
  date: (row) => row.date,
  label: (row) => normalizeHeaderName(row.label),
  amount: (row) => row.amount,
  type: (row) => row.type,
  category: (row) => normalizeHeaderName(categoryNameForRow(row) || ''),
  subCategory: (row) => normalizeHeaderName(subCategoryNameForRow(row) || ''),
}

const sortColumn = ref('date')
const sortDirection = ref('desc')

function setSort(column) {
  if (!SORT_ACCESSORS[column]) return

  if (sortColumn.value === column) {
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortColumn.value = column
    sortDirection.value = 'asc'
  }
}

function sortIndicator(column) {
  if (sortColumn.value !== column) return ''
  return sortDirection.value === 'asc' ? '▲' : '▼'
}

const sortedTransactions = computed(() => {
  const getValue = SORT_ACCESSORS[sortColumn.value] || SORT_ACCESSORS.date
  const direction = sortDirection.value === 'desc' ? -1 : 1

  return [...transactions.value].sort((a, b) => {
    const valueA = getValue(a)
    const valueB = getValue(b)

    if (valueA < valueB) return -1 * direction
    if (valueA > valueB) return 1 * direction
    return 0
  })
})

const includedTransactions = computed(() => transactions.value.filter((row) => row.included))

const summary = computed(() => {
  const included = includedTransactions.value
  const purchaseRows = included.filter((row) => row.type === 'achat')
  const incomeRows = included.filter((row) => row.type === 'revenu')

  return {
    purchaseCount: purchaseRows.length,
    incomeCount: incomeRows.length,
    total: included.reduce((sum, row) => {
      return sum + (row.type === 'achat' ? -row.amount : row.amount)
    }, 0),
  }
})

const missingCategoryCount = computed(() => {
  return includedTransactions.value.filter((row) => row.type === 'achat' && !row.categoryId && !row.isTransfer).length
})

const missingTransferAccountCount = computed(() => {
  return includedTransactions.value.filter((row) => row.isTransfer && !row.transferAccountId).length
})

const unmatchedBankCategories = computed(() => {
  const counts = new Map()

  for (const row of includedTransactions.value) {
    if (row.isTransfer || row.categoryId || !row.bankCategoryName) continue
    const key = `${row.type}::${row.bankCategoryName}`
    const existing = counts.get(key)

    if (existing) existing.count += 1
    else counts.set(key, { name: row.bankCategoryName, type: row.type, count: 1 })
  }

  return Array.from(counts.values())
})

async function createUnmatchedCategories() {
  isCreatingCategories.value = true

  for (const { name, type } of unmatchedBankCategories.value) {
    let categoryId = null

    try {
      const created = await store.createCategory({ name, type })
      categoryId = created.id
    } catch {
      categoryId = findMatchingCategoryId(name, type)
    }

    if (!categoryId) continue

    for (const row of transactions.value) {
      if (row.type === type && !row.categoryId && !row.isTransfer && row.bankCategoryName === name) {
        row.categoryId = categoryId
        row.subCategoryId = findMatchingSubCategoryId(categoryId, row.bankSubCategoryName)
      }
    }
  }

  isCreatingCategories.value = false
}

const canImport = computed(() => {
  return (
    includedTransactions.value.length > 0 &&
    missingCategoryCount.value === 0 &&
    missingTransferAccountCount.value === 0 &&
    Boolean(activeAccountId.value) &&
    !isImporting.value
  )
})

function scrollToFirstMissingCategory() {
  const row = includedTransactions.value.find(
    (item) => (item.type === 'achat' && !item.categoryId && !item.isTransfer) || (item.isTransfer && !item.transferAccountId)
  )
  if (!row) return

  const el = document.getElementById(`import-row-${row.id}`)
  if (!el) return

  el.scrollIntoView({ behavior: 'smooth', block: 'center' })
  el.classList.add('is-flagged')
  window.setTimeout(() => el.classList.remove('is-flagged'), 1600)
}

function toggleAll(value) {
  for (const row of transactions.value) row.included = value
}

function toggleTransferMode(row) {
  row.isTransfer = !row.isTransfer

  if (row.isTransfer) {
    row.transferAccountId = otherAccountsList.value[0]?.id ?? null
  } else {
    row.transferAccountId = null
  }
}

function handleCategoryChange(row, event) {
  const rawValue = event.target.value

  if (rawValue === '__new__') {
    event.target.value = row.categoryId ?? ''
    categoryModalError.value = ''
    categoryModalRow.value = row
    isCategoryModalOpen.value = true
    return
  }

  row.categoryId = rawValue ? Number(rawValue) : null
  row.subCategoryId = findMatchingSubCategoryId(row.categoryId, row.bankSubCategoryName)
  row.isSuggested = false
}

async function confirmCreateCategory(name) {
  const row = categoryModalRow.value
  if (!row) return

  isCreatingCategoryInline.value = true
  categoryModalError.value = ''

  try {
    const created = await store.createCategory({ name, type: row.type })
    row.categoryId = created.id
    row.subCategoryId = findMatchingSubCategoryId(row.categoryId, row.bankSubCategoryName)
    row.isSuggested = false
    isCategoryModalOpen.value = false
    categoryModalRow.value = null
  } catch (err) {
    categoryModalError.value = err instanceof Error ? err.message : 'Impossible de créer la catégorie.'
  } finally {
    isCreatingCategoryInline.value = false
  }
}

function handleSubCategoryChange(row, event) {
  const rawValue = event.target.value

  if (rawValue === '__new__') {
    event.target.value = row.subCategoryId ?? ''
    if (!row.categoryId) return
    subCategoryModalError.value = ''
    subCategoryModalRow.value = row
    isSubCategoryModalOpen.value = true
    return
  }

  row.subCategoryId = rawValue ? Number(rawValue) : null
}

async function confirmCreateSubCategory(name) {
  const row = subCategoryModalRow.value
  if (!row) return

  isCreatingSubCategoryInline.value = true
  subCategoryModalError.value = ''

  try {
    const created = await store.createSubCategory(row.categoryId, name)
    row.subCategoryId = created.id
    isSubCategoryModalOpen.value = false
    subCategoryModalRow.value = null
  } catch (err) {
    subCategoryModalError.value = err instanceof Error ? err.message : 'Impossible de créer la sous-catégorie.'
  } finally {
    isCreatingSubCategoryInline.value = false
  }
}

async function confirmImport() {
  if (!canImport.value) return

  isImporting.value = true

  let successCount = 0
  let failCount = 0

  for (const row of includedTransactions.value) {
    row.error = ''

    try {
      if (row.isTransfer) {
        const isOutgoing = row.type === 'achat'
        await store.createTransfer({
          fromAccountId: isOutgoing ? activeAccountId.value : row.transferAccountId,
          toAccountId: isOutgoing ? row.transferAccountId : activeAccountId.value,
          amount: row.amount,
          date: row.date,
          note: row.label,
          fromIsReconciled: isOutgoing ? row.isReconciled : false,
          fromStatementReference: isOutgoing ? row.reference : '',
          toIsReconciled: !isOutgoing ? row.isReconciled : false,
          toStatementReference: !isOutgoing ? row.reference : '',
        })
      } else if (row.type === 'achat') {
        await store.submitPurchase({
          merchant: row.label,
          categoryId: row.categoryId,
          accountId: activeAccountId.value,
          amount: row.amount,
          date: row.date,
          paymentMethod: row.paymentMethod || mapping.paymentMethod,
          note: '',
          reference: row.reference,
          operationLabel: row.operationLabel,
          additionalInfo: row.additionalInfo,
          subCategory: subCategoryNameForRow(row),
          operationDate: row.operationDate,
          valueDate: row.valueDate,
          isReconciled: row.isReconciled,
        })
      } else {
        await store.createIncome({
          source: row.label,
          accountId: activeAccountId.value,
          amount: row.amount,
          income_date: row.date,
          note: '',
          reference: row.reference,
          operationLabel: row.operationLabel,
          additionalInfo: row.additionalInfo,
          operationType: row.operationType,
          category: categoryNameForRow(row),
          subCategory: subCategoryNameForRow(row),
          operationDate: row.operationDate,
          valueDate: row.valueDate,
          isReconciled: row.isReconciled,
        })
      }

      successCount += 1
    } catch (err) {
      failCount += 1
      row.error = err instanceof Error ? err.message : 'Échec de l’import.'
    }
  }

  isImporting.value = false
  importSummary.value = { successCount, failCount }

  emit('imported', { successCount, failCount })

  if (failCount === 0) {
    resetState()
  }
}
</script>

<template>
  <main class="dashboard-content import-view">
    <PageHero
      eyebrow="Récupérer des données"
      title="Import Achats / Revenus"
      description="Importe le relevé CSV de ta banque pour créer automatiquement tes achats et revenus."
    >
      <template #actions>
        <button
          v-if="step !== 'upload'"
          class="ghost-btn"
          type="button"
          :disabled="isImporting"
          @click="resetState"
        >
          Recommencer
        </button>
      </template>
    </PageHero>

    <section class="panel import-card">
      <section v-if="step === 'upload'" class="import-step">
        <div v-if="savedDraft" class="import-draft-banner">
          <div>
            <strong>Reprendre l'import en cours ?</strong>
            <p>
              {{ savedDraft.fileName }} · {{ savedDraft.transactions.length }} ligne{{ savedDraft.transactions.length > 1 ? 's' : '' }}
              · sauvegardé {{ formatDraftDate(savedDraft.savedAt) }}
            </p>
          </div>
          <div class="import-draft-actions">
            <button class="ghost-btn" type="button" @click="discardDraft">Ignorer</button>
            <button class="primary-btn" type="button" @click="resumeDraft">Reprendre</button>
          </div>
        </div>

        <label
          class="import-dropzone"
          :class="{ 'is-dragging': isDragging }"
          @dragover.prevent="isDragging = true"
          @dragleave.prevent="isDragging = false"
          @drop.prevent="handleDrop"
        >
          <input type="file" accept=".csv,text/csv" @change="handleFileInput" />
          <span class="import-dropzone-title">Dépose ton fichier CSV ici</span>
          <span class="import-dropzone-caption">ou clique pour choisir un fichier exporté par ta banque</span>
        </label>

        <p v-if="fileError" class="form-error">{{ fileError }}</p>
      </section>

      <section v-else-if="step === 'mapping'" class="import-step">
        <p class="import-step-caption">
          <strong>{{ fileName }}</strong> · {{ csvRows.length }} ligne{{ csvRows.length > 1 ? 's' : '' }} détectée{{ csvRows.length > 1 ? 's' : '' }}.
          Vérifie la correspondance des colonnes avant de continuer.
        </p>

        <p class="import-step-caption">
          Import dans le compte actif : <strong>{{ store.getAccountName(activeAccountId) }}</strong>
          (change-le depuis la barre latérale si besoin).
        </p>

        <div class="import-mapping-grid">
          <label class="form-field">
            <span>Colonne Date</span>
            <select v-model="mapping.dateColumn">
              <option value="">Choisir une colonne</option>
              <option v-for="header in csvHeaders" :key="header" :value="header">{{ header }}</option>
            </select>
          </label>

          <label class="form-field">
            <span>Colonne Libellé</span>
            <select v-model="mapping.labelColumn">
              <option value="">Choisir une colonne</option>
              <option v-for="header in csvHeaders" :key="header" :value="header">{{ header }}</option>
            </select>
          </label>

          <label class="form-field form-field-full">
            <span>Format du montant</span>
            <div class="import-radio-row">
              <label class="import-radio">
                <input type="radio" value="single" v-model="mapping.amountMode" />
                Une colonne signée
              </label>
              <label class="import-radio">
                <input type="radio" value="split" v-model="mapping.amountMode" />
                Débit / Crédit séparés
              </label>
            </div>
          </label>

          <label v-if="mapping.amountMode === 'single'" class="form-field">
            <span>Colonne Montant</span>
            <select v-model="mapping.amountColumn">
              <option value="">Choisir une colonne</option>
              <option v-for="header in csvHeaders" :key="header" :value="header">{{ header }}</option>
            </select>
          </label>

          <template v-else>
            <label class="form-field">
              <span>Colonne Débit</span>
              <select v-model="mapping.debitColumn">
                <option value="">Choisir une colonne</option>
                <option v-for="header in csvHeaders" :key="header" :value="header">{{ header }}</option>
              </select>
            </label>

            <label class="form-field">
              <span>Colonne Crédit</span>
              <select v-model="mapping.creditColumn">
                <option value="">Choisir une colonne</option>
                <option v-for="header in csvHeaders" :key="header" :value="header">{{ header }}</option>
              </select>
            </label>
          </template>

          <label class="form-field">
            <span>Colonne Catégorie (banque)</span>
            <select v-model="mapping.categoryColumn">
              <option value="">Aucune (choisir manuellement)</option>
              <option v-for="header in csvHeaders" :key="header" :value="header">{{ header }}</option>
            </select>
          </label>

          <label class="form-field">
            <span>Mode de paiement par défaut (achats)</span>
            <select v-model="mapping.paymentMethod">
              <option v-for="method in PAYMENT_METHODS" :key="method" :value="method">{{ method }}</option>
            </select>
          </label>

          <p class="import-step-caption form-field-full">
            Champs additionnels détectés automatiquement (modifiables si besoin) :
          </p>

          <label class="form-field">
            <span>Colonne Référence</span>
            <select v-model="mapping.referenceColumn">
              <option value="">Aucune</option>
              <option v-for="header in csvHeaders" :key="header" :value="header">{{ header }}</option>
            </select>
          </label>

          <label class="form-field">
            <span>Colonne Libellé opération</span>
            <select v-model="mapping.operationLabelColumn">
              <option value="">Aucune</option>
              <option v-for="header in csvHeaders" :key="header" :value="header">{{ header }}</option>
            </select>
          </label>

          <label class="form-field">
            <span>Colonne Informations complémentaires</span>
            <select v-model="mapping.additionalInfoColumn">
              <option value="">Aucune</option>
              <option v-for="header in csvHeaders" :key="header" :value="header">{{ header }}</option>
            </select>
          </label>

          <label class="form-field">
            <span>Colonne Sous-catégorie</span>
            <select v-model="mapping.subCategoryColumn">
              <option value="">Aucune</option>
              <option v-for="header in csvHeaders" :key="header" :value="header">{{ header }}</option>
            </select>
          </label>

          <label class="form-field">
            <span>Colonne Type d'opération</span>
            <select v-model="mapping.typeOperationColumn">
              <option value="">Aucune</option>
              <option v-for="header in csvHeaders" :key="header" :value="header">{{ header }}</option>
            </select>
          </label>

          <label class="form-field">
            <span>Colonne Date opération</span>
            <select v-model="mapping.operationDateColumn">
              <option value="">Aucune</option>
              <option v-for="header in csvHeaders" :key="header" :value="header">{{ header }}</option>
            </select>
          </label>

          <label class="form-field">
            <span>Colonne Date de valeur</span>
            <select v-model="mapping.valueDateColumn">
              <option value="">Aucune</option>
              <option v-for="header in csvHeaders" :key="header" :value="header">{{ header }}</option>
            </select>
          </label>

          <label class="form-field">
            <span>Colonne Pointage</span>
            <select v-model="mapping.reconciledColumn">
              <option value="">Aucune</option>
              <option v-for="header in csvHeaders" :key="header" :value="header">{{ header }}</option>
            </select>
          </label>
        </div>

        <p class="import-step-caption">
          Si la banque fournit une colonne catégorie (et sous-catégorie), elle sera reliée automatiquement à une catégorie existante du même nom et du bon type (achat ou revenu) ; tu pourras corriger ou en créer une nouvelle à l'étape suivante si le rapprochement est incorrect ou absent.
          Pour les achats, la colonne « Type d'opération » remplace le mode de paiement par défaut quand elle est renseignée.
        </p>

        <div class="modal-actions">
          <button class="ghost-btn" type="button" @click="step = 'upload'">Retour</button>
          <button
            class="primary-btn"
            type="button"
            :disabled="!mapping.dateColumn || !mapping.labelColumn || (mapping.amountMode === 'single' ? !mapping.amountColumn : !mapping.debitColumn && !mapping.creditColumn)"
            @click="goToPreview"
          >
            Continuer
          </button>
        </div>
      </section>

      <section v-else class="import-step">
        <p class="import-step-caption">
          {{ summary.purchaseCount }} achat{{ summary.purchaseCount > 1 ? 's' : '' }} ·
          {{ summary.incomeCount }} revenu{{ summary.incomeCount > 1 ? 's' : '' }} ·
          solde {{ formatCurrency(summary.total) }}
          <span v-if="skippedRowCount">· {{ skippedRowCount }} ligne{{ skippedRowCount > 1 ? 's' : '' }} ignorée{{ skippedRowCount > 1 ? 's' : '' }} (date/montant illisible)</span>
          <span v-if="duplicateTransactions.length">· {{ duplicateTransactions.length }} ligne{{ duplicateTransactions.length > 1 ? 's' : '' }} déjà importée{{ duplicateTransactions.length > 1 ? 's' : '' }} (voir ci-dessous)</span>
        </p>

        <p class="import-step-caption">
          Catégorie et sous-catégorie sont rapprochées automatiquement quand la banque fournit un nom correspondant. Corrige la sélection ou crée une nouvelle catégorie/sous-catégorie si le rapprochement te semble incorrect.
          Les catégories proposées correspondent au type de la ligne (achat ou revenu).
        </p>

        <div class="import-select-all">
          <button class="ghost-btn" type="button" @click="toggleAll(true)">Tout inclure</button>
          <button class="ghost-btn" type="button" @click="toggleAll(false)">Tout exclure</button>
        </div>

        <div class="import-table-wrap">
          <table class="import-table">
            <thead>
              <tr>
                <th></th>
                <th class="sortable" @click="setSort('date')">
                  Date <span class="sort-indicator">{{ sortIndicator('date') }}</span>
                </th>
                <th class="sortable" @click="setSort('label')">
                  Libellé <span class="sort-indicator">{{ sortIndicator('label') }}</span>
                </th>
                <th class="sortable" @click="setSort('amount')">
                  Montant <span class="sort-indicator">{{ sortIndicator('amount') }}</span>
                </th>
                <th class="sortable" @click="setSort('type')">
                  Type <span class="sort-indicator">{{ sortIndicator('type') }}</span>
                </th>
                <th class="sortable" @click="setSort('category')">
                  Catégorie / Source <span class="sort-indicator">{{ sortIndicator('category') }}</span>
                </th>
                <th class="sortable" @click="setSort('subCategory')">
                  Sous-catégorie <span class="sort-indicator">{{ sortIndicator('subCategory') }}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="row in sortedTransactions"
                :id="`import-row-${row.id}`"
                :key="row.id"
                :class="{ 'is-excluded': !row.included }"
              >
                <td>
                  <input type="checkbox" v-model="row.included" />
                </td>
                <td>{{ formatDate(row.date) }}</td>
                <td class="import-label-cell" :title="row.label">{{ row.label }}</td>
                <td :class="row.type === 'achat' ? 'amount-negative' : 'amount-positive'">
                  {{ row.type === 'achat' ? '−' : '+' }}{{ formatCurrency(row.amount) }}
                </td>
                <td>
                  <span
                    class="import-type-badge"
                    :class="row.isTransfer ? 'is-transfer' : row.type === 'achat' ? 'is-achat' : 'is-revenu'"
                  >
                    {{ row.isTransfer ? 'Virement' : row.type === 'achat' ? 'Achat' : 'Revenu' }}
                  </span>
                </td>
                <td>
                  <template v-if="row.isTransfer">
                    <select
                      v-model="row.transferAccountId"
                      :class="{ 'field-missing': row.included && !row.transferAccountId }"
                    >
                      <option :value="null">Choisir un compte</option>
                      <option v-for="account in otherAccountsList" :key="account.id" :value="account.id">
                        {{ account.name }}
                      </option>
                    </select>
                    <p class="import-transfer-toggle">
                      <a href="#" @click.prevent="toggleTransferMode(row)">↩ Achat/revenu</a>
                    </p>
                    <p v-if="row.error" class="import-row-error">{{ row.error }}</p>
                  </template>
                  <template v-else>
                    <select
                      :value="row.categoryId"
                      :class="{ 'field-missing': row.included && !row.categoryId }"
                      @change="handleCategoryChange(row, $event)"
                    >
                      <option :value="null">Choisir une catégorie</option>
                      <option v-for="category in categoriesForRow(row)" :key="category.id" :value="category.id">
                        {{ category.name }}
                      </option>
                      <option value="__new__">+ Créer une nouvelle catégorie…</option>
                    </select>
                    <p v-if="row.bankCategoryName && !row.categoryId" class="import-bank-hint">
                      Banque : {{ row.bankCategoryName }}
                    </p>
                    <p v-if="row.isSuggested" class="import-suggested-hint" title="Basé sur un import précédent pour ce même libellé — vérifie avant de valider">
                      🔮 Suggéré, à vérifier
                    </p>
                    <p v-if="otherAccountsList.length" class="import-transfer-toggle">
                      <a href="#" @click.prevent="toggleTransferMode(row)">↔ Virement</a>
                    </p>
                    <p v-if="row.error" class="import-row-error">{{ row.error }}</p>
                  </template>
                </td>
                <td>
                  <select
                    :value="row.subCategoryId"
                    :disabled="!row.categoryId"
                    @change="handleSubCategoryChange(row, $event)"
                  >
                    <option :value="null">Aucune sous-catégorie</option>
                    <option v-for="sub in subCategoriesForRow(row)" :key="sub.id" :value="sub.id">
                      {{ sub.name }}
                    </option>
                    <option value="__new__">+ Créer une nouvelle sous-catégorie…</option>
                  </select>
                  <p v-if="row.bankSubCategoryName && !row.subCategoryId" class="import-bank-hint">
                    Banque : {{ row.bankSubCategoryName }}
                  </p>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-if="duplicateTransactions.length" class="import-duplicates-section">
          <p class="import-step-caption">
            {{ duplicateTransactions.length }} ligne{{ duplicateTransactions.length > 1 ? 's' : '' }} déjà importée{{ duplicateTransactions.length > 1 ? 's' : '' }} (référence déjà connue) — ignorée{{ duplicateTransactions.length > 1 ? 's' : '' }} automatiquement :
          </p>

          <div class="import-duplicates-list">
            <div v-for="row in duplicateTransactions" :key="row.id" class="import-duplicate-row">
              <span>{{ formatDate(row.date) }}</span>
              <span class="import-duplicate-label" :title="row.label">{{ row.label }}</span>
              <span :class="row.type === 'achat' ? 'amount-negative' : 'amount-positive'">
                {{ row.type === 'achat' ? '−' : '+' }}{{ formatCurrency(row.amount) }}
              </span>
              <span class="import-duplicate-reference">Réf. {{ row.reference }}</span>
            </div>
          </div>
        </div>

        <p v-if="importSummary && importSummary.failCount" class="form-error">
          {{ importSummary.successCount }} ligne{{ importSummary.successCount > 1 ? 's' : '' }} importée{{ importSummary.successCount > 1 ? 's' : '' }},
          {{ importSummary.failCount }} échec{{ importSummary.failCount > 1 ? 's' : '' }} (voir détail dans le tableau).
        </p>

        <p v-if="unmatchedBankCategories.length" class="import-create-banner">
          {{ unmatchedBankCategories.length }} catégorie{{ unmatchedBankCategories.length > 1 ? 's' : '' }} bancaire{{ unmatchedBankCategories.length > 1 ? 's' : '' }} inconnue{{ unmatchedBankCategories.length > 1 ? 's' : '' }} détectée{{ unmatchedBankCategories.length > 1 ? 's' : '' }} :
          <strong>{{ unmatchedBankCategories.map((c) => c.name).join(', ') }}</strong>.
          <button
            class="ghost-btn import-create-btn"
            type="button"
            :disabled="isCreatingCategories"
            @click="createUnmatchedCategories"
          >
            {{ isCreatingCategories ? 'Création…' : unmatchedBankCategories.length > 1 ? `Créer ces ${unmatchedBankCategories.length} catégories` : 'Créer cette catégorie' }}
          </button>
        </p>

        <p v-if="missingCategoryCount || missingTransferAccountCount" class="import-missing-warning">
          <template v-if="missingCategoryCount">
            {{ missingCategoryCount }} achat{{ missingCategoryCount > 1 ? 's' : '' }} sans catégorie
          </template>
          <template v-if="missingCategoryCount && missingTransferAccountCount"> et </template>
          <template v-if="missingTransferAccountCount">
            {{ missingTransferAccountCount }} virement{{ missingTransferAccountCount > 1 ? 's' : '' }} sans compte lié
          </template>
          — l'import reste désactivé tant qu'ils n'en ont pas.
          <button class="ghost-btn import-missing-jump" type="button" @click="scrollToFirstMissingCategory">
            Voir la première ligne concernée
          </button>
        </p>

        <div class="modal-actions">
          <button class="ghost-btn" type="button" @click="backToMapping" :disabled="isImporting">Retour</button>
          <button class="primary-btn" type="button" :disabled="!canImport" @click="confirmImport">
            {{ isImporting ? 'Import en cours…' : `Importer ${includedTransactions.length} ligne${includedTransactions.length > 1 ? 's' : ''}` }}
          </button>
        </div>
      </section>
    </section>

    <QuickCreateModal
      v-model="isCategoryModalOpen"
      eyebrow="Import"
      title="Nouvelle catégorie"
      label="Nom de la catégorie"
      :placeholder="categoryModalRow?.bankCategoryName || 'Ex: Alimentation'"
      :initial-value="categoryModalRow?.bankCategoryName || ''"
      confirm-label="Créer la catégorie"
      :is-submitting="isCreatingCategoryInline"
      :error-message="categoryModalError"
      @confirm="confirmCreateCategory"
    />

    <QuickCreateModal
      v-model="isSubCategoryModalOpen"
      eyebrow="Import"
      title="Nouvelle sous-catégorie"
      label="Nom de la sous-catégorie"
      :placeholder="subCategoryModalRow?.bankSubCategoryName || 'Ex: Courses alimentaires'"
      :initial-value="subCategoryModalRow?.bankSubCategoryName || ''"
      confirm-label="Créer la sous-catégorie"
      :is-submitting="isCreatingSubCategoryInline"
      :error-message="subCategoryModalError"
      @confirm="confirmCreateSubCategory"
    />
  </main>
</template>

<style scoped>
.import-view {
  display: grid;
  gap: 0.9rem;
}

.import-card {
  padding: 1.1rem;
}

.import-step {
  display: grid;
  gap: 1rem;
}

.import-step-caption {
  margin: 0;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.88rem;
}

.import-draft-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 0.9rem;
  padding: 0.9rem 1.1rem;
  border-radius: 14px;
  background: rgba(143, 168, 160, 0.1);
  border: 1px solid rgba(143, 168, 160, 0.28);
}

.import-draft-banner strong {
  color: var(--text, #eef1f3);
  font-size: 0.95rem;
}

.import-draft-banner p {
  margin: 0.25rem 0 0;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.82rem;
}

.import-draft-actions {
  display: flex;
  gap: 0.6rem;
  flex-shrink: 0;
}

.import-dropzone {
  position: relative;
  display: grid;
  place-items: center;
  gap: 0.4rem;
  padding: 2.4rem 1.2rem;
  border-radius: 18px;
  border: 1px dashed rgba(var(--tint-rgb), 0.16);
  background: rgba(var(--tint-rgb), 0.03);
  text-align: center;
  cursor: pointer;
  transition: border-color 140ms ease, background 140ms ease;
}

.import-dropzone.is-dragging {
  border-color: rgba(143, 168, 160, 0.5);
  background: rgba(143, 168, 160, 0.08);
}

.import-dropzone input[type='file'] {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}

.import-dropzone-title {
  color: var(--text, #eef1f3);
  font-weight: 600;
}

.import-dropzone-caption {
  color: var(--text-dim, #8a939d);
  font-size: 0.84rem;
}

.import-mapping-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.9rem;
}

.import-radio-row {
  display: flex;
  gap: 1rem;
  flex-wrap: wrap;
}

.import-radio {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  color: var(--text, #eef1f3);
  font-size: 0.9rem;
  cursor: pointer;
}

.import-select-all {
  display: flex;
  gap: 0.6rem;
}

.import-table-wrap {
  max-height: 480px;
  overflow: auto;
  border-radius: 14px;
  border: 1px solid rgba(var(--tint-rgb), 0.06);
}

.import-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.84rem;
}

.import-table th {
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
}

.import-table th.sortable {
  cursor: pointer;
  user-select: none;
  transition: color 140ms ease;
}

.import-table th.sortable:hover {
  color: var(--text, #eef1f3);
}

.import-table .sort-indicator {
  display: inline-block;
  width: 0.7em;
  font-size: 0.7rem;
}

.import-table td {
  padding: 0.5rem 0.6rem;
  border-top: 1px solid rgba(var(--tint-rgb), 0.05);
  color: var(--text, #eef1f3);
  vertical-align: middle;
}

.import-table select,
.import-table input[type='text'] {
  width: 100%;
  min-width: 160px;
  height: 32px;
  padding: 0 0.5rem;
  border-radius: 8px;
  border: 1px solid rgba(var(--tint-rgb), 0.1);
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--text, #eef1f3);
  font-size: 0.82rem;
}

.import-table select.field-missing {
  border-color: rgba(220, 38, 38, 0.5);
}

.import-table select:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.import-subcategory-na {
  color: var(--text-dim, #8a939d);
}

.import-label-cell {
  max-width: 260px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.import-row-error {
  margin: 0.25rem 0 0;
  color: var(--negative-text);
  font-size: 0.74rem;
}

.import-bank-hint {
  margin: 0.25rem 0 0;
  color: var(--text-dim, #8a939d);
  font-size: 0.72rem;
  font-style: italic;
}

.import-suggested-hint {
  margin: 0.25rem 0 0;
  color: var(--positive-text, #bfe0c9);
  font-size: 0.72rem;
  cursor: help;
}

.import-duplicates-section {
  display: grid;
  gap: 0.5rem;
}

.import-duplicates-list {
  display: grid;
  gap: 0.35rem;
  max-height: 200px;
  overflow-y: auto;
  padding: 0.6rem 0.7rem;
  border-radius: 12px;
  background: rgba(220, 38, 38, 0.06);
  border: 1px solid rgba(220, 38, 38, 0.16);
}

.import-duplicate-row {
  display: grid;
  grid-template-columns: 90px 1fr auto 160px;
  gap: 0.8rem;
  align-items: center;
  font-size: 0.82rem;
  color: var(--text-soft, #b3bbc4);
  opacity: 0.8;
}

.import-duplicate-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.import-duplicate-reference {
  color: var(--text-dim, #8a939d);
  font-size: 0.74rem;
  text-align: right;
  white-space: nowrap;
}

.is-excluded {
  opacity: 0.45;
}

.is-flagged {
  animation: import-row-flag 1.6s ease;
}

@keyframes import-row-flag {
  0%,
  100% {
    background: transparent;
  }
  20%,
  60% {
    background: rgba(220, 38, 38, 0.18);
  }
}

.import-missing-warning {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.6rem;
  margin: 0;
  padding: 0.7rem 0.9rem;
  border-radius: 12px;
  background: rgba(220, 38, 38, 0.1);
  border: 1px solid rgba(220, 38, 38, 0.22);
  color: var(--negative-text);
  font-size: 0.86rem;
}

.import-missing-jump {
  height: 32px;
  padding: 0 0.8rem;
  font-size: 0.8rem;
  flex-shrink: 0;
}

.import-create-banner {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.6rem;
  margin: 0;
  padding: 0.7rem 0.9rem;
  border-radius: 12px;
  background: rgba(143, 168, 160, 0.1);
  border: 1px solid rgba(143, 168, 160, 0.28);
  color: var(--text-soft, #b3bbc4);
  font-size: 0.86rem;
}

.import-create-btn {
  height: 32px;
  padding: 0 0.8rem;
  font-size: 0.8rem;
  flex-shrink: 0;
}

.import-table td.amount-negative {
  color: var(--negative-text);
  font-weight: 600;
  white-space: nowrap;
}

.import-table td.amount-positive {
  color: var(--positive-text);
  font-weight: 600;
  white-space: nowrap;
}

.import-type-badge {
  display: inline-block;
  padding: 0.25rem 0.65rem;
  border-radius: 999px;
  font-size: 0.78rem;
  font-weight: 600;
  white-space: nowrap;
}

.import-type-badge.is-achat {
  background: rgba(220, 38, 38, 0.12);
  color: var(--negative-text);
}

.import-type-badge.is-revenu {
  background: rgba(34, 197, 94, 0.12);
  color: var(--positive-text);
}

.import-type-badge.is-transfer {
  background: rgba(92, 200, 209, 0.14);
  color: #5cc8d1;
}

.import-transfer-toggle {
  margin: 0.3rem 0 0;
  font-size: 0.78rem;
}

.import-transfer-toggle a {
  color: var(--text-dim, #8a939d);
  text-decoration: underline;
  cursor: pointer;
}

@media (max-width: 680px) {
  .import-mapping-grid {
    grid-template-columns: 1fr;
  }
}
</style>
