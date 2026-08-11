<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'

import PageHero from '../Common/PageHero.vue'
import ConfirmModal from '../Common/ConfirmModal.vue'
import { usePurchasesStore } from '../../stores/purchases'
import {
  fetchRecurring,
  createRecurring,
  updateRecurring,
  deleteRecurring,
  executeRecurringNow,
} from '../../services/recurring'
import { formatCurrency } from '../../utils/format'

const store = usePurchasesStore()
const { accountsList, categoriesList } = storeToRefs(store)

const activeAccountId = computed({
  get: () => store.activeAccountId,
  set: (value) => store.setActiveAccountId(value),
})

const availableCategories = computed(() =>
  categoriesList.value.filter((category) => category.type !== 'revenu')
)

const items = ref([])
const isLoading = ref(false)
const loadError = ref('')

async function loadRecurring() {
  if (!activeAccountId.value) {
    items.value = []
    return
  }

  isLoading.value = true
  loadError.value = ''

  try {
    items.value = await fetchRecurring(activeAccountId.value)
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : 'Impossible de charger les récurrences.'
  } finally {
    isLoading.value = false
  }
}

watch(activeAccountId, loadRecurring, { immediate: true })

const sortedItems = computed(() => [...items.value].sort((a, b) => a.day_of_month - b.day_of_month))

function categoryName(categoryId) {
  return categoriesList.value.find((category) => category.id === categoryId)?.name || '—'
}

function formatNextRunDate(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat('fr-FR', { day: '2-digit', month: '2-digit', year: 'numeric' }).format(date)
}

function isOverdue(item) {
  const date = new Date(item.next_run_date)
  if (Number.isNaN(date.getTime())) return false

  const today = new Date()
  today.setHours(0, 0, 0, 0)

  return date <= today
}

const executingId = ref(null)

async function executeNow(item) {
  executingId.value = item.id
  loadError.value = ''

  try {
    await executeRecurringNow(item.id)
    await loadRecurring()
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : 'Impossible d’exécuter cette occurrence.'
  } finally {
    executingId.value = null
  }
}

const isModalOpen = ref(false)
const editingItem = ref(null)
const isSubmitting = ref(false)
const submitError = ref('')

const initialForm = () => ({
  type: editingItem.value?.type ?? 'achat',
  merchant: editingItem.value?.merchant ?? '',
  source: editingItem.value?.source ?? '',
  categoryId: editingItem.value?.category_id ?? null,
  paymentMethod: editingItem.value?.payment_method ?? 'Carte bancaire',
  category: editingItem.value?.category ?? '',
  subCategory: editingItem.value?.sub_category ?? '',
  amount: editingItem.value?.amount ?? null,
  firstRunDate: editingItem.value?.next_run_date
    ? editingItem.value.next_run_date.slice(0, 10)
    : new Date().toISOString().slice(0, 10),
  note: editingItem.value?.note ?? '',
})

const form = reactive(initialForm())

function openCreateModal() {
  editingItem.value = null
  Object.assign(form, initialForm())
  submitError.value = ''
  isModalOpen.value = true
}

function openEditModal(item) {
  editingItem.value = item
  Object.assign(form, initialForm())
  submitError.value = ''
  isModalOpen.value = true
}

function closeModal() {
  if (isSubmitting.value) return
  isModalOpen.value = false
}

