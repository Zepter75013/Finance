import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import {
  fetchPurchases,
  fetchCategories,
  createPurchase,
  updatePurchase,
  deletePurchase,
  createCategory as createCategoryApi,
  updateCategory as updateCategoryApi,
  deleteCategory as deleteCategoryApi,
} from '../services/purchases'
import {
  fetchIncomes,
  createIncome as createIncomeApi,
  updateIncome as updateIncomeApi,
  deleteIncome as deleteIncomeApi,
} from '../services/incomes'
import {
  fetchSubCategories,
  createSubCategory as createSubCategoryApi,
  updateSubCategory as updateSubCategoryApi,
  deleteSubCategory as deleteSubCategoryApi,
} from '../services/subcategories'
import {
  fetchAccounts,
  createAccount as createAccountApi,
  updateAccount as updateAccountApi,
  deleteAccount as deleteAccountApi,
  updateOpeningBalance as updateOpeningBalanceApi,
  clearOpeningBalance as clearOpeningBalanceApi,
  updateHasStatements as updateHasStatementsApi,
} from '../services/accounts'
import { fetchStatements } from '../services/statements'
import { fetchBudgets, upsertBudget, deleteBudget } from '../services/budgets'
import {
  fetchTransfers,
  createTransfer as createTransferApi,
  updateTransfer as updateTransferApi,
  deleteTransfer as deleteTransferApi,
} from '../services/transfers'
import { findLatestLockedStatement, computeRealBalance, signTransferLeg } from '../utils/realBalance'

