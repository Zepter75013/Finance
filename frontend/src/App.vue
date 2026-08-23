<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'

import { usePurchasesStore } from './stores/purchases'
import { useAuthStore } from './stores/auth'
import { useRecurringStore } from './stores/recurring'

import AppSidebar from './components/layout/AppSidebar.vue'
import LoginView from './components/auth/LoginView.vue'
import ChangePasswordModal from './components/auth/ChangePasswordModal.vue'
import AboutModal from './components/Common/AboutModal.vue'
import DashboardView from './components/dashboard/DashboardView.vue'
import MonthlyOverviewView from './components/dashboard/MonthlyOverviewView.vue'
import RecurringView from './components/recurring/RecurringView.vue'
import PurchasesView from './components/dashboard/PurchasesView.vue'
import PurchaseFormModal from './components/dashboard/PurchaseFormModal.vue'
import DeletePurchaseModal from './components/dashboard/DeletePurchaseModal.vue'
import BudgetsView from './components/budgets/BudgetsView.vue'
import ReconciliationView from './components/reconciliation/ReconciliationView.vue'
import CategoriesView from './components/categories/CategoriesView.vue'
import AccountsView from './components/accounts/AccountsView.vue'
import CategoryFormModal from './components/categories/CategoryFormModal.vue'
import DeleteCategoryModal from './components/categories/DeleteCategoryModal.vue'
import IncomesView from './components/incomes/IncomesView.vue'
import IncomeFormModal from './components/incomes/IncomeFormModal.vue'
import DeleteIncomeModal from './components/incomes/DeleteIncomeModal.vue'
import ImportView from './components/import/ImportView.vue'
import ImportCategoriesView from './components/categories/ImportCategoriesView.vue'
import ReportsView from './components/reports/ReportsView.vue'
import UsersView from './components/settings/UsersView.vue'
import AuditLogView from './components/auditlog/AuditLogView.vue'
import PreferencesView from './components/settings/PreferencesView.vue'
import { useThemeStore } from './stores/theme'
import { useIdleTimeout } from './composables/useIdleTimeout'

const purchasesStore = usePurchasesStore()
const authStore = useAuthStore()
const recurringStore = useRecurringStore()
useThemeStore()

const { isAuthenticated, isCheckingSession, username, avatarUrl } = storeToRefs(authStore)

const isChangePasswordModalOpen = ref(false)
const isAboutModalOpen = ref(false)

const activeView = ref('dashboard')
const isPurchaseModalOpen = ref(false)
const editingPurchaseId = ref(null)
const feedbackMessage = ref('')
const feedbackType = ref('success')
const feedbackTimeoutId = ref(null)
const deleteModalPurchase = ref(null)
const isDeletingPurchase = ref(false)

const isCategoryModalOpen = ref(false)
const editingCategory = ref(null)
const createCategoryType = ref('achat')
const deleteModalCategory = ref(null)
const isDeletingCategory = ref(false)
const dashboardOrigin = ref(null)

const isIncomeModalOpen = ref(false)
const editingIncome = ref(null)
const deleteModalIncome = ref(null)
const isDeletingIncome = ref(false)

const {
  isLoading,
  error,
  selectedPurchaseId,
  activeCategory,
} = storeToRefs(purchasesStore)

const editingPurchase = computed(() => {
  if (!editingPurchaseId.value) {
    return null
  }

  return (
    purchasesStore.purchases.find(
      (purchase) => purchase.id === editingPurchaseId.value
    ) || null
  )
})

const showBackToCategories = computed(() => {
  return (
    activeView.value === 'purchases' &&
    dashboardOrigin.value === 'categories' &&
    activeCategory.value !== 'Toutes'
  )
})

onMounted(async () => {
  await authStore.checkSession()

  if (isAuthenticated.value) {
    purchasesStore.loadPurchases()
  }
})

watch(isAuthenticated, (authenticated) => {
  if (authenticated) {
    purchasesStore.loadPurchases()
  }
})

// Une transaction récurrente peut être créée côté serveur pendant qu'un
// onglet reste ouvert en arrière-plan (planificateur de fond) — sans ce
// rechargement, elle n'apparaîtrait jamais sans rafraîchissement manuel.
function handleVisibilityChange() {
  if (document.visibilityState === 'visible' && isAuthenticated.value) {
    purchasesStore.loadPurchases()
    recurringStore.loadRecurring()
  }
}

onMounted(() => {
  document.addEventListener('visibilitychange', handleVisibilityChange)
})

onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})

async function handleLogout() {
  await authStore.logout()
  activeView.value = 'dashboard'
}

// Déconnexion automatique après 15 minutes sans interaction (souris, clavier,
// scroll, tactile) — ne surveille rien tant que l'utilisateur n'est pas
// connecté (donc pas sur l'écran de connexion lui-même).
const IDLE_TIMEOUT_MS = 15 * 60 * 1000