async function submitForm() {
  submitError.value = ''

  if (!form.amount || form.amount <= 0) {
    submitError.value = 'Le montant doit être supérieur à 0.'
    return
  }

  if (!form.firstRunDate) {
    submitError.value = 'La date de la première occurrence est obligatoire.'
    return
  }

  if (form.type === 'achat' && (!form.merchant.trim() || !form.categoryId)) {
    submitError.value = 'Le commerçant et la catégorie sont obligatoires pour un achat.'
    return
  }

  if (form.type === 'revenu' && !form.source.trim()) {
    submitError.value = 'La source est obligatoire pour un revenu.'
    return
  }

  const payload = {
    account_id: activeAccountId.value,
    type: form.type,
    merchant: form.merchant.trim(),
    source: form.source.trim(),
    category_id: form.categoryId || 0,
    payment_method: form.paymentMethod,
    category: form.category.trim(),
    sub_category: form.subCategory.trim(),
    amount: Number(form.amount),
    first_run_date: form.firstRunDate,
    note: form.note.trim(),
  }

  isSubmitting.value = true

  try {
    if (editingItem.value) {
      await updateRecurring(editingItem.value.id, payload)
    } else {
      await createRecurring(payload)
    }

    isModalOpen.value = false
    await loadRecurring()
  } catch (err) {
    submitError.value = err instanceof Error ? err.message : 'Impossible d’enregistrer la récurrence.'
  } finally {
    isSubmitting.value = false
  }
}

async function togglePause(item) {
  try {
    await updateRecurring(item.id, {
      account_id: item.account_id,
      type: item.type,
      merchant: item.merchant || '',
      source: item.source || '',
      category_id: item.category_id || 0,
      payment_method: item.payment_method || '',
      category: item.category || '',
      sub_category: item.sub_category || '',
      amount: item.amount,
      first_run_date: item.next_run_date.slice(0, 10),
      note: item.note || '',
      is_active: !item.is_active,
    })
    await loadRecurring()
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : 'Impossible de modifier la récurrence.'
  }
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
    await deleteRecurring(deleteTarget.value.id)
    deleteTarget.value = null
    await loadRecurring()
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : 'Impossible de supprimer la récurrence.'
  } finally {
    isDeleting.value = false
  }
}

onMounted(loadRecurring)
</script>

