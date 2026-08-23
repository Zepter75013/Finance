<script setup>
import { computed, ref } from 'vue'
import { storeToRefs } from 'pinia'

import PageHero from '../Common/PageHero.vue'
import QuickCreateModal from '../Common/QuickCreateModal.vue'
import ConfirmModal from '../Common/ConfirmModal.vue'
import CopyCategoriesModal from './CopyCategoriesModal.vue'
import AccountOpeningBalanceModal from './AccountOpeningBalanceModal.vue'
import TransferFormModal from '../transfers/TransferFormModal.vue'
import { usePurchasesStore } from '../../stores/purchases'
import { formatCurrency } from '../../utils/format'

const store = usePurchasesStore()
const { accountsList } = storeToRefs(store)


// Les achats/revenus en mémoire ne couvrent plus que le compte actif (chaque
// compte a désormais ses propres données) — l'utilisation de chaque compte
// vient donc directement des compteurs renvoyés par l'API plutôt que d'un
// filtrage local qui ne verrait que le compte actuellement sélectionné.
const accountCards = computed(() => {
  return accountsList.value
    .map((account) => ({
      ...account,
      isUsed:
        account.purchaseCount > 0 ||
        account.incomeCount > 0 ||
        account.categoryCount > 0 ||
        account.transferCount > 0,
    }))
    .sort((a, b) => a.name.localeCompare(b.name, 'fr', { sensitivity: 'base' }))
})

const copyModalTarget = ref(null)
const isCopyModalOpen = ref(false)

function handleCopyCategories(account) {
  copyModalTarget.value = account
  isCopyModalOpen.value = true
}

function handleCategoriesCopied() {
  store.loadAccounts()
}

const transferSourceAccountId = ref(null)
const isTransferModalOpen = ref(false)

function handleTransfer(account) {
  transferSourceAccountId.value = account.id
  isTransferModalOpen.value = true
}

const openingBalanceTarget = ref(null)
const isOpeningBalanceModalOpen = ref(false)

function handleOpeningBalance(account) {
  openingBalanceTarget.value = account
  isOpeningBalanceModalOpen.value = true
}

const isModalOpen = ref(false)
const modalMode = ref('create') // 'create' | 'rename'
const modalTarget = ref(null)
const modalError = ref('')
const isSubmitting = ref(false)

function handleCreate() {
  modalMode.value = 'create'
  modalTarget.value = null
  modalError.value = ''
  isModalOpen.value = true
}

function handleRename(account) {
  modalMode.value = 'rename'
  modalTarget.value = account
  modalError.value = ''
  isModalOpen.value = true
}

async function submitModal(name) {
  modalError.value = ''
  isSubmitting.value = true

  try {
    if (modalMode.value === 'rename') {
      await store.editAccount(modalTarget.value.id, name)
    } else {
      await store.createAccount(name)
    }

    isModalOpen.value = false
  } catch (err) {
    modalError.value = err instanceof Error ? err.message : 'Une erreur est survenue.'
  } finally {
    isSubmitting.value = false
  }
}

const deleteTarget = ref(null)
const isDeleteConfirmOpen = ref(false)
const isDeleting = ref(false)
const deleteError = ref('')

function requestDelete(account) {
  deleteError.value = ''
  deleteTarget.value = account
  isDeleteConfirmOpen.value = true
}

async function confirmDelete() {
  if (!deleteTarget.value) return

  isDeleting.value = true

  try {
    await store.removeAccount(deleteTarget.value.id)
    isDeleteConfirmOpen.value = false
    deleteTarget.value = null
  } catch (err) {
    deleteError.value = err instanceof Error ? err.message : 'Impossible de supprimer ce compte.'
    isDeleteConfirmOpen.value = false
  } finally {
    isDeleting.value = false
  }
}
</script>

