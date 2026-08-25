<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import {
  Chart as ChartJS,
  Title,
  Tooltip,
  Legend,
  CategoryScale,
  LinearScale,
  BarElement,
  ArcElement,
  BarController,
  DoughnutController,
} from 'chart.js'
import jsPDF from 'jspdf'
import autoTable from 'jspdf-autotable'

import PageHero from '../Common/PageHero.vue'
import MultiSelectDropdown from '../Common/MultiSelectDropdown.vue'
import { usePurchasesStore } from '../../stores/purchases'
import {
  fetchLatestReportMetadata,
  uploadGeneratedReport,
  viewLatestReportPdf,
} from '../../services/reports'
import { formatCurrency } from '../../utils/format'
import { signTransferLeg } from '../../utils/realBalance'

ChartJS.register(
  Title,
  Tooltip,
  Legend,
  CategoryScale,
  LinearScale,
  BarElement,
  ArcElement,
  BarController,
  DoughnutController
)

const CHART_COLORS = ['#8fa8a0', '#a9b8c6', '#dcbf8a', '#96b4a7', '#b19dcb', '#708999', '#c99478']

const store = usePurchasesStore()
const { purchases, incomes, categoriesList, subCategoriesList, transfers, accountsList } = storeToRefs(store)

// Un Livret n'a jamais d'achat/revenu — l'écran bascule sur un rapport
// crédit/débit basé sur ses virements, en réutilisant telle quelle toute la
// mécanique de filtrage/graphiques/PDF déjà écrite pour achats/revenus (un
// virement sortant "devient" un achat, un virement entrant "devient" un
// revenu, le compte lié tenant lieu de catégorie/source) — seuls les
// libellés affichés changent.
const activeAccount = computed(() =>
  accountsList.value.find((account) => Number(account.id) === Number(store.activeAccountId)) || null
)
const isSimplified = computed(() => activeAccount.value?.hasStatements === false)

// Même règle que computeRealBalance/le Dashboard : un solde initial déclaré
// n'ancre le calcul que sur les virements enregistrés DEPUIS sa date — ceux
// d'avant sont déjà représentés dedans, les compter en plus doublerait.
const openingBalanceSinceDate = computed(() => activeAccount.value?.openingBalanceDate || null)

const transferLegs = computed(() => {
  const sinceDate = openingBalanceSinceDate.value

  return transfers.value
    .map((t) => {
      const leg = signTransferLeg(t, store.activeAccountId)
      return {
        ...leg,
        otherAccountName: leg.isOutgoing ? t.toAccountName : t.fromAccountName,
        // jsPDF (polices standard, sans les 14 fonts) ne sait pas rendre
        // U+2192 « → » — le glyphe manquant casse aussi le calcul de largeur
        // de toute la cellule (lettres étirées sur toute la ligne). D'où un
        // séparateur ASCII ici, réservé au PDF (l'affichage à l'écran
        // ailleurs dans l'app utilise « → » sans souci, rendu par le
        // navigateur).
        pairLabel: `${t.fromAccountName} -> ${t.toAccountName}`,
      }
    })
    .filter((leg) => !sinceDate || leg.date >= sinceDate)
})

// Le solde initial compte comme un crédit (ou un débit s'il est négatif) au
// même titre qu'un virement, daté du jour où il a été déclaré — il traverse
// alors tel quel tout le filtrage par période/compte déjà en place (y
// compris son exclusion si la période choisie ne le couvre pas), sans
// mécanique séparée à écrire.
const openingBalanceItem = computed(() => {
  const account = activeAccount.value
  if (!account?.openingBalanceDate || account.openingBalanceAmount == null) return null

  return { date: account.openingBalanceDate, amount: Number(account.openingBalanceAmount) }
})

const debitItems = computed(() => {
  const items = transferLegs.value
    .filter((leg) => leg.isOutgoing)
    .map((leg) => ({
      date: leg.date,
      amount: Math.abs(leg.amount),
      merchant: leg.otherAccountName,
      category: leg.otherAccountName,
      accountPair: leg.pairLabel,
      subCategory: '',
      note: '',
    }))

  const opening = openingBalanceItem.value
  if (opening && opening.amount < 0) {
    items.push({
      date: opening.date,
      amount: Math.abs(opening.amount),
      merchant: 'Solde initial',
      category: 'Solde initial',
      subCategory: '',
      note: '',
    })
  }

  return items
})

const creditItems = computed(() => {
  const items = transferLegs.value
    .filter((leg) => !leg.isOutgoing)
    .map((leg) => ({
      income_date: leg.date,
      amount: leg.amount,
      source: leg.otherAccountName,
      category: leg.otherAccountName,
      accountPair: leg.pairLabel,
      subCategory: '',
      note: '',
      operationLabel: '',
    }))

  const opening = openingBalanceItem.value
  if (opening && opening.amount > 0) {
    items.push({
      income_date: opening.date,
      amount: opening.amount,
      source: 'Solde initial',
      category: 'Solde initial',
      subCategory: '',
      note: '',
      operationLabel: '',
    })
  }

  return items
})

