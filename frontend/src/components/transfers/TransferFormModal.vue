<script setup>
import { reactive, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { usePurchasesStore } from '../../stores/purchases'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  item: {
    type: Object,
    default: null,
  },
  // Pré-remplit le compte source à l'ouverture pour une création — utilisé
  // par le bouton « Transférer » de l'écran Comptes (le compte source est
  // déjà sous les yeux, pas besoin de le resélectionner).
  defaultFromAccountId: {
    type: [Number, String],
    default: null,
  },
})

const emit = defineEmits(['update:modelValue', 'saved'])

const store = usePurchasesStore()
const { accountsList } = storeToRefs(store)

const isSubmitting = ref(false)
const submitError = ref('')

function initialForm() {
  return {
    fromAccountId: props.item?.fromAccountId ?? props.defaultFromAccountId ?? accountsList.value[0]?.id ?? null,
    toAccountId:
      props.item?.toAccountId ??
      accountsList.value.find((a) => Number(a.id) !== Number(props.defaultFromAccountId))?.id ??
      null,
    amount: props.item?.amount ?? null,
    date: props.item?.date ?? new Date().toISOString().slice(0, 10),
    note: props.item?.note ?? '',
  }
}

const form = reactive(initialForm())

watch(
  () => props.modelValue,
  (isOpen) => {
    if (isOpen) {
      Object.assign(form, initialForm())
      submitError.value = ''
    }
  }
)

function closeModal() {
  if (isSubmitting.value) return
  emit('update:modelValue', false)
}

async function submitForm() {
  submitError.value = ''

  if (!form.fromAccountId || !form.toAccountId) {
    submitError.value = 'Choisis un compte source et un compte destination.'
    return
  }

  if (Number(form.fromAccountId) === Number(form.toAccountId)) {
    submitError.value = 'Le compte source et le compte destination doivent être différents.'
    return
  }

  if (!form.amount || form.amount <= 0) {
    submitError.value = 'Le montant doit être supérieur à 0.'
    return
  }

  if (!form.date) {
    submitError.value = 'La date est obligatoire.'
    return
  }

  isSubmitting.value = true

  try {
    let saved

    if (props.item) {
      saved = await store.editTransfer(props.item.id, { ...props.item, ...form })
    } else {
      saved = await store.createTransfer(form)
    }

    emit('saved', saved)
    emit('update:modelValue', false)
  } catch (err) {
    submitError.value = err instanceof Error ? err.message : 'Impossible d’enregistrer le virement.'
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <div v-if="modelValue" class="modal-overlay" @click.self="closeModal">
    <section class="modal-card">
      <div class="modal-header">
        <div>
          <p class="eyebrow">{{ item ? 'Modifier le virement' : 'Nouveau virement' }}</p>
          <h2>{{ item ? 'Mettre à jour' : 'Créer un virement' }}</h2>
        </div>

        <button class="ghost-btn" type="button" @click="closeModal" :disabled="isSubmitting">Fermer</button>
      </div>

      <form class="transfer-form" @submit.prevent="submitForm">
        <label class="form-field">
          <span>Compte source</span>
          <select v-model.number="form.fromAccountId">
            <option :value="null">Choisir un compte</option>
            <option v-for="account in accountsList" :key="account.id" :value="account.id">
              {{ account.name }}
            </option>
          </select>
        </label>

        <label class="form-field">
          <span>Compte destination</span>
          <select v-model.number="form.toAccountId">
            <option :value="null">Choisir un compte</option>
            <option v-for="account in accountsList" :key="account.id" :value="account.id">
              {{ account.name }}
            </option>
          </select>
        </label>

        <label class="form-field">
          <span>Montant</span>
          <input v-model.number="form.amount" type="number" min="0.01" step="0.01" placeholder="0.00" />
        </label>

        <label class="form-field">
          <span>Date</span>
          <input v-model="form.date" type="date" />
        </label>

        <label class="form-field form-field-full">
          <span>Note</span>
          <textarea v-model.trim="form.note" rows="3" placeholder="Ajouter un commentaire" />
        </label>

        <p v-if="submitError" class="form-error">{{ submitError }}</p>

        <div class="modal-actions">
          <button class="ghost-btn" type="button" @click="closeModal" :disabled="isSubmitting">Annuler</button>
          <button class="primary-btn" type="submit" :disabled="isSubmitting">
            {{ isSubmitting ? 'Enregistrement...' : item ? 'Enregistrer' : 'Créer' }}
          </button>
        </div>
      </form>
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

.transfer-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
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
  grid-column: 1 / -1;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.4rem;
  grid-column: 1 / -1;
}
</style>