async function handleIdleLogout() {
  try {
    localStorage.setItem('idleLogoutMessage', '1')
  } catch {
    // stockage indisponible : tant pis pour le message, la déconnexion reste prioritaire
  }

  await handleLogout()
}

useIdleTimeout(isAuthenticated, IDLE_TIMEOUT_MS, handleIdleLogout)

onBeforeUnmount(() => {
  clearFeedbackTimeout()
})

function clearFeedbackTimeout() {
  if (feedbackTimeoutId.value) {
    window.clearTimeout(feedbackTimeoutId.value)
    feedbackTimeoutId.value = null
  }
}

function showFeedback(message, type = 'success') {
  feedbackMessage.value = message
  feedbackType.value = type

  clearFeedbackTimeout()

  feedbackTimeoutId.value = window.setTimeout(() => {
    feedbackMessage.value = ''
    feedbackTimeoutId.value = null
  }, 3000)
}

function navigateTo(view) {
  activeView.value = view

  if (view !== 'purchases') {
    dashboardOrigin.value = null
  }
}

function handlePurchaseSaved(payload) {
  editingPurchaseId.value = null

  if (payload?.type === 'edit') {
    showFeedback(`Achat mis à jour : ${payload.merchant}`)
    return
  }

  showFeedback(`Achat ajouté : ${payload?.merchant ?? ''}`)
}

function openPurchaseModal() {
  editingPurchaseId.value = null
  isPurchaseModalOpen.value = true
}

function openEditPurchaseModal(purchase) {
  if (!purchase?.id) {
    return
  }

  editingPurchaseId.value = purchase.id
  isPurchaseModalOpen.value = true
}

function closePurchaseModal(value) {
  isPurchaseModalOpen.value = value

  if (!value) {
    editingPurchaseId.value = null
  }
}

function openDeleteModal(purchase) {
  if (!purchase?.id) {
    return
  }

  deleteModalPurchase.value = purchase
}

function closeDeleteModal() {
  if (isDeletingPurchase.value) {
    return
  }

  deleteModalPurchase.value = null
}

async function confirmDeletePurchase(purchase) {
  const targetPurchase = purchase ?? deleteModalPurchase.value

  if (!targetPurchase?.id) {
    return
  }

  isDeletingPurchase.value = true

  try {
    const deletedId = targetPurchase.id
    const wasSelected = selectedPurchaseId.value === deletedId

    await purchasesStore.removePurchase(deletedId)

    if (wasSelected) {
      const nextPurchase = purchasesStore.purchases[0] ?? null
      purchasesStore.selectPurchase(nextPurchase?.id ?? null)
    }

    showFeedback('Achat supprimé avec succès.')
    deleteModalPurchase.value = null
  } catch (err) {
    const message =
      err instanceof Error ? err.message : 'Impossible de supprimer l’achat.'
    showFeedback(message, 'error')
  } finally {
    isDeletingPurchase.value = false
  }
}

function openCreateCategoryModal(type = 'achat') {
  editingCategory.value = null
  createCategoryType.value = type === 'revenu' ? 'revenu' : 'achat'
  isCategoryModalOpen.value = true
}

function openEditCategoryModal(category) {
  if (!category?.id) {
    return
  }

  editingCategory.value = category
  isCategoryModalOpen.value = true
}

function closeCategoryModal(value) {
  isCategoryModalOpen.value = value

  if (!value) {
    editingCategory.value = null
  }
}

async function handleCategorySaved(payload) {
  if (!payload?.category) {
    return
  }

  try {
    if (payload.type === 'edit' && editingCategory.value?.id) {
      const updated = await purchasesStore.editCategory(
        editingCategory.value.id,
        payload.category
      )
      showFeedback(`Catégorie mise à jour : ${updated?.name ?? payload.category.name}`)
    } else {
      const created = await purchasesStore.createCategory(payload.category)
      showFeedback(`Catégorie ajoutée : ${created?.name ?? payload.category.name}`)
    }

    editingCategory.value = null
    isCategoryModalOpen.value = false
  } catch (err) {
    const message =
      err instanceof Error ? err.message : "Impossible d'enregistrer la catégorie."
    showFeedback(message, 'error')
  }
}

function openDeleteCategoryModal(category) {
  if (!category?.id) {
    return
  }

  deleteModalCategory.value = category
}

function closeDeleteCategoryModal() {
  if (isDeletingCategory.value) {
    return
  }

  deleteModalCategory.value = null
}

async function confirmDeleteCategory(category) {
  const targetCategory = category ?? deleteModalCategory.value

  if (!targetCategory?.id) {
    return
  }

  isDeletingCategory.value = true

  try {
    await purchasesStore.removeCategory(targetCategory.id)
    showFeedback('Catégorie supprimée avec succès.')
    deleteModalCategory.value = null
  } catch (err) {
    const message =
      err instanceof Error ? err.message : 'Impossible de supprimer la catégorie.'
    showFeedback(message, 'error')
  } finally {
    isDeletingCategory.value = false
  }
}