const reportType = ref('both')
// "Cette année" par défaut (pas "Tout") — générer un rapport sur tout
// l'historique peut inclure des milliers de lignes et rendre la génération
// très longue ; l'utilisateur peut toujours choisir "Tout" explicitement.
const reportPeriod = ref('ytd')
const reportCustomStart = ref('')
const reportCustomEnd = ref('')
const reportCategories = ref([])
const reportSubCategories = ref([])

watch(reportType, () => {
  reportCategories.value = []
  reportSubCategories.value = []
})

watch(reportCategories, () => {
  reportSubCategories.value = []
})

const periodOptions = [
  { value: '30d', label: '30 jours' },
  { value: '90d', label: '90 jours' },
  { value: '12m', label: '12 mois' },
  { value: 'ytd', label: 'Cette année' },
  { value: 'all', label: 'Tout' },
  { value: 'custom', label: 'Période personnalisée' },
]

function isWithinRange(dateValue, range) {
  if (range === 'all') return true

  const date = new Date(dateValue)
  if (Number.isNaN(date.getTime())) return false

  if (range === 'custom') {
    if (reportCustomStart.value) {
      const start = new Date(reportCustomStart.value)
      start.setHours(0, 0, 0, 0)
      if (date < start) return false
    }

    if (reportCustomEnd.value) {
      const end = new Date(reportCustomEnd.value)
      end.setHours(23, 59, 59, 999)
      if (date > end) return false
    }

    return true
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

const includesPurchases = computed(() => reportType.value === 'both' || reportType.value === 'achats')
const includesIncomes = computed(() => reportType.value === 'both' || reportType.value === 'revenus')

// Liste combinée : catégories d'achats (référentiel categoriesList) + valeurs
// de catégorie libres relevées sur les revenus — pour que la case à cocher
// filtre les deux types de transactions en une seule sélection, y compris
// quand le type choisi est "Achats et revenus".
const categoryFilterOptions = computed(() => {
  const values = new Set()

  if (isSimplified.value) {
    // Un virement n'a pas de catégorie — le compte lié en tient lieu.
    if (includesPurchases.value) {
      for (const item of debitItems.value) values.add(item.category)
    }
    if (includesIncomes.value) {
      for (const item of creditItems.value) values.add(item.category)
    }

    return Array.from(values)
      .sort((a, b) => a.localeCompare(b, 'fr', { sensitivity: 'base' }))
      .map((value) => ({ value, label: value }))
  }

  if (includesPurchases.value) {
    for (const category of categoriesList.value) {
      values.add(category.name)
    }
  }

  if (includesIncomes.value) {
    for (const income of incomes.value) {
      const value = income.category?.trim()
      if (value) values.add(value)
    }
  }

  return Array.from(values)
    .sort((a, b) => a.localeCompare(b, 'fr', { sensitivity: 'base' }))
    .map((value) => ({ value, label: value }))
})

// Un groupe par catégorie cochée, avec ses propres sous-catégories — la
// clé de chaque option combine catégorie + sous-catégorie ("Catégorie||Sous")
// pour que deux catégories partageant un nom de sous-catégorie (ex. "Divers")
// restent des cases indépendantes.
const subCategoryGroups = computed(() => {
  if (!reportCategories.value.length) return []

  const groups = []

  for (const categoryName of reportCategories.value) {
    const subNames = new Set()

    if (includesPurchases.value) {
      const category = categoriesList.value.find((c) => c.name === categoryName)
      if (category) {
        for (const sub of subCategoriesList.value) {
          if (Number(sub.categoryId) === Number(category.id)) subNames.add(sub.name)
        }
      }
    }

    if (includesIncomes.value) {
      for (const income of incomes.value) {
        if ((income.category?.trim() || '') !== categoryName) continue
        const value = income.subCategory?.trim()
        if (value) subNames.add(value)
      }
    }

    if (!subNames.size) continue

    groups.push({
      key: categoryName,
      label: categoryName,
      options: Array.from(subNames)
        .sort((a, b) => a.localeCompare(b, 'fr', { sensitivity: 'base' }))
        .map((subName) => ({ value: `${categoryName}||${subName}`, label: subName })),
    })
  }

  return groups.sort((a, b) => a.label.localeCompare(b.label, 'fr', { sensitivity: 'base' }))
})

const filteredReportPurchases = computed(() => {
  if (!includesPurchases.value) return []

  const source = isSimplified.value ? debitItems.value : purchases.value

  return source.filter((purchase) => {
    if (!isWithinRange(purchase.date, reportPeriod.value)) return false

    if (reportCategories.value.length) {
      const category = purchase.category?.trim() || 'Sans catégorie'
      if (!reportCategories.value.includes(category)) return false

      if (reportSubCategories.value.length) {
        const subCategory = purchase.subCategory?.trim() || 'Sans sous-catégorie'
        if (!reportSubCategories.value.includes(`${category}||${subCategory}`)) return false
      }
    }

    return true
  })
})

const filteredReportIncomes = computed(() => {
  if (!includesIncomes.value) return []

  const source = isSimplified.value ? creditItems.value : incomes.value

  return source.filter((income) => {
    if (!isWithinRange(income.income_date, reportPeriod.value)) return false

    if (reportCategories.value.length) {
      const category = income.category?.trim() || 'Sans catégorie'
      if (!reportCategories.value.includes(category)) return false

      if (reportSubCategories.value.length) {
        const subCategory = income.subCategory?.trim() || 'Sans sous-catégorie'
        if (!reportSubCategories.value.includes(`${category}||${subCategory}`)) return false
      }
    }

    return true
  })
})

const totalPurchasesAmount = computed(() =>
  filteredReportPurchases.value.reduce((sum, purchase) => sum + Number(purchase.amount || 0), 0)
)
const totalIncomesAmount = computed(() =>
  filteredReportIncomes.value.reduce((sum, income) => sum + Number(income.amount || 0), 0)
)
const netBalance = computed(() => totalIncomesAmount.value - totalPurchasesAmount.value)
const purchaseCount = computed(() => filteredReportPurchases.value.length)
const incomeCount = computed(() => filteredReportIncomes.value.length)
const averagePurchase = computed(() =>
  purchaseCount.value > 0 ? totalPurchasesAmount.value / purchaseCount.value : 0
)
const averageIncome = computed(() =>
  incomeCount.value > 0 ? totalIncomesAmount.value / incomeCount.value : 0
)

function monthKey(date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`
}

function formatMonthLabel(date) {
  const formatted = new Intl.DateTimeFormat('fr-FR', { month: 'short', year: '2-digit' }).format(date)
  return formatted.charAt(0).toUpperCase() + formatted.slice(1)
}

const monthlyEvolution = computed(() => {
  const spendingByMonth = new Map()
  const incomeByMonth = new Map()

  for (const purchase of filteredReportPurchases.value) {
    if (!purchase.date) continue
    const date = new Date(purchase.date)
    if (Number.isNaN(date.getTime())) continue
    const key = monthKey(date)
    spendingByMonth.set(key, (spendingByMonth.get(key) || 0) + Number(purchase.amount || 0))
  }

  for (const income of filteredReportIncomes.value) {
    if (!income.income_date) continue
    const date = new Date(income.income_date)
    if (Number.isNaN(date.getTime())) continue
    const key = monthKey(date)
    incomeByMonth.set(key, (incomeByMonth.get(key) || 0) + Number(income.amount || 0))
  }

  const keys = Array.from(new Set([...spendingByMonth.keys(), ...incomeByMonth.keys()])).sort()

  const labels = keys.map((key) => {
    const [year, month] = key.split('-').map(Number)
    return formatMonthLabel(new Date(year, month - 1, 1))
  })
  const spending = keys.map((key) => spendingByMonth.get(key) || 0)
  const income = keys.map((key) => incomeByMonth.get(key) || 0)

  return { labels, spending, income }
})

function buildBreakdown(items, amountOf, keyOf) {
  const totals = new Map()

  for (const item of items) {
    const key = keyOf(item)?.trim() || 'Sans catégorie'
    totals.set(key, (totals.get(key) || 0) + amountOf(item))
  }

  const sorted = Array.from(totals.entries()).sort((a, b) => b[1] - a[1])
  const top = sorted.slice(0, 6)
  const others = sorted.slice(6)
  const othersTotal = others.reduce((sum, [, amount]) => sum + amount, 0)

  const labels = top.map(([label]) => label)
  const values = top.map(([, value]) => value)

  if (othersTotal > 0) {
    labels.push('Autres')
    values.push(othersTotal)
  }

  return { labels, values }
}

const purchaseBreakdown = computed(() =>
  buildBreakdown(filteredReportPurchases.value, (p) => Number(p.amount || 0), (p) => p.category)
)
const incomeBreakdown = computed(() =>
  buildBreakdown(filteredReportIncomes.value, (i) => Number(i.amount || 0), (i) => i.source)
)

const reportTransactions = computed(() => {
  const rows = []

  for (const purchase of filteredReportPurchases.value) {
    rows.push({
      date: purchase.date,
      type: isSimplified.value ? 'Débit' : 'Achat',
      categoryOrSource: purchase.category || '—',
      accountPair: purchase.accountPair || purchase.category || '—',
      subCategory: purchase.subCategory || '—',
      label: purchase.merchant || purchase.note || '—',
      amount: -Number(purchase.amount || 0),
    })
  }

  for (const income of filteredReportIncomes.value) {
    rows.push({
      date: income.income_date,
      type: isSimplified.value ? 'Crédit' : 'Revenu',
      categoryOrSource: income.source || income.category || '—',
      accountPair: income.accountPair || income.source || income.category || '—',
      subCategory: income.subCategory || '—',
      label: income.note || income.operationLabel || '—',
      amount: Number(income.amount || 0),
    })
  }

  return rows.sort((a, b) => new Date(b.date) - new Date(a.date))
})

function formatFilterDate(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat('fr-FR').format(date)
}

const periodLabel = computed(() => {
  if (reportPeriod.value === 'custom') {
    const start = reportCustomStart.value ? formatFilterDate(reportCustomStart.value) : '…'
    const end = reportCustomEnd.value ? formatFilterDate(reportCustomEnd.value) : '…'
    return `Du ${start} au ${end}`
  }

  const found = periodOptions.find((option) => option.value === reportPeriod.value)
  return found ? found.label : 'Tout'
})

const typeLabel = computed(() => {
  if (isSimplified.value) {
    if (reportType.value === 'achats') return 'Débit uniquement'
    if (reportType.value === 'revenus') return 'Crédit uniquement'
    return 'Débit et crédit'
  }

  if (reportType.value === 'achats') return 'Achats uniquement'
  if (reportType.value === 'revenus') return 'Revenus uniquement'
  return 'Achats et revenus'
})

const criteriaDescription = computed(() => {
  const parts = [`Période : ${periodLabel.value}`, `Type : ${typeLabel.value}`]

  if (reportCategories.value.length) {
    parts.push(`${isSimplified.value ? 'Compte(s) lié(s)' : 'Catégorie(s)'} : ${reportCategories.value.join(', ')}`)

    if (reportSubCategories.value.length) {
      const labels = reportSubCategories.value.map((key) => {
        const [categoryName, subName] = key.split('||')
        return `${categoryName} : ${subName}`
      })
      parts.push(`Sous-catégorie(s) : ${labels.join(', ')}`)
    }
  }

  return parts.join(' • ')
})

function resetFilters() {
  reportType.value = 'both'
  reportPeriod.value = 'ytd'
  reportCustomStart.value = ''
  reportCustomEnd.value = ''
  reportCategories.value = []
  reportSubCategories.value = []
}


// Les polices standard de jsPDF (Helvetica) n'ont pas les glyphes des espaces
// insécable/fine insécable utilisées par Intl pour séparer les milliers —
// elles s'affichaient comme des "/" ou cassaient l'alignement des chiffres.
// On les remplace par un espace classique uniquement pour le texte du PDF.
function formatCurrencyForPdf(value) {
  return formatCurrency(value).replace(/[  ]/g, ' ')
}

function formatTableDate(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const day = String(date.getDate()).padStart(2, '0')
  const month = String(date.getMonth() + 1).padStart(2, '0')
  return `${day}/${month}/${date.getFullYear()}`
}

// Aperçu à l'écran des graphiques : mêmes canvases que ceux capturés pour le
// PDF (toBase64Image), pour que "ce qu'on voit" soit garanti "ce qu'on obtient".
const purchaseBreakdownCanvas = ref(null)
const incomeBreakdownCanvas = ref(null)
const evolutionCanvas = ref(null)

let purchaseBreakdownChart = null
let incomeBreakdownChart = null
let evolutionChart = null

function destroyCharts() {
  purchaseBreakdownChart?.destroy()
  incomeBreakdownChart?.destroy()
  evolutionChart?.destroy()
  purchaseBreakdownChart = null
  incomeBreakdownChart = null
  evolutionChart = null
}

async function renderCharts() {
  destroyCharts()
  await nextTick()

  if (includesPurchases.value && purchaseBreakdown.value.labels.length && purchaseBreakdownCanvas.value) {
    purchaseBreakdownChart = new ChartJS(purchaseBreakdownCanvas.value.getContext('2d'), {
      type: 'doughnut',
      data: {
        labels: purchaseBreakdown.value.labels,
        datasets: [{ data: purchaseBreakdown.value.values, backgroundColor: CHART_COLORS, borderWidth: 0 }],
      },
      options: {
        responsive: false,
        animation: false,
        plugins: {
          legend: { position: 'right', labels: { color: '#3a3f33', font: { size: 11 }, boxWidth: 12 } },
          title: {
            display: true,
            text: isSimplified.value ? 'Débit par compte lié' : 'Achats par catégorie',
            color: '#22262a',
            font: { size: 13, weight: '600' },
          },
        },
      },
    })
  }

  if (includesIncomes.value && incomeBreakdown.value.labels.length && incomeBreakdownCanvas.value) {
    incomeBreakdownChart = new ChartJS(incomeBreakdownCanvas.value.getContext('2d'), {
      type: 'doughnut',
      data: {
        labels: incomeBreakdown.value.labels,
        datasets: [{ data: incomeBreakdown.value.values, backgroundColor: CHART_COLORS, borderWidth: 0 }],
      },
      options: {
        responsive: false,
        animation: false,
        plugins: {
          legend: { position: 'right', labels: { color: '#3a3f33', font: { size: 11 }, boxWidth: 12 } },
          title: {
            display: true,
            text: isSimplified.value ? 'Crédit par compte lié' : 'Revenus par source',
            color: '#22262a',
            font: { size: 13, weight: '600' },
          },
        },
      },
    })
  }

  if (monthlyEvolution.value.labels.length && evolutionCanvas.value) {
    const datasets = []
    if (includesPurchases.value) {
      datasets.push({
        label: isSimplified.value ? 'Débit' : 'Dépenses',
        data: monthlyEvolution.value.spending,
        backgroundColor: '#c99478',
      })
    }
    if (includesIncomes.value) {
      datasets.push({
        label: isSimplified.value ? 'Crédit' : 'Revenus',
        data: monthlyEvolution.value.income,
        backgroundColor: '#8fa8a0',
      })
    }

    evolutionChart = new ChartJS(evolutionCanvas.value.getContext('2d'), {
      type: 'bar',
      data: { labels: monthlyEvolution.value.labels, datasets },
      options: {
        responsive: false,
        animation: false,
        scales: {
          x: { ticks: { color: '#3a3f33' }, grid: { display: false } },
          y: { ticks: { color: '#3a3f33' }, grid: { color: '#e5e7eb' } },
        },
        plugins: {
          legend: { position: 'top', labels: { color: '#3a3f33' } },
          title: { display: true, text: 'Évolution mensuelle', color: '#22262a', font: { size: 13, weight: '600' } },
        },
      },
    })
  }
}

watch(
  [
    reportType,
    reportPeriod,
    reportCustomStart,
    reportCustomEnd,
    reportCategories,
    reportSubCategories,
    purchases,
    incomes,
  ],
  renderCharts,
  { deep: true }
)

onMounted(renderCharts)
onBeforeUnmount(destroyCharts)

async function addChartToDoc(doc, chart, x, y, maxWidth) {
  if (!chart) return y

  const dataUrl = chart.toBase64Image()
  const props = doc.getImageProperties(dataUrl)
  const width = maxWidth
  const height = width * (props.height / props.width)

  doc.addImage(dataUrl, 'PNG', x, y, width, height)

  return y + height
}

const isGenerating = ref(false)
const generationError = ref('')
const generationSuccess = ref('')

const latestReportMeta = ref(null)
const isLoadingLatest = ref(true)
const latestReportError = ref('')

async function loadLatestReport() {
  isLoadingLatest.value = true
  latestReportError.value = ''

  try {
    latestReportMeta.value = await fetchLatestReportMetadata()
  } catch (err) {
    latestReportError.value = err instanceof Error ? err.message : 'Erreur inconnue.'
  } finally {
    isLoadingLatest.value = false
  }
}

onMounted(loadLatestReport)

async function openLatestReport() {
  latestReportError.value = ''

  try {
    await viewLatestReportPdf()
  } catch (err) {
    latestReportError.value = err instanceof Error ? err.message : 'Impossible d’ouvrir le rapport.'
  }
}

function formatGeneratedAt(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat('fr-FR', { dateStyle: 'long', timeStyle: 'short' }).format(date)
}

function formatSize(bytes) {
  const size = Number(bytes || 0)
  if (size < 1024) return `${size} o`
  const kb = size / 1024
  if (kb < 1024) return `${kb.toFixed(0)} Ko`
  return `${(kb / 1024).toFixed(1)} Mo`
}

async function generateReport() {
  isGenerating.value = true
  generationError.value = ''
  generationSuccess.value = ''

  try {
    await renderCharts()
    await nextTick()

    const doc = new jsPDF({ unit: 'mm', format: 'a4' })
    const pageWidth = doc.internal.pageSize.getWidth()
    const marginX = 14

    doc.setFillColor(58, 74, 66)
    doc.rect(0, 0, pageWidth, 32, 'F')
    doc.setTextColor(255, 255, 255)
    doc.setFont('helvetica', 'bold')
    doc.setFontSize(20)
    doc.text('Rapport financier', marginX, 18)
    doc.setFont('helvetica', 'normal')
    doc.setFontSize(10)
    const generatedAtLabel = new Intl.DateTimeFormat('fr-FR', {
      dateStyle: 'long',
      timeStyle: 'short',
    }).format(new Date())
    doc.text(`Généré le ${generatedAtLabel}`, marginX, 26)

    let cursorY = 42
    doc.setTextColor(58, 63, 51)
    doc.setFont('helvetica', 'bold')
    doc.setFontSize(11)
    doc.text('Critères appliqués', marginX, cursorY)
    cursorY += 6
    doc.setFont('helvetica', 'normal')
    doc.setFontSize(9.5)
    doc.setTextColor(90, 96, 88)
    doc.text(criteriaDescription.value, marginX, cursorY)
    cursorY += 10

    const kpis = []
    if (includesPurchases.value) {
      kpis.push({ label: isSimplified.value ? 'Total débit' : 'Total dépensé', value: formatCurrencyForPdf(totalPurchasesAmount.value) })
      kpis.push({ label: isSimplified.value ? 'Nb débits' : 'Nb achats', value: String(purchaseCount.value) })
      kpis.push({ label: isSimplified.value ? 'Débit moyen' : 'Panier moyen', value: formatCurrencyForPdf(averagePurchase.value) })
    }
    if (includesIncomes.value) {
      kpis.push({ label: isSimplified.value ? 'Total crédit' : 'Total revenus', value: formatCurrencyForPdf(totalIncomesAmount.value) })
      kpis.push({ label: isSimplified.value ? 'Nb crédits' : 'Nb revenus', value: String(incomeCount.value) })
      kpis.push({ label: isSimplified.value ? 'Crédit moyen' : 'Revenu moyen', value: formatCurrencyForPdf(averageIncome.value) })
    }
    if (includesPurchases.value && includesIncomes.value) {
      kpis.push({ label: 'Solde net', value: formatCurrencyForPdf(netBalance.value) })
    }

    const cardsPerRow = 4
    const cardGap = 4
    const cardHeight = 20
    const cardWidth = (pageWidth - marginX * 2 - cardGap * (cardsPerRow - 1)) / cardsPerRow

    kpis.forEach((kpi, index) => {
      const row = Math.floor(index / cardsPerRow)
      const col = index % cardsPerRow
      const x = marginX + col * (cardWidth + cardGap)
      const y = cursorY + row * (cardHeight + cardGap)

      doc.setFillColor(240, 242, 238)
      doc.roundedRect(x, y, cardWidth, cardHeight, 2, 2, 'F')
      doc.setTextColor(120, 128, 114)
      doc.setFont('helvetica', 'normal')
      doc.setFontSize(7.5)
      doc.text(kpi.label, x + 3, y + 6)
      doc.setTextColor(40, 46, 38)
      doc.setFont('helvetica', 'bold')
      doc.setFontSize(11)
      doc.text(kpi.value, x + 3, y + 15)
    })

    const kpiRows = Math.ceil(kpis.length / cardsPerRow)
    cursorY += kpiRows * (cardHeight + cardGap) + 6

    const halfWidth = (pageWidth - marginX * 2 - 6) / 2

    if (purchaseBreakdownChart && incomeBreakdownChart) {
      const bottom1 = await addChartToDoc(doc, purchaseBreakdownChart, marginX, cursorY, halfWidth)
      const bottom2 = await addChartToDoc(
        doc,
        incomeBreakdownChart,
        marginX + halfWidth + 6,
        cursorY,
        halfWidth
      )
      cursorY = Math.max(bottom1, bottom2) + 8
    } else if (purchaseBreakdownChart) {
      cursorY = (await addChartToDoc(doc, purchaseBreakdownChart, marginX, cursorY, pageWidth - marginX * 2)) + 8
    } else if (incomeBreakdownChart) {
      cursorY = (await addChartToDoc(doc, incomeBreakdownChart, marginX, cursorY, pageWidth - marginX * 2)) + 8
    }

    if (evolutionChart) {
      cursorY = (await addChartToDoc(doc, evolutionChart, marginX, cursorY, pageWidth - marginX * 2)) + 8
    }

    doc.addPage()
    doc.setTextColor(40, 46, 38)
    doc.setFont('helvetica', 'bold')
    doc.setFontSize(13)
    doc.text('Détail des transactions', marginX, 16)

    autoTable(doc, {
      startY: 22,
      head: isSimplified.value
        ? [['Date', 'Type', 'Compte lié', 'Libellé', 'Montant']]
        : [['Date', 'Type', 'Catégorie / Source', 'Sous-catégorie', 'Libellé', 'Montant']],
      body: reportTransactions.value.map((row) =>
        isSimplified.value
          ? [formatTableDate(row.date), row.type, row.accountPair, row.label, formatCurrencyForPdf(row.amount)]
          : [
              formatTableDate(row.date),
              row.type,
              row.categoryOrSource,
              row.subCategory,
              row.label,
              formatCurrencyForPdf(row.amount),
            ]
      ),
      styles: { fontSize: 8, cellPadding: 2.4 },
      headStyles: { fillColor: [58, 74, 66], textColor: 255 },
      alternateRowStyles: { fillColor: [244, 245, 242] },
      columnStyles: { [isSimplified.value ? 4 : 5]: { halign: 'right' } },
      margin: { left: marginX, right: marginX },
    })

    const pageCount = doc.internal.getNumberOfPages()
    for (let i = 1; i <= pageCount; i += 1) {
      doc.setPage(i)
      doc.setFont('helvetica', 'normal')
      doc.setFontSize(8)
      doc.setTextColor(150, 150, 150)
      doc.text(
        `Page ${i} / ${pageCount}`,
        pageWidth - marginX,
        doc.internal.pageSize.getHeight() - 8,
        { align: 'right' }
      )
      doc.text('Finance UI', marginX, doc.internal.pageSize.getHeight() - 8)
    }

    const blob = doc.output('blob')
    const meta = await uploadGeneratedReport(blob, criteriaDescription.value)
    latestReportMeta.value = meta
    generationSuccess.value = 'Le rapport a été généré avec succès.'
  } catch (err) {
    generationError.value = err instanceof Error ? err.message : 'Échec de la génération du rapport.'
  } finally {
    isGenerating.value = false
  }
}
</script>

<template>
  <main class="dashboard-content reports-view">
    <PageHero
      eyebrow="Synthèse"
      title="Éditions"
      :description="
        isSimplified
          ? 'Génère un état PDF de ce compte : crédit/débit, répartition par compte lié, évolution mensuelle et détail des virements.'
          : 'Génère un état PDF complet de tes finances : indicateurs clés, répartitions, évolution mensuelle et détail des transactions.'
      "
    />

    <section class="panel latest-report-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Dernier état généré</p>
          <h2>Rapport disponible</h2>
        </div>
      </div>

      <p v-if="isLoadingLatest" class="muted-text">Chargement…</p>

      <template v-else-if="latestReportMeta">
        <div class="latest-report-details">
          <div>
            <p class="eyebrow">Généré le</p>
            <p class="detail-value">{{ formatGeneratedAt(latestReportMeta.generated_at) }}</p>
          </div>
          <div>
            <p class="eyebrow">Critères</p>
            <p class="detail-value">{{ latestReportMeta.criteria || '—' }}</p>
          </div>
          <div>
            <p class="eyebrow">Taille</p>
            <p class="detail-value">{{ formatSize(latestReportMeta.size_bytes) }}</p>
          </div>
        </div>

        <button class="primary-btn" type="button" @click="openLatestReport">
          Ouvrir le PDF
        </button>
      </template>

      <p v-else class="muted-text">Aucun état n’a encore été généré.</p>

      <p v-if="latestReportError" class="error-text">{{ latestReportError }}</p>
    </section>

    <section class="panel generate-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Nouveau rapport</p>
          <h2>Critères de génération</h2>
        </div>
      </div>

      <div class="reports-toolbar">
        <label class="sort-field">
          <span class="sort-label">Période</span>
          <select v-model="reportPeriod" class="sort-select">
            <option v-for="option in periodOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </label>

        <template v-if="reportPeriod === 'custom'">
          <label class="sort-field">
            <span class="sort-label">Du</span>
            <input v-model="reportCustomStart" type="date" class="sort-select" />
          </label>

          <label class="sort-field">
            <span class="sort-label">Au</span>
            <input v-model="reportCustomEnd" type="date" class="sort-select" />
          </label>
        </template>

        <label class="sort-field">
          <span class="sort-label">Type</span>
          <select v-model="reportType" class="sort-select">
            <template v-if="isSimplified">
              <option value="both">Débit et crédit</option>
              <option value="achats">Débit uniquement</option>
              <option value="revenus">Crédit uniquement</option>
            </template>
            <template v-else>
              <option value="both">Achats et revenus</option>
              <option value="achats">Achats uniquement</option>
              <option value="revenus">Revenus uniquement</option>
            </template>
          </select>
        </label>

        <MultiSelectDropdown
          v-model="reportCategories"
          :label="isSimplified ? 'Compte lié' : 'Catégorie'"
          :all-label="isSimplified ? 'Tous les comptes liés' : 'Toutes les catégories'"
          :options="categoryFilterOptions"
        />

        <MultiSelectDropdown
          v-if="reportCategories.length && !isSimplified"
          v-model="reportSubCategories"
          label="Sous-catégorie"
          all-label="Toutes les sous-catégories"
          :groups="subCategoryGroups"
        />

        <button class="ghost-btn reset-filters-btn" type="button" @click="resetFilters">
          Réinitialiser
        </button>
      </div>

      <section class="reports-stats-grid">
        <article v-if="includesPurchases" class="stat-card-mini">
          <p class="eyebrow">{{ isSimplified ? 'Total débit' : 'Total dépensé' }}</p>
          <h3>{{ formatCurrency(totalPurchasesAmount) }}</h3>
        </article>
        <article v-if="includesIncomes" class="stat-card-mini">
          <p class="eyebrow">{{ isSimplified ? 'Total crédit' : 'Total revenus' }}</p>
          <h3>{{ formatCurrency(totalIncomesAmount) }}</h3>
        </article>
        <article v-if="includesPurchases && includesIncomes" class="stat-card-mini">
          <p class="eyebrow">Solde net</p>
          <h3>{{ formatCurrency(netBalance) }}</h3>
        </article>
        <article class="stat-card-mini">
          <p class="eyebrow">Transactions</p>
          <h3>{{ purchaseCount + incomeCount }}</h3>
        </article>
      </section>

      <section class="reports-preview">
        <div class="preview-charts">
          <div v-if="includesPurchases && purchaseBreakdown.labels.length" class="preview-chart-box">
            <canvas ref="purchaseBreakdownCanvas" width="480" height="280"></canvas>
          </div>
          <div v-if="includesIncomes && incomeBreakdown.labels.length" class="preview-chart-box">
            <canvas ref="incomeBreakdownCanvas" width="480" height="280"></canvas>
          </div>
        </div>

        <div v-if="monthlyEvolution.labels.length" class="preview-chart-box preview-chart-wide">
          <canvas ref="evolutionCanvas" width="1000" height="300"></canvas>
        </div>

        <p v-if="!purchaseCount && !incomeCount" class="muted-text">
          Aucune donnée ne correspond à ces critères pour le moment.
        </p>
      </section>

      <div class="generate-actions">
        <button class="primary-btn" type="button" :disabled="isGenerating" @click="generateReport">
          {{ isGenerating ? 'Génération en cours…' : 'Générer le PDF' }}
        </button>

        <p v-if="generationSuccess" class="success-text">{{ generationSuccess }}</p>
        <p v-if="generationError" class="error-text">{{ generationError }}</p>
      </div>
    </section>
  </main>
</template>

<style scoped>
.reports-view {
  display: grid;
  gap: 0.9rem;
}

.latest-report-card,
.generate-card {
  padding: 0.95rem 1rem;
}

.panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.7rem;
}

.panel-header h2 {
  margin: 0.2rem 0 0;
  color: var(--text, #eef1f3);
  font-size: 1.05rem;
}

.eyebrow {
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.72rem;
}

.muted-text {
  color: var(--text-soft, #b3bbc4);
  font-size: 0.88rem;
}

.error-text {
  color: var(--negative-text, #e1b4b4);
  font-size: 0.85rem;
  margin-top: 0.5rem;
}

.success-text {
  color: var(--positive-text, #bfe0c9);
  font-size: 0.85rem;
  margin-top: 0.5rem;
}

.latest-report-details {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.9rem;
  margin-bottom: 0.9rem;
}

.detail-value {
  margin: 0.2rem 0 0;
  color: var(--text, #eef1f3);
  font-size: 0.92rem;
}

.reports-toolbar {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  flex-wrap: wrap;
  margin-bottom: 0.9rem;
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

.sort-select {
  height: 32px;
  min-width: 168px;
  border-radius: 10px;
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--text, #eef1f3);
  padding: 0 0.7rem;
  font-size: 0.8rem;
  outline: none;
}

.sort-select:focus {
  border-color: rgba(143, 168, 160, 0.34);
  box-shadow: 0 0 0 3px rgba(143, 168, 160, 0.12);
}

.reset-filters-btn {
  height: 32px;
  padding: 0 0.7rem;
  border-radius: 10px;
  font-size: 0.8rem;
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  background: rgba(var(--tint-rgb), 0.03);
  color: var(--text-soft, #b3bbc4);
  cursor: pointer;
}

.reset-filters-btn:hover {
  background: rgba(var(--tint-rgb), 0.06);
}

.reports-stats-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.65rem;
  margin-bottom: 1rem;
}

.stat-card-mini {
  padding: 0.65rem 0.75rem;
  border-radius: 12px;
  background: rgba(var(--tint-rgb), 0.035);
  border: 1px solid rgba(var(--tint-rgb), 0.06);
}

.stat-card-mini h3 {
  margin: 0.2rem 0 0;
  color: var(--text, #eef1f3);
  font-size: 1.05rem;
}

.reports-preview {
  margin-bottom: 1rem;
}

.preview-charts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}

.preview-chart-box {
  background: #f6f7f5;
  border-radius: 14px;
  padding: 0.5rem;
  overflow: auto;
}

.preview-chart-box canvas {
  max-width: 100%;
  height: auto;
}

.preview-chart-wide {
  width: 100%;
}

.generate-actions {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.3rem;
}

@media (max-width: 900px) {
  .latest-report-details {
    grid-template-columns: 1fr;
  }

  .reports-stats-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .reports-toolbar {
    width: 100%;
    align-items: stretch;
  }

  .sort-select {
    width: 100%;
  }

  .reports-stats-grid {
    grid-template-columns: 1fr;
  }
}
</style>
