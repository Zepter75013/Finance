<script setup>
import { computed, ref, watch } from 'vue'
import { usePurchasesStore } from '../../stores/purchases'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  targetAccount: {
    type: Object,
    default: null,
  },
  accounts: {
    type: Array,
    default: () => [],
  },
})

const emit = defineEmits(['update:modelValue', 'copied'])

const store = usePurchasesStore()

// 'confirm' (oui/non) -> 'pick' (choix du compte source) -> 'result' (résumé).
// Le bouton qui ouvre cette modale reste toujours visible sur l'écran Comptes
// (il ne disparaît pas après usage) — l'action est donc rejouable à tout
// moment, par exemple pour rattraper de nouvelles catégories ajoutées ailleurs.
const step = ref('confirm')
const sourceAccountId = ref(null)
const isProcessing = ref(false)
const error = ref('')
const result = ref(null)

const otherAccounts = computed(() =>
  props.accounts.filter((account) => Number(account.id) !== Number(props.targetAccount?.id))
)

watch(
  () => props.modelValue,
  (isOpen) => {
    if (!isOpen) return

    step.value = 'confirm'
    sourceAccountId.value = otherAccounts.value[0]?.id ?? null
    isProcessing.value = false
    error.value = ''
    result.value = null
  }
)

function closeModal() {
  if (isProcessing.value) return
  emit('update:modelValue', false)
}

function answerYes() {
  step.value = 'pick'
}

function answerNo() {
  closeModal()
}

async function confirmCopy() {
  if (!sourceAccountId.value || !props.targetAccount?.id) return

  isProcessing.value = true
  error.value = ''

  try {
    const copyResult = await store.copyCategoriesFromAccount(sourceAccountId.value, props.targetAccount.id)
    result.value = copyResult
    step.value = 'result'
    emit('copied', copyResult)
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Impossible de copier les catégories.'
  } finally {
    isProcessing.value = false
  }
}
</script>

<template>
  <div v-if="modelValue" class="modal-overlay" @click.self="closeModal">
    <section class="modal-card">
      <div class="modal-header">
        <div>
          <p class="eyebrow">Catégories</p>
          <h2>Copier des catégories vers « {{ targetAccount?.name }} »</h2>
        </div>
      </div>

      <template v-if="step === 'confirm'">
        <p class="modal-text">
          Veux-tu copier les catégories (et sous-catégories) d'un autre compte vers
          <strong>{{ targetAccount?.name }}</strong> ?
        </p>

        <div class="modal-actions">
          <button class="ghost-btn" type="button" @click="answerNo">Non</button>
          <button class="primary-btn" type="button" :disabled="!otherAccounts.length" @click="answerYes">
            Oui
          </button>
        </div>

        <p v-if="!otherAccounts.length" class="modal-note">
          Aucun autre compte disponible pour l'instant.
        </p>
      </template>

      <template v-else-if="step === 'pick'">
        <label class="form-field">
          <span>Copier depuis</span>
          <select v-model.number="sourceAccountId">
            <option v-for="account in otherAccounts" :key="account.id" :value="account.id">
              {{ account.name }}
            </option>
          </select>
        </label>

        <p class="modal-note">
          Les catégories déjà présentes (même nom, même type) dans « {{ targetAccount?.name }} » sont
          réutilisées plutôt que dupliquées.
        </p>

        <p v-if="error" class="form-error">{{ error }}</p>

        <div class="modal-actions">
          <button class="ghost-btn" type="button" :disabled="isProcessing" @click="closeModal">
            Annuler
          </button>
          <button class="primary-btn" type="button" :disabled="isProcessing" @click="confirmCopy">
            {{ isProcessing ? 'Copie en cours...' : 'Copier' }}
          </button>
        </div>
      </template>

      <template v-else-if="step === 'result'">
        <p class="modal-text">
          {{ result?.createdCategories || 0 }} catégorie{{ (result?.createdCategories || 0) > 1 ? 's' : '' }} et
          {{ result?.createdSubCategories || 0 }} sous-catégorie{{ (result?.createdSubCategories || 0) > 1 ? 's' : '' }}
          copiée{{ (result?.createdCategories || 0) > 1 ? 's' : '' }} vers « {{ targetAccount?.name }} ».
        </p>

        <div class="modal-actions">
          <button class="primary-btn" type="button" @click="closeModal">Fermer</button>
        </div>
      </template>
    </section>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 80;
  display: grid;
  place-items: center;
  padding: 1.5rem;
  background: var(--modal-overlay);
  backdrop-filter: blur(6px);
}

.modal-card {
  width: min(100%, 460px);
  border-radius: 24px;
  padding: 1.35rem;
  background: var(--modal-bg);
  border: 1px solid rgba(var(--tint-rgb), 0.06);
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.34);
}

.modal-header {
  margin-bottom: 0.9rem;
}

.eyebrow {
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.74rem;
}

.modal-header h2 {
  margin-top: 0.35rem;
  font-size: 1.2rem;
  line-height: 1.25;
  color: var(--text, #eef1f3);
}

.modal-text {
  margin: 0 0 1.1rem;
  line-height: 1.6;
  color: var(--text-soft, #b3bbc4);
}

.modal-text strong {
  color: var(--text, #eef1f3);
}

.modal-note {
  margin: 0.7rem 0 1.1rem;
  padding: 0.75rem 0.9rem;
  border-radius: 12px;
  background: rgba(var(--tint-rgb), 0.04);
  border: 1px solid rgba(var(--tint-rgb), 0.05);
  color: var(--text-dim, #8a939d);
  font-size: 0.85rem;
  line-height: 1.5;
}

.form-field {
  display: grid;
  gap: 0.4rem;
  margin-bottom: 0.9rem;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.86rem;
  font-weight: 600;
}

.form-field select {
  height: 40px;
  border-radius: 10px;
  border: 1px solid rgba(var(--tint-rgb), 0.1);
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--text, #eef1f3);
  padding: 0 0.7rem;
  font-size: 0.9rem;
}

.form-error {
  margin: 0 0 0.9rem;
  padding: 0.7rem 0.85rem;
  border-radius: 12px;
  background: rgba(239, 68, 68, 0.12);
  color: var(--negative-text);
  border: 1px solid rgba(239, 68, 68, 0.18);
  font-size: 0.85rem;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}

.ghost-btn,
.primary-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 120px;
  border: none;
  border-radius: 14px;
  padding: 0.8rem 1.1rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 140ms ease, background 140ms ease, opacity 140ms ease;
}

.ghost-btn {
  background: rgba(var(--tint-rgb), 0.06);
  color: var(--text);
}

.primary-btn {
  background: #5cc8a0;
  color: #0d1210;
}

.ghost-btn:hover:not(:disabled),
.primary-btn:hover:not(:disabled) {
  transform: translateY(-1px);
}

.ghost-btn:disabled,
.primary-btn:disabled {
  cursor: not-allowed;
  opacity: 0.6;
  transform: none;
}

@media (max-width: 640px) {
  .modal-card {
    padding: 1.1rem;
    border-radius: 20px;
  }

  .modal-actions {
    flex-direction: column;
    align-items: stretch;
  }

  .ghost-btn,
  .primary-btn {
    width: 100%;
    min-width: 0;
  }
}
</style>