function openCategoryPurchases(category) {
  if (!category?.name) {
    return
  }

  purchasesStore.setActiveCategory(category.name)
  dashboardOrigin.value = 'categories'
  activeView.value = 'purchases'
  showFeedback(`Affichage des achats pour la catégorie : ${category.name}`)
}

function returnToCategories() {
  purchasesStore.setActiveCategory('Toutes')
  dashboardOrigin.value = null
  activeView.value = 'categories'
}

function openCreateIncomeModal() {
  editingIncome.value = null
  isIncomeModalOpen.value = true
}

function openEditIncomeModal(income) {
  if (!income?.id) {
    return
  }

  editingIncome.value = income
  isIncomeModalOpen.value = true
}

function closeIncomeModal(value) {
  isIncomeModalOpen.value = value

  if (!value) {
    editingIncome.value = null
  }
}

function handleIncomeSaved(payload) {
  editingIncome.value = null

  if (payload?.type === 'edit') {
    showFeedback(`Revenu mis à jour : ${payload.source}`)
    return
  }

  showFeedback(`Revenu ajouté : ${payload?.source ?? ''}`)
}

function openDeleteIncomeModal(income) {
  if (!income?.id) {
    return
  }

  deleteModalIncome.value = income
}

function closeDeleteIncomeModal() {
  if (isDeletingIncome.value) {
    return
  }

  deleteModalIncome.value = null
}

async function confirmDeleteIncome(income) {
  const targetIncome = income ?? deleteModalIncome.value

  if (!targetIncome?.id) {
    return
  }

  isDeletingIncome.value = true

  try {
    await purchasesStore.removeIncome(targetIncome.id)
    showFeedback('Revenu supprimé avec succès.')
    deleteModalIncome.value = null
  } catch (err) {
    const message =
      err instanceof Error ? err.message : 'Impossible de supprimer le revenu.'
    showFeedback(message, 'error')
  } finally {
    isDeletingIncome.value = false
  }
}

function handleImported(result) {
  const { successCount, failCount } = result ?? {}

  if (!failCount) {
    showFeedback(
      `${successCount} ligne${successCount > 1 ? 's' : ''} importée${successCount > 1 ? 's' : ''} avec succès.`
    )
    return
  }

  showFeedback(
    `${successCount} ligne${successCount > 1 ? 's' : ''} importée${successCount > 1 ? 's' : ''}, ${failCount} échec${failCount > 1 ? 's' : ''}.`,
    'error'
  )
}

function handleBulkDeleted(result) {
  const { successCount, failCount } = result ?? {}

  if (!failCount) {
    showFeedback(
      `${successCount} ligne${successCount > 1 ? 's' : ''} supprimée${successCount > 1 ? 's' : ''}.`
    )
    return
  }

  showFeedback(
    `${successCount} supprimée${successCount > 1 ? 's' : ''}, ${failCount} échec${failCount > 1 ? 's' : ''}.`,
    'error'
  )
}
</script>