<template>
  <main class="dashboard-content recurring-view">
    <PageHero
      eyebrow="Automatisation"
      title="Récurrences"
      description="Loyer, abonnements, salaire — créés automatiquement chaque mois."
    >
      <template #actions>
        <div v-if="accountsList.length > 1" class="account-switcher">
          <select v-model.number="activeAccountId" class="account-switcher-select">
            <option v-for="account in accountsList" :key="account.id" :value="account.id">
              {{ account.name }}
            </option>
          </select>
        </div>

        <button class="primary-btn" type="button" @click="openCreateModal">+ Nouvelle récurrence</button>
      </template>
    </PageHero>

    <p v-if="loadError" class="form-error">{{ loadError }}</p>

    <section class="panel recurring-list-card">
      <p v-if="isLoading" class="recurring-empty">Chargement...</p>

      <p v-else-if="!sortedItems.length" class="recurring-empty">
        Aucune récurrence pour l’instant — clique sur « Nouvelle récurrence » pour en créer une.
      </p>

      <div v-else class="recurring-list">
        <article v-for="item in sortedItems" :key="item.id" class="recurring-row" :class="{ 'is-paused': !item.is_active }">
          <span class="recurring-type-badge" :class="`recurring-type-${item.type}`">
            {{ item.type === 'achat' ? 'Achat' : 'Revenu' }}
          </span>

          <div class="recurring-row-main">
            <strong>{{ item.type === 'achat' ? item.merchant : item.source }}</strong>
            <span class="recurring-row-meta">
              {{ item.type === 'achat' ? categoryName(item.category_id) : (item.category || '—') }}
              · le {{ item.day_of_month }} de chaque mois
              · prochaine occurrence le {{ formatNextRunDate(item.next_run_date) }}
              <span v-if="item.is_active && isOverdue(item)" class="recurring-overdue-badge">en retard</span>
            </span>
          </div>

          <strong class="recurring-row-amount">{{ formatCurrency(item.amount) }}</strong>

          <div class="recurring-row-actions">
            <button
              v-if="item.is_active && isOverdue(item)"
              class="ghost-btn"
              type="button"
              :disabled="executingId === item.id"
              @click="executeNow(item)"
            >
              {{ executingId === item.id ? 'Exécution...' : 'Exécuter maintenant' }}
            </button>
            <button class="ghost-btn" type="button" @click="togglePause(item)">
              {{ item.is_active ? 'Mettre en pause' : 'Reprendre' }}
            </button>
            <button class="ghost-btn" type="button" @click="openEditModal(item)">Modifier</button>
            <button class="ghost-btn ghost-btn-danger" type="button" @click="requestDelete(item)">
              Supprimer
            </button>
          </div>
        </article>
      </div>
    </section>

    <div v-if="isModalOpen" class="modal-overlay" @click.self="closeModal">
      <section class="modal-card">
        <div class="modal-header">
          <div>
            <p class="eyebrow">{{ editingItem ? 'Modifier la récurrence' : 'Nouvelle récurrence' }}</p>
            <h2>{{ editingItem ? 'Mettre à jour' : 'Créer une récurrence' }}</h2>
          </div>

          <button class="ghost-btn" type="button" @click="closeModal" :disabled="isSubmitting">Fermer</button>
        </div>

        <form class="recurring-form" @submit.prevent="submitForm">
          <div class="recurring-type-toggle">
            <button
              type="button"
              class="ghost-btn"
              :class="{ 'is-active': form.type === 'achat' }"
              @click="form.type = 'achat'"
            >
              Achat
            </button>
            <button
              type="button"
              class="ghost-btn"
              :class="{ 'is-active': form.type === 'revenu' }"
              @click="form.type = 'revenu'"
            >
              Revenu
            </button>
          </div>

          <template v-if="form.type === 'achat'">
            <label class="form-field">
              <span>Commerçant</span>
              <input v-model.trim="form.merchant" type="text" placeholder="Ex: EDF" />
            </label>

            <label class="form-field">
              <span>Catégorie</span>
              <select v-model.number="form.categoryId">
                <option :value="null">Choisir une catégorie</option>
                <option v-for="category in availableCategories" :key="category.id" :value="category.id">
                  {{ category.name }}
                </option>
              </select>
            </label>

            <label class="form-field">
              <span>Moyen de paiement</span>
              <select v-model="form.paymentMethod">
                <option value="Carte bancaire">Carte bancaire</option>
                <option value="Visa">Visa</option>
                <option value="Mastercard">Mastercard</option>
                <option value="Espèces">Espèces</option>
                <option value="Virement">Virement</option>
                <option value="Prélèvement">Prélèvement</option>
              </select>
            </label>
          </template>

          <template v-else>
            <label class="form-field form-field-full">
              <span>Source</span>
              <input v-model.trim="form.source" type="text" placeholder="Ex: Salaire" />
            </label>
          </template>

          <label class="form-field">
            <span>Montant</span>
            <input v-model.number="form.amount" type="number" min="0.01" step="0.01" placeholder="0.00" />
          </label>

          <label class="form-field">
            <span>{{ editingItem ? 'Prochaine occurrence' : 'Première occurrence' }}</span>
            <input v-model="form.firstRunDate" type="date" />
          </label>

          <label class="form-field form-field-full">
            <span>Note</span>
            <textarea v-model.trim="form.note" rows="3" placeholder="Ajouter un commentaire" />
          </label>

          <p v-if="submitError" class="form-error">{{ submitError }}</p>

          <div class="modal-actions">
            <button class="ghost-btn" type="button" @click="closeModal" :disabled="isSubmitting">Annuler</button>
            <button class="primary-btn" type="submit" :disabled="isSubmitting">
              {{ isSubmitting ? 'Enregistrement...' : editingItem ? 'Enregistrer' : 'Créer' }}
            </button>
          </div>
        </form>
      </section>
    </div>

    <ConfirmModal
      :model-value="Boolean(deleteTarget)"
      title="Supprimer cette récurrence ?"
      :message="`Cette action est définitive. La récurrence « ${deleteTarget?.merchant || deleteTarget?.source || ''} » sera supprimée.`"
      confirm-label="Supprimer"
      :is-processing="isDeleting"
      @update:model-value="deleteTarget = null"
      @confirm="confirmDelete"
    />
  </main>