export const usePurchasesStore = defineStore('purchases', () => {
  const purchases = ref([])
  const incomes = ref([])
  const categoriesList = ref([])
  const subCategoriesList = ref([])
  const accountsList = ref([])
  const statements = ref([])
  const transfers = ref([])

  // Compte "actif" : celui qui pré-remplit le champ compte lorsqu'on saisit
  // un nouvel achat/revenu ou qu'on importe un fichier, pour ne pas avoir à
  // le resélectionner à chaque fois. Partagé par tous les écrans (Achats,
  // Revenus, Import, Pointage) — le changer à un endroit le change partout.
  const ACTIVE_ACCOUNT_STORAGE_KEY = 'activeAccountId'

  function loadStoredActiveAccountId() {
    const raw = Number(localStorage.getItem(ACTIVE_ACCOUNT_STORAGE_KEY))
    return Number.isFinite(raw) && raw > 0 ? raw : null
  }

  const activeAccountId = ref(loadStoredActiveAccountId())

  function setActiveAccountId(id) {
    activeAccountId.value = id || null

    if (activeAccountId.value) {
      localStorage.setItem(ACTIVE_ACCOUNT_STORAGE_KEY, String(activeAccountId.value))
    } else {
      localStorage.removeItem(ACTIVE_ACCOUNT_STORAGE_KEY)
    }
  }

  const activeCategory = ref('Toutes')
  const selectedPurchaseId = ref(null)

  // Peuplé depuis le serveur par loadBudgetOverrides() (voir plus bas) — ne
  // vit plus dans localStorage, pour que le planificateur d'email backend
  // puisse calculer les mêmes budgets que cet écran.
  const categoryBudgetOverrides = ref({})

  const isLoading = ref(false)
  const error = ref('')

  const categoriesById = computed(() => {
    return categoriesList.value.reduce((acc, category) => {
      acc[category.id] = category
      return acc
    }, {})
  })

  function getCategoryName(categoryId) {
    return categoriesById.value[categoryId]?.name || 'Catégorie inconnue'
  }

  const accountsById = computed(() => {
    return accountsList.value.reduce((acc, account) => {
      acc[account.id] = account
      return acc
    }, {})
  })

  function getAccountName(accountId) {
    return accountsById.value[accountId]?.name || 'Compte inconnu'
  }

  function refreshPurchaseCategoryLabels() {
    purchases.value = purchases.value.map((purchase) => ({
      ...purchase,
      category: getCategoryName(purchase.categoryId),
    }))
  }

  function refreshPurchaseAccountLabels() {
    purchases.value = purchases.value.map((purchase) => ({
      ...purchase,
      account: getAccountName(purchase.accountId),
    }))
  }

  function refreshIncomeAccountLabels() {
    incomes.value = incomes.value.map((income) => ({
      ...income,
      account: getAccountName(income.accountId),
    }))
  }

  function refreshAccountLabels() {
    refreshPurchaseAccountLabels()
    refreshIncomeAccountLabels()
  }

  function mapPurchaseFromApi(purchase) {
    return {
      id: purchase.id,
      merchant: purchase.merchant,
      paymentMethod: purchase.payment_method,
      category: getCategoryName(purchase.category_id),
      categoryId: purchase.category_id,
      account: getAccountName(purchase.account_id),
      accountId: purchase.account_id,
      amount: Number(purchase.amount || 0),
      date: purchase.purchase_date?.slice(0, 10) || '',
      note: purchase.note || '',
      reference: purchase.reference || '',
      operationLabel: purchase.operation_label || '',
      additionalInfo: purchase.additional_info || '',
      subCategory: purchase.sub_category || '',
      operationDate: purchase.operation_date?.slice(0, 10) || '',
      valueDate: purchase.value_date?.slice(0, 10) || '',
      isReconciled: Boolean(purchase.is_reconciled),
      statementReference: purchase.statement_reference || '',
      createdAt: purchase.created_at,
      updatedAt: purchase.updated_at,
    }
  }

  function mapCategoryFromApi(category) {
    return {
      id: category.id,
      accountId: category.account_id,
      name: category.name,
      type: category.type === 'revenu' ? 'revenu' : 'achat',
      description: category.description || '',
      createdAt: category.created_at,
      updatedAt: category.updated_at,
    }
  }

  function mapSubCategoryFromApi(subCategory) {
    return {
      id: subCategory.id,
      categoryId: subCategory.category_id,
      name: subCategory.name,
      createdAt: subCategory.created_at,
      updatedAt: subCategory.updated_at,
    }
  }

  function mapAccountFromApi(account) {
    return {
      id: account.id,
      name: account.name,
      purchaseCount: Number(account.purchase_count || 0),
      incomeCount: Number(account.income_count || 0),
      totalExpense: Number(account.total_expense || 0),
      totalIncome: Number(account.total_income || 0),
      categoryCount: Number(account.category_count || 0),
      transferCount: Number(account.transfer_count || 0),
      openingBalanceAmount: account.opening_balance_amount ?? null,
      openingBalanceDate: account.opening_balance_date ?? null,
      hasStatements: account.has_statements !== false,
    }
  }

  function mapIncomeFromApi(income) {
    return {
      id: income.id,
      account: getAccountName(income.account_id),
      accountId: income.account_id,
      source: income.source,
      amount: Number(income.amount || 0),
      income_date: income.income_date?.slice(0, 10) || '',
      note: income.note || '',
      reference: income.reference || '',
      operationLabel: income.operation_label || '',
      additionalInfo: income.additional_info || '',
      operationType: income.operation_type || '',
      category: income.category || '',
      subCategory: income.sub_category || '',
      operationDate: income.operation_date?.slice(0, 10) || '',
      valueDate: income.value_date?.slice(0, 10) || '',
      isReconciled: Boolean(income.is_reconciled),
      statementReference: income.statement_reference || '',
      createdAt: income.created_at,
      updatedAt: income.updated_at,
    }
  }

  function mapTransferFromApi(t) {
    return {
      id: t.id,
      fromAccountId: t.from_account_id,
      toAccountId: t.to_account_id,
      fromAccountName: getAccountName(t.from_account_id),
      toAccountName: getAccountName(t.to_account_id),
      amount: Number(t.amount || 0),
      date: t.transfer_date?.slice(0, 10) || '',
      note: t.note || '',
      fromIsReconciled: Boolean(t.from_is_reconciled),
      fromStatementReference: t.from_statement_reference || '',
      toIsReconciled: Boolean(t.to_is_reconciled),
      toStatementReference: t.to_statement_reference || '',
      originType: t.origin_type || null,
      originPayload: t.origin_payload || null,
    }
  }

  function buildTransferPayload(t) {
    return {
      from_account_id: Number(t.fromAccountId || 0),
      to_account_id: Number(t.toAccountId || 0),
      origin_type: t.originType || null,
      origin_payload: t.originPayload || null,
      amount: Number(t.amount || 0),
      transfer_date: t.date || new Date().toISOString().slice(0, 10),
      note: t.note?.trim() || '',
      from_is_reconciled: Boolean(t.fromIsReconciled),
      from_statement_reference: t.fromStatementReference?.trim() || '',
      to_is_reconciled: Boolean(t.toIsReconciled),
      to_statement_reference: t.toStatementReference?.trim() || '',
    }
  }

  function buildPurchasePayload(purchase) {
    return {
      merchant: purchase.merchant?.trim() || '',
      payment_method: purchase.paymentMethod?.trim() || '',
      category_id: Number(purchase.categoryId || 0),
      account_id: Number(purchase.accountId || 0),
      amount: Number(purchase.amount || 0),
      purchase_date: purchase.date || new Date().toISOString().slice(0, 10),
      note: purchase.note?.trim() || '',
      reference: purchase.reference?.trim() || '',
      operation_label: purchase.operationLabel?.trim() || '',
      additional_info: purchase.additionalInfo?.trim() || '',
      sub_category: purchase.subCategory?.trim() || '',
      operation_date: purchase.operationDate || '',
      value_date: purchase.valueDate || '',
      is_reconciled: Boolean(purchase.isReconciled),
      statement_reference: purchase.statementReference?.trim() || '',
    }
  }

  function buildCategoryPayload(category) {
    return {
      name: category.name?.trim() || '',
      type: category.type === 'revenu' ? 'revenu' : 'achat',
      description: category.description?.trim() || '',
      account_id: Number(category.accountId || activeAccountId.value || 0),
    }
  }

  function buildIncomePayload(income) {
    return {
      account_id: Number(income.accountId || 0),
      source: income.source?.trim() || '',
      amount: Number(income.amount || 0),
      income_date: income.income_date || new Date().toISOString().slice(0, 10),
      note: income.note?.trim() || '',
      reference: income.reference?.trim() || '',
      operation_label: income.operationLabel?.trim() || '',
      additional_info: income.additionalInfo?.trim() || '',
      operation_type: income.operationType?.trim() || '',
      category: income.category?.trim() || '',
      sub_category: income.subCategory?.trim() || '',
      operation_date: income.operationDate || '',
      value_date: income.valueDate || '',
      is_reconciled: Boolean(income.isReconciled),
      statement_reference: income.statementReference?.trim() || '',
    }
  }

  function validatePurchasePayload(payload) {
    if (!payload.merchant) {
      throw new Error('Le commerçant est obligatoire.')
    }

    if (!payload.payment_method) {
      throw new Error('Le mode de paiement est obligatoire.')
    }

    if (!payload.category_id) {
      throw new Error('La catégorie est obligatoire.')
    }

    if (!payload.account_id) {
      throw new Error('Le compte est obligatoire.')
    }

    if (payload.amount <= 0) {
      throw new Error('Le montant doit être supérieur à 0.')
    }

    if (!payload.purchase_date) {
      throw new Error('La date d’achat est obligatoire.')
    }
  }

  function validateCategoryPayload(payload, editingCategoryId = null) {
    if (!payload.name) {
      throw new Error('Le nom de la catégorie est obligatoire.')
    }

    if (!payload.account_id) {
      throw new Error('Le compte est obligatoire.')
    }

    const normalizedName = payload.name.trim().toLowerCase()

    const alreadyExists = categoriesList.value.some((category) => {
      return (
        Number(category.id) !== Number(editingCategoryId) &&
        category.name?.trim().toLowerCase() === normalizedName
      )
    })

    if (alreadyExists) {
      throw new Error('Une catégorie avec ce nom existe déjà.')
    }
  }

  function validateIncomePayload(payload) {
    if (!payload.source) {
      throw new Error('La source du revenu est obligatoire.')
    }

    if (!payload.account_id) {
      throw new Error('Le compte est obligatoire.')
    }

    if (payload.amount <= 0) {
      throw new Error('Le montant doit être supérieur à 0.')
    }

    if (!payload.income_date) {
      throw new Error('La date du revenu est obligatoire.')
    }
  }

  function syncSelectionAfterMutation(preferredId = null) {
    if (!filteredPurchases.value.length) {
      activeCategory.value = 'Toutes'
    }

    if (preferredId) {
      const match = filteredPurchases.value.find(
        (purchase) => purchase.id === preferredId
      )

      if (match) {
        selectedPurchaseId.value = preferredId
        return
      }
    }

    selectedPurchaseId.value = filteredPurchases.value[0]?.id ?? null
  }

  const categories = computed(() => {
    return ['Toutes', ...categoriesList.value.map((category) => category.name)]
  })

  const filteredPurchases = computed(() => {
    if (activeCategory.value === 'Toutes') {
      return purchases.value
    }

    return purchases.value.filter(
      (purchase) => purchase.category === activeCategory.value
    )
  })

  const selectedPurchase = computed(() => {
    return (
      filteredPurchases.value.find(
        (purchase) => purchase.id === selectedPurchaseId.value
      ) || filteredPurchases.value[0] || null
    )
  })

  const totalMonthAmount = computed(() => {
    return purchases.value.reduce((sum, purchase) => {
      return sum + Number(purchase.amount || 0)
    }, 0)
  })

  const totalIncomeAmount = computed(() => {
    return incomes.value.reduce((sum, income) => {
      return sum + Number(income.amount || 0)
    }, 0)
  })

  const purchaseCount = computed(() => purchases.value.length)

  const incomeCount = computed(() => incomes.value.length)

  const averageBasket = computed(() => {
    if (!purchases.value.length) return 0
    return totalMonthAmount.value / purchases.value.length
  })

  const averageIncome = computed(() => {
    if (!incomes.value.length) return 0
    return totalIncomeAmount.value / incomes.value.length
  })

  const currentMonthKey = new Date().toISOString().slice(0, 7)

  // Nombre de mois déjà terminés (achats OU revenus, toutes catégories) — sert
  // de dénominateur à la moyenne mensuelle réelle d'une catégorie. Le mois en
  // cours est exclu car incomplet : sinon, dès qu'une seule dépense existe ce
  // mois-ci (dans n'importe quelle catégorie), il gonflerait le dénominateur
  // et diluerait la moyenne des catégories qui n'y ont pas encore de dépense.
  const pastTrackedMonthCount = computed(() => {
    const monthKeys = new Set()

    for (const purchase of purchases.value) {
      if (purchase.date && purchase.date.slice(0, 7) < currentMonthKey) {
        monthKeys.add(purchase.date.slice(0, 7))
      }
    }
    for (const income of incomes.value) {
      if (income.income_date && income.income_date.slice(0, 7) < currentMonthKey) {
        monthKeys.add(income.income_date.slice(0, 7))
      }
    }

    return monthKeys.size
  })

  // Moyenne mensuelle réelle d'une catégorie sur les mois déjà terminés — sert
  // de budget par défaut pour le mois en cours tant qu'il n'est pas ajusté.
  function suggestedCategoryBudget(category) {
    const monthCount = pastTrackedMonthCount.value
    if (!monthCount) return 0

    const total =
      category.type === 'revenu'
        ? incomes.value
            .filter(
              (income) => income.category === category.name && income.income_date?.slice(0, 7) < currentMonthKey
            )
            .reduce((sum, income) => sum + Number(income.amount || 0), 0)
        : purchases.value
            .filter(
              (purchase) =>
                Number(purchase.categoryId) === Number(category.id) && purchase.date?.slice(0, 7) < currentMonthKey
            )
            .reduce((sum, purchase) => sum + Number(purchase.amount || 0), 0)

    return total / monthCount
  }

  // Budget effectif d'une catégorie pour le mois en cours : l'ajustement
  // ponctuel s'il existe, sinon la moyenne réelle.
  function currentMonthCategoryBudget(category) {
    return isCategoryBudgetOverridden(category.id, currentMonthKey)
      ? getCategoryBudgetForMonth(category.id, currentMonthKey)
      : suggestedCategoryBudget(category)
  }

  const currentMonthExpenseBudget = computed(() =>
    categoriesList.value
      .filter((category) => category.type !== 'revenu')
      .reduce((sum, category) => sum + currentMonthCategoryBudget(category), 0)
  )

  const currentMonthExpenseSpent = computed(() =>
    purchases.value
      .filter((purchase) => purchase.date?.slice(0, 7) === currentMonthKey)
      .reduce((sum, purchase) => sum + Number(purchase.amount || 0), 0)
  )

  const currentMonthBudgetRemaining = computed(
    () => currentMonthExpenseBudget.value - currentMonthExpenseSpent.value
  )

  // Le dernier relevé VERROUILLÉ (et non simplement le plus récent, qui
  // pourrait être un brouillon en cours de saisie) sert d'ancrage fiable —
  // c'est un solde bancaire confirmé. Logique partagée avec la vue
  // consolidée multi-comptes (utils/realBalance.js), qui recalcule la même
  // chose pour chaque compte.
  const latestLockedStatement = computed(() => findLatestLockedStatement(statements.value))

  const activeAccount = computed(() =>
    accountsList.value.find((account) => Number(account.id) === Number(activeAccountId.value)) || null
  )

  // Solde réel = solde de fin du dernier relevé verrouillé + tous les achats/
  // revenus pas encore rattachés à un relevé (statementReference vide) — ce
  // sont les mouvements que la banque a déjà comptabilisés mais que le
  // pointage personnel n'a pas encore rapprochés d'un relevé précis. Sans
  // relevé verrouillé, repli sur un solde initial déclaré si le compte en a
  // un (compte type Livret), sinon sur les seuls mouvements non rapprochés —
  // voir le commentaire de computeRealBalance pour le détail des trois cas.
  // Chaque virement est signé du point de vue du compte actif (débit si ce
  // compte en est la source, crédit s'il en est la destination) — un seul
  // calcul de "isOutgoing", partagé avec ConsolidatedView.vue et
  // ReconciliationView.vue via signTransferLeg.
  const transferLegsForActiveAccount = computed(() =>
    transfers.value.map((t) => signTransferLeg(t, activeAccountId.value))
  )

  const realBalanceInfo = computed(() =>
    computeRealBalance({
      statements: statements.value,
      purchases: purchases.value.map((p) => ({ amount: p.amount, statement_reference: p.statementReference, date: p.date })),
      incomes: incomes.value.map((i) => ({ amount: i.amount, statement_reference: i.statementReference, date: i.income_date })),
      transfers: transferLegsForActiveAccount.value,
      openingBalance: activeAccount.value
        ? { amount: activeAccount.value.openingBalanceAmount, date: activeAccount.value.openingBalanceDate }
        : null,
    })
  )
  const realCurrentBalance = computed(() => realBalanceInfo.value.balance)
  const realBalanceSource = computed(() => realBalanceInfo.value.source)
  // true = solde non ancré à un relevé bancaire (pas "approximatif" : le
  // calcul reste exact, juste pas confirmé par la banque).
  const isRealBalanceEstimate = computed(() => realBalanceSource.value !== 'statement')

  async function loadCategories() {
    if (!activeAccountId.value) {
      categoriesList.value = []
      return
    }

    const data = await fetchCategories(activeAccountId.value)
    categoriesList.value = data.map(mapCategoryFromApi)
    refreshPurchaseCategoryLabels()
  }

  async function loadSubCategories() {
    const data = await fetchSubCategories()
    subCategoriesList.value = data.map(mapSubCategoryFromApi)
  }

  async function loadAccounts() {
    const data = await fetchAccounts()
    accountsList.value = data.map(mapAccountFromApi)
    refreshAccountLabels()

    // Le compte actif mémorisé peut avoir été supprimé entre deux visites —
    // dans ce cas (ou s'il n'y en a jamais eu) on retombe sur le premier
    // compte disponible plutôt que de laisser un id invalide.
    const isStillValid = accountsList.value.some(
      (account) => Number(account.id) === Number(activeAccountId.value)
    )

    if (!activeAccountId.value || !isStillValid) {
      setActiveAccountId(accountsList.value[0]?.id ?? null)
    }
  }

  async function loadIncomes() {
    if (!activeAccountId.value) {
      incomes.value = []
      return
    }

    const data = await fetchIncomes(activeAccountId.value)
    incomes.value = data.map(mapIncomeFromApi)
  }

  async function loadPurchasesForActiveAccount() {
    if (!activeAccountId.value) {
      purchases.value = []
      return
    }

    const data = await fetchPurchases(activeAccountId.value)
    purchases.value = data.map(mapPurchaseFromApi)
  }

  // Chargé pour calculer le solde réel (dernier relevé verrouillé + achats/
  // revenus pas encore rattachés à un relevé) — pas d'échec bloquant : le
  // dashboard doit rester utilisable même si les relevés ne se chargent pas.
  async function loadStatements() {
    if (!activeAccountId.value) {
      statements.value = []
      return
    }

    try {
      statements.value = await fetchStatements(activeAccountId.value)
    } catch {
      statements.value = []
    }
  }

  // Charge les budgets ajustés du mois courant depuis le serveur — pas
  // d'échec bloquant, même raison que loadStatements ci-dessus. L'API
  // renvoie une map category_id -> montant (clés JSON en chaîne) ;
  // reconstruite ici sous la forme `${monthKey}:${categoryId}` déjà utilisée
  // partout ailleurs dans ce store.
  async function loadBudgetOverrides() {
    if (!activeAccountId.value) {
      categoryBudgetOverrides.value = {}
      return
    }

    try {
      const data = await fetchBudgets(activeAccountId.value, currentMonthKey)
      const next = {}

      for (const [categoryId, amount] of Object.entries(data || {})) {
        next[`${currentMonthKey}:${categoryId}`] = Number(amount)
      }

      categoryBudgetOverrides.value = next
    } catch {
      categoryBudgetOverrides.value = {}
    }
  }

  // Virements touchant le compte actif (source OU destination) — pas
  // d'échec bloquant, même raison que loadStatements ci-dessus.
  async function loadTransfers() {
    if (!activeAccountId.value) {
      transfers.value = []
      return
    }

    try {
      const data = await fetchTransfers(activeAccountId.value)
      transfers.value = data.map(mapTransferFromApi)
    } catch {
      transfers.value = []
    }
  }

  // Chaque compte a ses propres achats, revenus et catégories — bascule
  // complète des trois à chaque changement de compte actif (bootstrap initial
  // ET changement ultérieur via le sélecteur), pour que le dashboard, le
  // budget, le pointage et les éditions ne montrent jamais que le compte
  // actuellement sélectionné.
  async function reloadAccountScopedData() {
    isLoading.value = true
    error.value = ''

    try {
      await loadCategories()
      await loadIncomes()
      await loadPurchasesForActiveAccount()
      await loadStatements()
      await loadBudgetOverrides()
      await loadTransfers()
      activeCategory.value = 'Toutes'
      selectedPurchaseId.value = purchases.value[0]?.id ?? null
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Erreur inconnue.'
      purchases.value = []
      incomes.value = []
      categoriesList.value = []
      selectedPurchaseId.value = null
    } finally {
      isLoading.value = false
    }
  }

  // Le changement de compte actif ne doit déclencher un rechargement que s'il
  // survient APRÈS le chargement initial (ex: l'utilisateur bascule via le
  // sélecteur) — pas pendant le bootstrap, où loadAccounts() peut lui-même
  // fixer le compte actif une première fois alors que reloadAccountScopedData
  // va de toute façon être appelé juste après.
  let suppressActiveAccountWatch = true

  watch(activeAccountId, async (id, previousId) => {
    if (suppressActiveAccountWatch) return
    if (!id || id === previousId) return

    await reloadAccountScopedData()
  })

  async function loadPurchases() {
    isLoading.value = true
    error.value = ''

    try {
      await loadAccounts()
      await loadSubCategories()
      await reloadAccountScopedData()
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Erreur inconnue.'
      purchases.value = []
      incomes.value = []
      categoriesList.value = []
      subCategoriesList.value = []
      accountsList.value = []
      selectedPurchaseId.value = null
    } finally {
      isLoading.value = false
      suppressActiveAccountWatch = false
    }
  }

  async function createSubCategory(categoryId, name) {
    const created = await createSubCategoryApi({ category_id: Number(categoryId), name: name?.trim() || '' })
    const mapped = mapSubCategoryFromApi(created)

    if (!subCategoriesList.value.some((sc) => sc.id === mapped.id)) {
      subCategoriesList.value = [...subCategoriesList.value, mapped]
    }

    return mapped
  }

  async function editSubCategory(id, name) {
    const current = subCategoriesList.value.find((sc) => sc.id === id)

    const updated = await updateSubCategoryApi(id, name?.trim() || '')
    const mapped = mapSubCategoryFromApi(updated)

    subCategoriesList.value = subCategoriesList.value.map((sc) => (sc.id === id ? mapped : sc))

    // Le backend renomme déjà la sous-catégorie sur les achats/revenus en
    // base — on répercute le même changement en mémoire pour que la liste
    // affichée reflète immédiatement le nouveau libellé sans recharger.
    if (current && current.name !== mapped.name) {
      const categoryName = getCategoryName(mapped.categoryId)

      purchases.value = purchases.value.map((purchase) => {
        if (Number(purchase.categoryId) !== Number(mapped.categoryId)) return purchase
        if (purchase.subCategory !== current.name) return purchase
        return { ...purchase, subCategory: mapped.name }
      })

      incomes.value = incomes.value.map((income) => {
        if ((income.category?.trim() || '') !== categoryName) return income
        if (income.subCategory !== current.name) return income
        return { ...income, subCategory: mapped.name }
      })
    }

    return mapped
  }

  async function removeSubCategory(id) {
    await deleteSubCategoryApi(id)
    subCategoriesList.value = subCategoriesList.value.filter((sc) => sc.id !== id)
  }

  async function submitPurchase(purchase) {
    error.value = ''

    const payload = buildPurchasePayload(purchase)
    validatePurchasePayload(payload)

    const created = await createPurchase(payload)
    const mappedPurchase = mapPurchaseFromApi(created)

    purchases.value.unshift(mappedPurchase)
    activeCategory.value = 'Toutes'
    selectedPurchaseId.value = mappedPurchase.id

    return mappedPurchase
  }

  async function editPurchase(id, purchase) {
    error.value = ''

    const payload = buildPurchasePayload(purchase)
    validatePurchasePayload(payload)

    const updated = await updatePurchase(id, payload)
    const mappedPurchase = mapPurchaseFromApi(updated)

    purchases.value = purchases.value.map((currentPurchase) => {
      return currentPurchase.id === id ? mappedPurchase : currentPurchase
    })

    syncSelectionAfterMutation(mappedPurchase.id)

    return mappedPurchase
  }

  async function removePurchase(id) {
    error.value = ''

    await deletePurchase(id)

    purchases.value = purchases.value.filter((purchase) => purchase.id !== id)
    syncSelectionAfterMutation()
  }

  async function createCategory(category) {
    error.value = ''

    const payload = buildCategoryPayload(category)
    validateCategoryPayload(payload)

    const created = await createCategoryApi(payload)
    const mappedCategory = mapCategoryFromApi(created)

    categoriesList.value.push(mappedCategory)
    refreshPurchaseCategoryLabels()

    return mappedCategory
  }

  async function editCategory(id, category) {
    error.value = ''

    const payload = buildCategoryPayload(category)
    validateCategoryPayload(payload, id)

    const updated = await updateCategoryApi(id, payload)
    const mappedCategory = mapCategoryFromApi(updated)

    const previousCategory = categoriesList.value.find(
      (categoryItem) => Number(categoryItem.id) === Number(id)
    )

    categoriesList.value = categoriesList.value.map((currentCategory) => {
      return Number(currentCategory.id) === Number(id)
        ? mappedCategory
        : currentCategory
    })

    refreshPurchaseCategoryLabels()

    if (
      previousCategory &&
      activeCategory.value !== 'Toutes' &&
      activeCategory.value === previousCategory.name
    ) {
      activeCategory.value = mappedCategory.name
    }

    return mappedCategory
  }

  async function removeCategory(id) {
    error.value = ''

    const linkedPurchases = purchases.value.filter(
      (purchase) => Number(purchase.categoryId) === Number(id)
    )

    if (linkedPurchases.length > 0) {
      throw new Error(
        'Impossible de supprimer une catégorie déjà utilisée par des achats.'
      )
    }

    const categoryToDelete = categoriesList.value.find(
      (category) => Number(category.id) === Number(id)
    )

    await deleteCategoryApi(id)

    categoriesList.value = categoriesList.value.filter(
      (category) => Number(category.id) !== Number(id)
    )

    if (activeCategory.value === categoryToDelete?.name) {
      activeCategory.value = 'Toutes'
      selectedPurchaseId.value = filteredPurchases.value[0]?.id ?? null
    }

    refreshPurchaseCategoryLabels()
  }

  function validateAccountPayload(payload, editingAccountId = null) {
    if (!payload.name) {
      throw new Error('Le nom du compte est obligatoire.')
    }

    const normalizedName = payload.name.trim().toLowerCase()

    const alreadyExists = accountsList.value.some((account) => {
      return (
        Number(account.id) !== Number(editingAccountId) &&
        account.name?.trim().toLowerCase() === normalizedName
      )
    })

    if (alreadyExists) {
      throw new Error('Un compte avec ce nom existe déjà.')
    }
  }

  async function createAccount(name) {
    error.value = ''

    const payload = { name: name?.trim() || '' }
    validateAccountPayload(payload)

    const created = await createAccountApi(payload)
    const mappedAccount = mapAccountFromApi(created)

    accountsList.value.push(mappedAccount)

    return mappedAccount
  }

  async function editAccount(id, name) {
    error.value = ''

    const payload = { name: name?.trim() || '' }
    validateAccountPayload(payload, id)

    const updated = await updateAccountApi(id, payload)
    const mappedAccount = mapAccountFromApi(updated)

    accountsList.value = accountsList.value.map((currentAccount) => {
      return Number(currentAccount.id) === Number(id) ? mappedAccount : currentAccount
    })

    refreshAccountLabels()

    return mappedAccount
  }

  async function setAccountOpeningBalance(id, amount, date) {
    error.value = ''

    const updated = await updateOpeningBalanceApi(id, amount, date)
    const mappedAccount = mapAccountFromApi(updated)

    accountsList.value = accountsList.value.map((currentAccount) => {
      return Number(currentAccount.id) === Number(id) ? mappedAccount : currentAccount
    })

    return mappedAccount
  }

  async function clearAccountOpeningBalance(id) {
    error.value = ''

    const updated = await clearOpeningBalanceApi(id)
    const mappedAccount = mapAccountFromApi(updated)

    accountsList.value = accountsList.value.map((currentAccount) => {
      return Number(currentAccount.id) === Number(id) ? mappedAccount : currentAccount
    })

    return mappedAccount
  }

  async function setAccountHasStatements(id, hasStatements) {
    error.value = ''

    const updated = await updateHasStatementsApi(id, hasStatements)
    const mappedAccount = mapAccountFromApi(updated)

    accountsList.value = accountsList.value.map((currentAccount) => {
      return Number(currentAccount.id) === Number(id) ? mappedAccount : currentAccount
    })

    return mappedAccount
  }

  async function createTransfer(transfer) {
    error.value = ''

    const payload = buildTransferPayload(transfer)
    const created = await createTransferApi(payload)
    const mapped = mapTransferFromApi(created)

    if (
      Number(mapped.fromAccountId) === Number(activeAccountId.value) ||
      Number(mapped.toAccountId) === Number(activeAccountId.value)
    ) {
      transfers.value.unshift(mapped)
    }

    return mapped
  }

  async function editTransfer(id, transfer) {
    error.value = ''

    const payload = buildTransferPayload(transfer)
    const updated = await updateTransferApi(id, payload)
    const mapped = mapTransferFromApi(updated)

    transfers.value = transfers.value.map((current) => (current.id === id ? mapped : current))

    return mapped
  }

  async function removeTransfer(id) {
    error.value = ''

    await deleteTransferApi(id)

    transfers.value = transfers.value.filter((t) => t.id !== id)
  }

  // Annule un virement créé par conversion d'une ligne (voir la case
  // « virement » de PurchaseFormModal/IncomeFormModal, ou l'import CSV) :
  // recrée l'achat/revenu d'origine à partir de sa copie (originPayload) puis
  // supprime le virement — retour exact à l'état initial. Un virement sans
  // copie d'origine (créé directement, ex: le bouton du Dashboard LDDS) ne
  // peut pas être annulé automatiquement : rien à restaurer.
  async function undoTransfer(transfer) {
    error.value = ''

    if (!transfer.originType || !transfer.originPayload) {
      throw new Error('Ce virement n’a pas de ligne d’origine — il ne peut pas être annulé automatiquement.')
    }

    let originalRow
    try {
      originalRow = JSON.parse(transfer.originPayload)
    } catch {
      throw new Error('Impossible de lire les données d’origine de ce virement.')
    }

    if (transfer.originType === 'achat') {
      const payload = buildPurchasePayload(originalRow)
      validatePurchasePayload(payload)
      await createPurchase(payload)
    } else {
      const payload = buildIncomePayload(originalRow)
      validateIncomePayload(payload)
      await createIncomeApi(payload)
    }

    await removeTransfer(transfer.id)
    // Recharge tout plutôt que de patcher purchases.value/incomes.value à la
    // main : la ligne restaurée peut appartenir à l'autre compte du virement
    // (annulation déclenchée depuis le côté crédité), pas forcément celui
    // actuellement actif.
    await reloadAccountScopedData()
  }

  async function removeAccount(id) {
    error.value = ''

    // Les achats/revenus en mémoire ne couvrent plus que le compte actif (les
    // autres comptes ne sont plus chargés du tout) — impossible de pré-vérifier
    // l'utilisation côté client de façon fiable pour un compte quelconque. Le
    // backend refuse la suppression (contrainte FK) et renvoie un message
    // explicite si le compte est encore utilisé (achats, revenus, catégories).
    await deleteAccountApi(id)

    accountsList.value = accountsList.value.filter(
      (account) => Number(account.id) !== Number(id)
    )

    if (Number(activeAccountId.value) === Number(id)) {
      setActiveAccountId(accountsList.value[0]?.id ?? null)
    }
  }

  // Duplique les catégories (et leurs sous-catégories) d'un compte source vers
  // un compte cible — utile quand un compte n'a encore aucune catégorie
  // (nouveau compte, ou compte créé avant l'isolation par compte). Une
  // catégorie déjà présente dans le compte cible (même nom, même type) est
  // réutilisée plutôt que dupliquée, ce qui rend l'action rejouable sans
  // risque si elle est relancée plus tard.
  async function copyCategoriesFromAccount(sourceAccountId, targetAccountId) {
    error.value = ''

    if (!sourceAccountId || !targetAccountId || Number(sourceAccountId) === Number(targetAccountId)) {
      throw new Error('Choisis un compte source différent du compte cible.')
    }

    const [sourceCategoriesRaw, targetCategoriesRaw, allSubCategoriesRaw] = await Promise.all([
      fetchCategories(sourceAccountId),
      fetchCategories(targetAccountId),
      fetchSubCategories(),
    ])

    const sourceCategories = sourceCategoriesRaw.map(mapCategoryFromApi)
    const targetCategories = targetCategoriesRaw.map(mapCategoryFromApi)
    const allSubCategories = allSubCategoriesRaw.map(mapSubCategoryFromApi)

    let createdCategories = 0
    let createdSubCategories = 0

    for (const sourceCategory of sourceCategories) {
      let targetCategory = targetCategories.find(
        (category) =>
          category.type === sourceCategory.type &&
          category.name.trim().toLowerCase() === sourceCategory.name.trim().toLowerCase()
      )

      if (!targetCategory) {
        const createdRaw = await createCategoryApi({
          name: sourceCategory.name,
          type: sourceCategory.type,
          account_id: Number(targetAccountId),
        })

        targetCategory = mapCategoryFromApi(createdRaw)
        targetCategories.push(targetCategory)
        createdCategories += 1
      }

      const sourceSubCategories = allSubCategories.filter(
        (sub) => Number(sub.categoryId) === Number(sourceCategory.id)
      )
      const existingTargetSubNames = new Set(
        allSubCategories
          .filter((sub) => Number(sub.categoryId) === Number(targetCategory.id))
          .map((sub) => sub.name.trim().toLowerCase())
      )

      for (const sourceSub of sourceSubCategories) {
        if (existingTargetSubNames.has(sourceSub.name.trim().toLowerCase())) continue

        const createdSub = await createSubCategoryApi({
          category_id: Number(targetCategory.id),
          name: sourceSub.name,
        })

        allSubCategories.push(mapSubCategoryFromApi(createdSub))
        createdSubCategories += 1
      }
    }

    // Si le compte cible est celui actuellement affiché, on répercute
    // immédiatement les nouvelles catégories/sous-catégories sans recharger la page.
    if (Number(targetAccountId) === Number(activeAccountId.value)) {
      await loadCategories()
      await loadSubCategories()
    }

    return { createdCategories, createdSubCategories }
  }

  async function createIncome(income) {
    error.value = ''

    const payload = buildIncomePayload(income)
    validateIncomePayload(payload)

    const created = await createIncomeApi(payload)
    const mappedIncome = mapIncomeFromApi(created)

    incomes.value.unshift(mappedIncome)

    return mappedIncome
  }

  async function editIncome(id, income) {
    error.value = ''

    const payload = buildIncomePayload(income)
    validateIncomePayload(payload)

    const updated = await updateIncomeApi(id, payload)
    const mappedIncome = mapIncomeFromApi(updated)

    incomes.value = incomes.value.map((currentIncome) => {
      return currentIncome.id === id ? mappedIncome : currentIncome
    })

    return mappedIncome
  }

  async function removeIncome(id) {
    error.value = ''

    await deleteIncomeApi(id)

    incomes.value = incomes.value.filter((income) => income.id !== id)
  }

  function getCategoryBudgetForMonth(categoryId, monthKey) {
    const overrideKey = `${monthKey}:${categoryId}`
    return categoryBudgetOverrides.value[overrideKey] ?? 0
  }

  function isCategoryBudgetOverridden(categoryId, monthKey) {
    return `${monthKey}:${categoryId}` in categoryBudgetOverrides.value
  }

  async function setCategoryBudgetOverride(categoryId, monthKey, value) {
    const amount = Number(value)

    if (!Number.isFinite(amount) || amount < 0) {
      throw new Error('Le budget doit être un montant positif ou nul.')
    }

    await upsertBudget(categoryId, monthKey, amount)

    const key = `${monthKey}:${categoryId}`
    categoryBudgetOverrides.value = { ...categoryBudgetOverrides.value, [key]: amount }
  }

  async function clearCategoryBudgetOverride(categoryId, monthKey) {
    const key = `${monthKey}:${categoryId}`

    if (!(key in categoryBudgetOverrides.value)) return

    await deleteBudget(categoryId, monthKey)

    const next = { ...categoryBudgetOverrides.value }
    delete next[key]

    categoryBudgetOverrides.value = next
  }

  function setActiveCategory(category) {
    activeCategory.value = category
    selectedPurchaseId.value = filteredPurchases.value[0]?.id ?? null
  }

  function selectPurchase(id) {
    selectedPurchaseId.value = id
  }

  return {
    purchases,
    incomes,
    categoriesList,
    subCategoriesList,
    accountsList,
    activeAccountId,
    setActiveAccountId,
    activeCategory,
    selectedPurchaseId,
    categoryBudgetOverrides,
    isLoading,
    error,
    categoriesById,
    accountsById,
    getAccountName,
    categories,
    filteredPurchases,
    selectedPurchase,
    totalMonthAmount,
    totalIncomeAmount,
    purchaseCount,
    incomeCount,
    averageBasket,
    averageIncome,
    currentMonthExpenseBudget,
    currentMonthExpenseSpent,
    currentMonthBudgetRemaining,
    statements,
    transfers,
    transferLegsForActiveAccount,
    latestLockedStatement,
    activeAccount,
    realCurrentBalance,
    isRealBalanceEstimate,
    realBalanceSource,
    suggestedCategoryBudget,
    currentMonthCategoryBudget,
    loadCategories,
    loadSubCategories,
    loadAccounts,
    loadIncomes,
    loadPurchases,
    submitPurchase,
    editPurchase,
    removePurchase,
    createCategory,
    editCategory,
    removeCategory,
    createSubCategory,
    editSubCategory,
    removeSubCategory,
    createAccount,
    editAccount,
    setAccountOpeningBalance,
    clearAccountOpeningBalance,
    setAccountHasStatements,
    removeAccount,
    copyCategoriesFromAccount,
    loadTransfers,
    createTransfer,
    editTransfer,
    removeTransfer,
    undoTransfer,
    createIncome,
    editIncome,
    removeIncome,
    getCategoryBudgetForMonth,
    isCategoryBudgetOverridden,
    setCategoryBudgetOverride,
    clearCategoryBudgetOverride,
    setActiveCategory,
    selectPurchase,
  }
})
