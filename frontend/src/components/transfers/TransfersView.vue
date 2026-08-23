<script setup>
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'

import PageHero from '../Common/PageHero.vue'
import ConfirmModal from '../Common/ConfirmModal.vue'
import TransferFormModal from './TransferFormModal.vue'
import { usePurchasesStore } from '../../stores/purchases'
import { formatCurrency, formatDate } from '../../utils/format'

const store = usePurchasesStore()
const { accountsList, transfers, isLoading } = storeToRefs(store)

const activeAccountId = computed({
  get: () => store.activeAccountId,
  set: (value) => store.setActiveAccountId(value),
})

const loadError = ref('')

const sortedTransfers = computed(() => [...transfers.value].sort((a, b) => (a.date < b.date ? 1 : -1)))

const isModalOpen = ref(false)
const editingItem = ref(null)

function openCreateModal() {
  editingItem.value = null
  isModalOpen.value = true
}

function openEditModal(item) {
  editingItem.value = item
  isModalOpen.value = true
}

const deleteTarget = ref(null)
const isDeleting = ref(false)

function requestDelete(item) {
  deleteTarget.value = item
}

async function confirmDelete() {
  if (!deleteTarget.value) return

  isDeleting.value = true

  try {
    await store.removeTransfer(deleteTarget.value.id)
    deleteTarget.value = null
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : 'Impossible de supprimer le virement.'
  } finally {
    isDeleting.value = false
  }
}

onMounted(() => {
  if (!transfers.value.length) {
    store.loadTransfers()
  }
})
</script>

<template>
  <main class="dashboard-content transfers-view">
    <PageHero
      eyebrow="Mouvements internes"
      title="Transferts"
      description="Virements entre tes comptes — exclus des budgets et des rapports."
    >
      <template #actions>
        <div v-if="accountsList.length > 1" class="account-switcher">
          <select v-model.number="activeAccountId" class="account-switcher-select">
            <option v-for="account in accountsList" :key="account.id" :value="account.id">
              {{ account.name }}
            </option>
          </select>
        </div>

        <button class="primary-btn" type="button" @click="openCreateModal">+ Nouveau virement</button>
      </template>
    </PageHero>

    <p v-if="loadError" class="form-error">{{ loadError }}</p>

    <section class="panel transfers-list-card">
      <p v-if="isLoading" class="transfers-empty">Chargement...</p>

      <p v-else-if="!sortedTransfers.length" class="transfers-empty">
        Aucun virement pour l’instant — clique sur « Nouveau virement » pour en créer un.
      </p>

      <div v-else class="transfers-list">
        <article v-for="item in sortedTransfers" :key="item.id" class="transfer-row">
          <div class="transfer-row-main">
            <strong>{{ item.fromAccountName }} → {{ item.toAccountName }}</strong>
            <span class="transfer-row-meta">
              {{ formatDate(item.date) }}
              <template v-if="item.note"> · {{ item.note }}</template>
            </span>
          </div>

          <strong class="transfer-row-amount">{{ formatCurrency(item.amount) }}</strong>

          <div class="transfer-row-actions">
            <button class="ghost-btn" type="button" @click="openEditModal(item)">Modifier</button>
            <button class="ghost-btn ghost-btn-danger" type="button" @click="requestDelete(item)">
              Supprimer
            </button>
          </div>
        </article>
      </div>
    </section>

    <TransferFormModal v-model="isModalOpen" :item="editingItem" />

    <ConfirmModal
      :model-value="Boolean(deleteTarget)"
      title="Supprimer ce virement ?"
      :message="`Cette action est définitive. Le virement de ${deleteTarget?.fromAccountName ?? ''} vers ${deleteTarget?.toAccountName ?? ''} sera supprimé.`"
      confirm-label="Supprimer"
      :is-processing="isDeleting"
      @update:model-value="deleteTarget = null"
      @confirm="confirmDelete"
    />
  </main>
</template>

<style scoped>
.transfers-view {
  display: grid;
  gap: 0.9rem;
}

.account-switcher {
  display: flex;
  align-items: center;
  padding: 0.3rem 0.7rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.04);
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
}

.account-switcher-select {
  border: none;
  background: transparent;
  color: var(--text, #eef1f3);
  font-weight: 700;
  font-size: 0.95rem;
  padding: 0.1rem 0;
  outline: none;
  cursor: pointer;
}

.transfers-list-card {
  padding: 1.1rem;
}

.transfers-empty {
  margin: 0;
  color: var(--text-dim, #8a939d);
  font-size: 0.88rem;
}

.transfers-list {
  display: grid;
  gap: 0.6rem;
}

.transfer-row {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.85rem 1rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.03);
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
  flex-wrap: wrap;
}

.transfer-row-main {
  display: grid;
  gap: 0.2rem;
  min-width: 0;
  flex: 1;
}

.transfer-row-main strong {
  color: var(--text, #eef1f3);
  font-size: 0.92rem;
}

.transfer-row-meta {
  color: var(--text-dim, #8a939d);
  font-size: 0.78rem;
}

.transfer-row-amount {
  flex-shrink: 0;
  color: var(--text, #eef1f3);
  font-size: 0.95rem;
}

.transfer-row-actions {
  display: flex;
  gap: 0.5rem;
  flex-shrink: 0;
}

.transfer-row-actions .ghost-btn {
  padding: 0.5rem 0.85rem;
  font-size: 0.82rem;
}

.ghost-btn-danger {
  color: var(--negative-text);
}

.form-error {
  margin: 0;
  padding: 0.85rem 1rem;
  border-radius: 14px;
  background: rgba(239, 68, 68, 0.12);
  color: var(--negative-text);
  border: 1px solid rgba(239, 68, 68, 0.18);
  font-size: 0.92rem;
}
</style>
