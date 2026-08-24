<script setup>
import { computed, ref } from 'vue'
import { storeToRefs } from 'pinia'
import ConfirmModal from '../Common/ConfirmModal.vue'
import TransferQuickAddModal from './TransferQuickAddModal.vue'
import { usePurchasesStore } from '../../stores/purchases'
import { formatCurrency, formatDate } from '../../utils/format'
import { signTransferLeg } from '../../utils/realBalance'

const store = usePurchasesStore()
const { transfers } = storeToRefs(store)

const isQuickAddOpen = ref(false)
const editingTransfer = ref(null)

function openEditTransfer(row) {
  editingTransfer.value = row.raw
  isQuickAddOpen.value = true
}

function openCreateTransfer() {
  editingTransfer.value = null
  isQuickAddOpen.value = true
}

function mapTransferRow(t) {
  const leg = signTransferLeg(t, store.activeAccountId)
  const otherAccountName = leg.isOutgoing ? t.toAccountName : t.fromAccountName

  return {
    id: t.id,
    date: leg.date,
    label: `Virement ${leg.isOutgoing ? 'vers' : 'depuis'} ${otherAccountName}`,
    sens: leg.isOutgoing ? 'Sortant' : 'Entrant',
    amount: leg.amount,
    raw: t,
  }
}

const rows = computed(() =>
  [...transfers.value].map(mapTransferRow).sort((a, b) => (a.date < b.date ? 1 : -1))
)

const isUndoingId = ref(null)
const undoError = ref('')

// Un virement créé directement (pas via la case « virement » de Pointage,
// ex: import CSV) n'a pas de ligne d'origine à restaurer.
function canUndoTransfer(row) {
  return Boolean(row.raw.originType) && Boolean(row.raw.originPayload)
}

async function undoTransfer(row) {
  undoError.value = ''
  isUndoingId.value = row.id

  try {
    await store.undoTransfer(row.raw)
  } catch (err) {
    undoError.value = err instanceof Error ? err.message : 'Impossible d’annuler ce virement.'
  } finally {
    isUndoingId.value = null
  }
}

const deleteTarget = ref(null)
const isDeleting = ref(false)
const deleteError = ref('')

function requestDelete(row) {
  deleteError.value = ''
  deleteTarget.value = row
}

async function confirmDelete() {
  if (!deleteTarget.value) return

  isDeleting.value = true
  deleteError.value = ''

  try {
    await store.removeTransfer(deleteTarget.value.id)
    deleteTarget.value = null
  } catch (err) {
    deleteError.value = err instanceof Error ? err.message : 'Impossible de supprimer ce virement.'
  } finally {
    isDeleting.value = false
  }
}
</script>

<template>
  <section class="panel transfer-history-card">
    <div class="panel-header">
      <div>
        <p class="eyebrow">Historique</p>
        <h2>Virements ({{ rows.length }})</h2>
      </div>

      <button class="primary-btn" type="button" @click="openCreateTransfer">
        + Ajouter un mouvement
      </button>
    </div>

    <p v-if="undoError" class="form-error">{{ undoError }}</p>

    <div v-if="!rows.length" class="empty-state-inline">
      Aucun virement pour l'instant sur ce compte.
    </div>

    <div v-else class="transfer-history-table-wrap">
      <table class="transfer-history-table">
        <thead>
          <tr>
            <th>Date</th>
            <th>Libellé</th>
            <th>Sens</th>
            <th>Montant</th>
            <th class="actions-col">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.id" @dblclick="openEditTransfer(row)">
            <td>{{ formatDate(row.date) }}</td>
            <td class="label-cell" :title="row.label">{{ row.label }}</td>
            <td>{{ row.sens }}</td>
            <td :class="row.amount < 0 ? 'amount-negative' : 'amount-positive'">
              {{ row.amount >= 0 ? '+' : '' }}{{ formatCurrency(row.amount) }}
            </td>
            <td class="actions-col" @dblclick.stop>
              <button
                class="icon-btn"
                type="button"
                title="Modifier"
                aria-label="Modifier ce virement"
                @click="openEditTransfer(row)"
              >
                ✏️
              </button>
              <button
                v-if="canUndoTransfer(row)"
                class="icon-btn"
                type="button"
                title="Annuler ce virement et revenir à l'achat/revenu d'origine"
                aria-label="Annuler ce virement"
                :disabled="isUndoingId === row.id"
                @click="undoTransfer(row)"
              >
                {{ isUndoingId === row.id ? '…' : '↩️' }}
              </button>
              <button
                class="icon-btn icon-btn-danger"
                type="button"
                title="Supprimer"
                aria-label="Supprimer ce virement"
                @click="requestDelete(row)"
              >
                🗑️
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <ConfirmModal
      :model-value="Boolean(deleteTarget)"
      title="Supprimer ce virement ?"
      :message="`${deleteTarget?.label} — ${formatCurrency(Math.abs(deleteTarget?.amount ?? 0))} ne sera plus suivi dans l'application.`"
      :note="deleteError || 'Cette action est irréversible.'"
      confirm-label="Supprimer"
      :is-processing="isDeleting"
      @update:model-value="deleteTarget = null"
      @confirm="confirmDelete"
    />

    <TransferQuickAddModal v-model="isQuickAddOpen" :transfer="editingTransfer" />
  </section>
</template>

<style scoped>
.transfer-history-card {
  padding: 0.95rem;
}

.panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.7rem;
  flex-wrap: wrap;
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

.form-error {
  margin: 0 0 0.9rem;
  padding: 0.85rem 1rem;
  border-radius: 14px;
  background: rgba(239, 68, 68, 0.12);
  color: var(--negative-text);
  border: 1px solid rgba(239, 68, 68, 0.18);
  font-size: 0.88rem;
}

.empty-state-inline {
  padding: 1.4rem;
  text-align: center;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.92rem;
}

.transfer-history-table-wrap {
  max-height: 420px;
  overflow: auto;
  border-radius: 14px;
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.06));
}

.transfer-history-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.86rem;
}

.transfer-history-table th {
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

.transfer-history-table td {
  padding: 0.55rem 0.6rem;
  border-top: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
  color: var(--text, #eef1f3);
  vertical-align: middle;
}

.label-cell {
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

.icon-btn + .icon-btn {
  margin-left: 0.4rem;
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
</style>