<template>
  <main v-if="isCheckingSession" class="auth-loading">
    Vérification de la session…
  </main>

  <LoginView v-else-if="!isAuthenticated" />

  <div v-else class="app-shell">
    <AppSidebar
      :active-view="activeView"
      :username="username"
      :avatar-url="avatarUrl"
      @navigate="navigateTo"
      @logout="handleLogout"
      @change-password="isChangePasswordModalOpen = true"
      @about="isAboutModalOpen = true"
    />

    <div class="app-main">
      <main class="dashboard-content">
        <section
          v-if="feedbackMessage"
          class="dashboard-feedback"
          :class="`dashboard-feedback--${feedbackType}`"
        >
          {{ feedbackMessage }}
        </section>

        <section v-if="isLoading" class="dashboard-state">
          Chargement des données...
        </section>

        <section v-else-if="error" class="dashboard-state dashboard-state--error">
          {{ error }}
        </section>

        <BudgetsView v-else-if="activeView === 'budgets'" />

        <ReconciliationView v-else-if="activeView === 'reconciliation'" />

        <CategoriesView
          v-else-if="activeView === 'categories'"
          @create="openCreateCategoryModal"
          @edit="openEditCategoryModal"
          @delete="openDeleteCategoryModal"
          @open-purchases="openCategoryPurchases"
          @bulk-delete="handleBulkDeleted"
        />

        <IncomesView
          v-else-if="activeView === 'incomes'"
          :is-loading="isLoading"
          @create="openCreateIncomeModal"
          @edit="openEditIncomeModal"
          @delete="openDeleteIncomeModal"
          @bulk-delete="handleBulkDeleted"
        />

        <ImportView v-else-if="activeView === 'import'" @imported="handleImported" />

        <ImportCategoriesView v-else-if="activeView === 'import-categories'" />

        <ReportsView v-else-if="activeView === 'reports'" />

        <AccountsView v-else-if="activeView === 'accounts'" />

        <UsersView v-else-if="activeView === 'users'" />

        <AuditLogView v-else-if="activeView === 'audit-log'" />

        <PreferencesView v-else-if="activeView === 'preferences'" />

        <DashboardView
          v-else-if="activeView === 'dashboard'"
          @open-recurring="navigateTo('recurring')"
        />

        <MonthlyOverviewView v-else-if="activeView === 'monthly'" />

        <RecurringView v-else-if="activeView === 'recurring'" />

        <template v-else>
          <section
            v-if="showBackToCategories"
            class="dashboard-context-bar"
          >
            <button
              class="ghost-btn dashboard-back-btn"
              type="button"
              @click="returnToCategories"
            >
              ← Retour aux catégories
            </button>

            <p>
              Vue filtrée sur la catégorie :
              <strong>{{ activeCategory }}</strong>
            </p>
          </section>

          <PurchasesView
            :is-loading="isLoading"
            @create="openPurchaseModal"
            @edit="openEditPurchaseModal"
            @delete="openDeleteModal"
            @bulk-delete="handleBulkDeleted"
          />
        </template>
      </main>
    </div>

    <PurchaseFormModal
      :model-value="isPurchaseModalOpen"
      :purchase="editingPurchase"
      @update:model-value="closePurchaseModal"
      @saved="handlePurchaseSaved"
    />

    <DeletePurchaseModal
      :model-value="Boolean(deleteModalPurchase)"
      :purchase="deleteModalPurchase"
      :is-deleting="isDeletingPurchase"
      @update:model-value="closeDeleteModal"
      @confirm="confirmDeletePurchase"
    />

    <CategoryFormModal
      :model-value="isCategoryModalOpen"
      :category="editingCategory"
      :default-type="createCategoryType"
      @update:model-value="closeCategoryModal"
      @saved="handleCategorySaved"
    />

    <DeleteCategoryModal
      :model-value="Boolean(deleteModalCategory)"
      :category="deleteModalCategory"
      :is-deleting="isDeletingCategory"
      @update:model-value="closeDeleteCategoryModal"
      @confirm="confirmDeleteCategory"
    />

    <IncomeFormModal
      :model-value="isIncomeModalOpen"
      :income="editingIncome"
      @update:model-value="closeIncomeModal"
      @saved="handleIncomeSaved"
    />

    <DeleteIncomeModal
      :model-value="Boolean(deleteModalIncome)"
      :income="deleteModalIncome"
      :is-deleting="isDeletingIncome"
      @update:model-value="closeDeleteIncomeModal"
      @confirm="confirmDeleteIncome"
    />

    <ChangePasswordModal v-model="isChangePasswordModalOpen" />
    <AboutModal v-model="isAboutModalOpen" />
  </div>
</template>

<style scoped>
.auth-loading {
  min-height: 100vh;
  display: grid;
  place-items: center;
  background: var(--bg, #1f2226);
  color: var(--text-soft, #b3bbc4);
  font-size: 0.95rem;
}
</style>

<style scoped>
.dashboard-feedback {
  margin-bottom: 1rem;
  padding: 0.9rem 1rem;
  border-radius: 14px;
  font-weight: 600;
  animation: feedback-slide-in 180ms ease;
}

.dashboard-feedback--success {
  background: rgba(34, 197, 94, 0.12);
  color: #166534;
  border: 1px solid rgba(34, 197, 94, 0.2);
}

.dashboard-feedback--error {
  background: rgba(239, 68, 68, 0.12);
  color: #991b1b;
  border: 1px solid rgba(239, 68, 68, 0.2);
}

.dashboard-context-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
  padding: 1rem 1.1rem;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.dashboard-context-bar p {
  margin: 0;
  color: var(--text-soft, #b3bbc4);
}

.dashboard-back-btn {
  flex-shrink: 0;
}

.ghost-btn {
  border: none;
  border-radius: 14px;
  padding: 0.85rem 1rem;
  font-weight: 600;
  cursor: pointer;
  background: rgba(255, 255, 255, 0.06);
  color: #eef1f3;
  transition: transform 140ms ease, opacity 140ms ease, background 140ms ease;
}

.ghost-btn:hover {
  transform: translateY(-1px);
}

@keyframes feedback-slide-in {
  from {
    opacity: 0;
    transform: translateY(-6px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 720px) {
  .dashboard-context-bar {
    flex-direction: column;
    align-items: flex-start;
  }

  .dashboard-back-btn {
    width: 100%;
  }
}
</style>