</template>

<style scoped>
.recurring-view {
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

.recurring-list-card {
  padding: 1.1rem;
}

.recurring-empty {
  margin: 0;
  color: var(--text-dim, #8a939d);
  font-size: 0.88rem;
}

.recurring-list {
  display: grid;
  gap: 0.6rem;
}

.recurring-row {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.85rem 1rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.03);
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
  flex-wrap: wrap;
}

.recurring-row.is-paused {
  opacity: 0.55;
}

.recurring-type-badge {
  flex-shrink: 0;
  padding: 0.3rem 0.7rem;
  border-radius: 999px;
  font-size: 0.76rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.recurring-type-achat {
  background: rgba(240, 169, 94, 0.16);
  color: #f0a95e;
}

.recurring-type-revenu {
  background: rgba(94, 203, 143, 0.16);
  color: #5ecb8f;
}

.recurring-row-main {
  display: grid;
  gap: 0.2rem;
  min-width: 0;
  flex: 1;
}

.recurring-row-main strong {
  color: var(--text, #eef1f3);
  font-size: 0.92rem;
}

.recurring-row-meta {
  color: var(--text-dim, #8a939d);
  font-size: 0.78rem;
}

.recurring-overdue-badge {
  display: inline-block;
  margin-left: 0.4rem;
  padding: 0.1rem 0.5rem;
  border-radius: 999px;
  background: rgba(239, 68, 68, 0.16);
  color: var(--negative-text, #fca5a5);
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.02em;
}

.recurring-row-amount {
  flex-shrink: 0;
  color: var(--text, #eef1f3);
  font-size: 0.95rem;
}

.recurring-row-actions {
  display: flex;
  gap: 0.5rem;
  flex-shrink: 0;
}

.recurring-row-actions .ghost-btn {
  padding: 0.5rem 0.85rem;
  font-size: 0.82rem;
}

.ghost-btn-danger {
  color: var(--negative-text);
}

.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 80;
  display: grid;
  place-items: center;
  padding: 1.25rem;
  background: var(--modal-overlay);
  backdrop-filter: blur(6px);
}

.modal-card {
  width: min(100%, 480px);
  border-radius: 24px;
  padding: 1.4rem;
  background: var(--modal-bg);
  border: 1px solid rgba(var(--tint-rgb), 0.06);
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.34);
  max-height: 90vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  margin-bottom: 1.2rem;
}

.modal-header h2 {
  margin-top: 0.35rem;
  font-size: 1.4rem;
  line-height: 1.15;
  color: var(--text, #eef1f3);
}

.eyebrow {
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.74rem;
}

.recurring-form {
  display: grid;
  gap: 1rem;
}

.recurring-type-toggle {
  display: flex;
  gap: 0.5rem;
}

.recurring-type-toggle .ghost-btn.is-active {
  background: rgba(114, 137, 152, 0.18);
  color: var(--text, #eef1f3);
  border-color: rgba(114, 137, 152, 0.5);
}

.form-field {
  display: grid;
  gap: 0.45rem;
}

.form-field-full {
  grid-column: 1 / -1;
}

.form-field span {
  color: var(--text-soft, #b3bbc4);
  font-size: 0.92rem;
  font-weight: 600;
}

.form-field input,
.form-field select,
.form-field textarea {
  width: 100%;
  border: 1px solid rgba(var(--tint-rgb), 0.07);
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.035);
  color: var(--text, #eef1f3);
  padding: 0.9rem 1rem;
  outline: none;
  font-family: inherit;
  font-size: 0.92rem;
  transition: border-color 140ms ease, background 140ms ease;
}

.form-field input:focus,
.form-field select:focus,
.form-field textarea:focus {
  border-color: rgba(219, 230, 223, 0.35);
  background: rgba(var(--tint-rgb), 0.05);
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

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.4rem;
}
</style>