<template>
  <main class="dashboard-content accounts-view">
    <PageHero
      eyebrow="Organisation"
      title="Comptes"
      description="Gère tes comptes bancaires pour séparer achats, revenus et pointage par compte."
    >
      <template #actions>
        <button class="primary-btn" type="button" @click="handleCreate">
          + Nouveau compte
        </button>
      </template>
    </PageHero>

    <p v-if="deleteError" class="form-error">{{ deleteError }}</p>

    <section v-if="!accountCards.length" class="panel empty-state">
      <div class="empty-state-icon">€</div>
      <h2>Aucun compte pour le moment</h2>
      <p>Crée un premier compte pour commencer à rattacher tes achats et revenus.</p>
      <button class="primary-btn" type="button" @click="handleCreate">
        Créer un compte
      </button>
    </section>

    <section v-else class="panel accounts-list-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Comptes</p>
          <h2>{{ accountCards.length }} compte{{ accountCards.length > 1 ? 's' : '' }}</h2>
        </div>
      </div>

      <div class="accounts-list">
        <article v-for="account in accountCards" :key="account.id" class="account-row">
          <div class="account-row-main">
            <span class="row-icon" aria-hidden="true">€</span>
            <div class="account-row-info">
              <h3>{{ account.name }}</h3>
              <p class="account-row-meta">
                {{ account.purchaseCount }} achat{{ account.purchaseCount > 1 ? 's' : '' }}
                ({{ formatCurrency(account.totalExpense) }}) ·
                {{ account.incomeCount }} revenu{{ account.incomeCount > 1 ? 's' : '' }}
                ({{ formatCurrency(account.totalIncome) }}) ·
                {{ account.categoryCount }} catégorie{{ account.categoryCount > 1 ? 's' : '' }}
              </p>
            </div>
          </div>

          <div class="account-row-actions">
            <button class="ghost-btn" type="button" @click="handleCopyCategories(account)">
              Copier des catégories
            </button>
            <button class="ghost-btn" type="button" @click="handleRename(account)">
              Renommer
            </button>
            <button class="ghost-btn" type="button" @click="handleOpeningBalance(account)">
              Solde initial
            </button>
            <button class="ghost-btn" type="button" @click="handleTransfer(account)">
              Transférer
            </button>
            <button
              class="ghost-btn ghost-btn-danger"
              type="button"
              :disabled="account.isUsed"
              :title="account.isUsed ? 'Compte utilisé par des achats, des revenus ou des catégories — suppression impossible' : ''"
              @click="requestDelete(account)"
            >
              Supprimer
            </button>
          </div>
        </article>
      </div>
    </section>

    <QuickCreateModal
      v-model="isModalOpen"
      :eyebrow="modalMode === 'rename' ? 'Modifier un compte' : 'Nouveau compte'"
      :title="modalMode === 'rename' ? `Renommer « ${modalTarget?.name ?? ''} »` : 'Ajouter un compte'"
      label="Nom du compte"
      placeholder="Ex: Compte courant"
      :confirm-label="modalMode === 'rename' ? 'Enregistrer' : 'Créer'"
      :initial-value="modalMode === 'rename' ? modalTarget?.name ?? '' : ''"
      :is-submitting="isSubmitting"
      :error-message="modalError"
      @confirm="submitModal"
    />

    <ConfirmModal
      v-model="isDeleteConfirmOpen"
      :title="`Supprimer « ${deleteTarget?.name ?? ''} » ?`"
      message="Ce compte sera définitivement supprimé."
      note="Un compte utilisé par des achats, des revenus ou des catégories ne peut pas être supprimé."
      confirm-label="Supprimer"
      :is-processing="isDeleting"
      @confirm="confirmDelete"
    />

    <CopyCategoriesModal
      v-model="isCopyModalOpen"
      :target-account="copyModalTarget"
      :accounts="accountsList"
      @copied="handleCategoriesCopied"
    />

    <AccountOpeningBalanceModal
      v-model="isOpeningBalanceModalOpen"
      :account="openingBalanceTarget"
    />

    <TransferFormModal
      v-model="isTransferModalOpen"
      :default-from-account-id="transferSourceAccountId"
    />
  </main>
</template>

<style scoped>
.accounts-view {
  display: grid;
  gap: 0.9rem;
}

.form-error {
  margin: 0;
  padding: 0.85rem 1rem;
  border-radius: 14px;
  background: rgba(239, 68, 68, 0.12);
  color: var(--negative-text);
  border: 1px solid rgba(239, 68, 68, 0.18);
  font-size: 0.88rem;
}

.accounts-list-card {
  padding: 0.95rem;
}

.accounts-list {
  display: grid;
  gap: 0.5rem;
  margin-top: 0.9rem;
}

.account-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.7rem 0.85rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.028);
  border: 1px solid rgba(var(--tint-rgb), 0.05);
}

.account-row-main {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  min-width: 0;
}

.row-icon {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 10px;
  display: grid;
  place-items: center;
  background: rgba(var(--tint-rgb), 0.06);
  color: var(--text-dim, #8a939d);
  font-weight: 700;
}

.account-row-info h3 {
  margin: 0;
  color: var(--text, #eef1f3);
  font-size: 0.95rem;
}

.account-row-meta {
  margin: 0.15rem 0 0;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.8rem;
}

.account-row-actions {
  display: flex;
  gap: 0.5rem;
  flex-shrink: 0;
}

.ghost-btn {
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  border-radius: 10px;
  padding: 0.45rem 0.8rem;
  font-size: 0.82rem;
  font-weight: 600;
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--text-soft, #b3bbc4);
  cursor: pointer;
  transition: background 140ms ease, transform 140ms ease;
}

.ghost-btn:hover {
  background: rgba(var(--tint-rgb), 0.08);
  transform: translateY(-1px);
}

.ghost-btn-danger {
  color: var(--negative-text);
  border-color: rgba(220, 38, 38, 0.18);
  background: rgba(220, 38, 38, 0.1);
}

.ghost-btn-danger:hover:not(:disabled) {
  background: rgba(220, 38, 38, 0.16);
}

.ghost-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
  transform: none;
}

.empty-state {
  padding: 1.8rem 1.3rem;
  text-align: center;
}

.empty-state h2 {
  margin: 0.75rem 0 0.5rem;
  color: var(--text, #eef1f3);
}

.empty-state p {
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

@media (max-width: 640px) {
  .account-row {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
