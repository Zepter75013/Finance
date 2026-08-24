<template>
  <div v-if="modelValue" class="modal-overlay" @click.self="closeModal">
    <section class="modal-card" role="dialog" aria-modal="true">
      <div class="modal-header">
        <div>
          <p class="eyebrow">Solde initial</p>
          <h2>{{ account?.name }}</h2>
        </div>

        <button class="ghost-btn" type="button" @click="closeModal" :disabled="isSubmitting">
          Fermer
        </button>
      </div>

      <p class="opening-balance-hint">
        Le solde affiché deviendra ce montant + les mouvements enregistrés depuis cette date — utile pour un compte
        sans relevé (ex: un livret), sans avoir à ressaisir tout l'historique. Les mouvements antérieurs à cette
        date ne seront pas comptés.
      </p>

      <form class="opening-balance-form" @submit.prevent="handleConfirm">
        <label class="form-field">
          <span>Montant ({{ currency }})</span>
          <input
            ref="amountInputRef"
            v-model="amount"
            type="text"
            inputmode="decimal"
            placeholder="Ex: 8000"
            :disabled="isSubmitting"
          />
        </label>

        <label class="form-field">
          <span>À cette date</span>
          <input v-model="date" type="date" :disabled="isSubmitting" />
        </label>

        <p v-if="errorMessage" class="form-error">{{ errorMessage }}</p>

        <div class="modal-actions">
          <button
            v-if="hasExistingOpeningBalance"
            class="ghost-btn ghost-btn-danger"
            type="button"
            :disabled="isSubmitting"
            @click="handleClear"
          >
            Effacer le solde initial
          </button>
          <button class="ghost-btn" type="button" @click="closeModal" :disabled="isSubmitting">
            Annuler
          </button>
          <button class="primary-btn" type="submit" :disabled="isSubmitting || !amount.trim() || !date">
            {{ isSubmitting ? 'Enregistrement...' : 'Enregistrer' }}
          </button>
        </div>
      </form>
    </section>
  </div>
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { usePurchasesStore } from '../../stores/purchases'
import { currencySymbol } from '../../utils/format'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  account: {
    type: Object,
    default: null,
  },
})

const emit = defineEmits(['update:modelValue'])

const store = usePurchasesStore()

const amount = ref('')
const date = ref('')
const amountInputRef = ref(null)
const isSubmitting = ref(false)
const errorMessage = ref('')

const hasExistingOpeningBalance = computed(() => props.account?.openingBalanceAmount != null)
const currency = computed(() => currencySymbol())

watch(
  () => props.modelValue,
  (isOpen) => {
    if (isOpen) {
      amount.value = props.account?.openingBalanceAmount != null ? String(props.account.openingBalanceAmount) : ''
      date.value = props.account?.openingBalanceDate || ''
      errorMessage.value = ''
      nextTick(() => amountInputRef.value?.focus())
    }
  },
  { immediate: true }
)

function closeModal() {
  if (isSubmitting.value) return
  emit('update:modelValue', false)
}

async function handleConfirm() {
  errorMessage.value = ''

  const normalized = amount.value.replace(',', '.').trim()
  const parsedAmount = Number(normalized)

  if (!normalized || !Number.isFinite(parsedAmount)) {
    errorMessage.value = 'Le montant doit être un nombre.'
    return
  }

  if (!date.value) {
    errorMessage.value = 'La date est obligatoire.'
    return
  }

  isSubmitting.value = true

  try {
    await store.setAccountOpeningBalance(props.account.id, parsedAmount, date.value)
    emit('update:modelValue', false)
  } catch (err) {
    errorMessage.value = err instanceof Error ? err.message : 'Impossible d’enregistrer le solde initial.'
  } finally {
    isSubmitting.value = false
  }
}

async function handleClear() {
  isSubmitting.value = true
  errorMessage.value = ''

  try {
    await store.clearAccountOpeningBalance(props.account.id)
    emit('update:modelValue', false)
  } catch (err) {
    errorMessage.value = err instanceof Error ? err.message : 'Impossible d’effacer le solde initial.'
  } finally {
    isSubmitting.value = false
  }
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 90;
  display: grid;
  place-items: center;
  padding: 1.25rem;
  background: var(--modal-overlay);
  backdrop-filter: blur(6px);
}

.modal-card {
  width: min(100%, 440px);
  border-radius: 24px;
  padding: 1.35rem;
  background: var(--modal-bg);
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.06));
  box-shadow: var(--shadow, 0 24px 60px rgba(0, 0, 0, 0.34));
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  margin-bottom: 0.9rem;
}

.modal-header h2 {
  margin-top: 0.3rem;
  font-size: 1.3rem;
  line-height: 1.15;
  color: var(--text, #eef1f3);
}

.eyebrow {
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.74rem;
}

.opening-balance-hint {
  margin: 0 0 1.1rem;
  color: var(--text-dim, #8a939d);
  font-size: 0.82rem;
  line-height: 1.4;
}

.opening-balance-form {
  display: grid;
  gap: 1rem;
}

.form-field {
  display: grid;
  gap: 0.45rem;
}

.form-field span {
  color: var(--text-soft, #b3bbc4);
  font-size: 0.9rem;
  font-weight: 600;
}

.form-field input {
  width: 100%;
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--text, #eef1f3);
  padding: 0.85rem 1rem;
  outline: none;
  font-family: inherit;
  transition: border-color 140ms ease, background 140ms ease;
}

.form-field input:focus {
  border-color: rgba(114, 137, 152, 0.55);
  box-shadow: 0 0 0 3px rgba(114, 137, 152, 0.12);
}

.form-error {
  margin: 0;
  padding: 0.75rem 0.9rem;
  border-radius: 14px;
  background: rgba(239, 68, 68, 0.12);
  color: var(--negative-text, #f3b1b1);
  border: 1px solid rgba(239, 68, 68, 0.18);
  font-size: 0.88rem;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.7rem;
  margin-top: 0.2rem;
  flex-wrap: wrap;
}

.primary-btn,
.ghost-btn {
  border: none;
  border-radius: 14px;
  padding: 0.85rem 1.05rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 140ms ease, opacity 140ms ease, background 140ms ease;
}

.primary-btn:hover,
.ghost-btn:hover {
  transform: translateY(-1px);
}

.primary-btn:disabled,
.ghost-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
  transform: none;
}

.primary-btn {
  background: #dbe6df;
  color: #1c2421;
}

.ghost-btn {
  background: rgba(var(--tint-rgb), 0.06);
  color: var(--text, #eef1f3);
}

.ghost-btn-danger {
  color: var(--negative-text);
}

@media (max-width: 480px) {
  .modal-card {
    padding: 1.1rem;
    border-radius: 20px;
  }

  .modal-header,
  .modal-actions {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
