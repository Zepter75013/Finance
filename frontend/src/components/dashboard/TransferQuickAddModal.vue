<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { usePurchasesStore } from '../../stores/purchases'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  // Présent = édition d'un virement existant (montant/date/note/compte
  // lié restent modifiables) ; absent = création d'un nouveau mouvement.
  transfer: {
    type: Object,
    default: null,
  },
})

const emit = defineEmits(['update:modelValue'])

const store = usePurchasesStore()
const { accountsList, activeAccountId } = storeToRefs(store)

const isEditMode = computed(() => Boolean(props.transfer?.id))

const otherAccountsList = computed(() =>
  accountsList.value.filter((account) => Number(account.id) !== Number(activeAccountId.value))
)

function initialForm() {
  if (props.transfer) {
    const isOutgoing = Number(props.transfer.fromAccountId) === Number(activeAccountId.value)

    return {
      direction: isOutgoing ? 'debit' : 'credit',
      otherAccountId: isOutgoing ? props.transfer.toAccountId : props.transfer.fromAccountId,
      amount: props.transfer.amount,
      date: props.transfer.date,
      note: props.transfer.note || '',
    }
  }

  return {
    direction: 'credit', // 'credit' = reçu depuis l'autre compte, 'debit' = envoyé vers l'autre compte
    otherAccountId: otherAccountsList.value[0]?.id ?? null,
    amount: null,
    date: new Date().toISOString().slice(0, 10),
    note: '',
  }
}

const form = reactive(initialForm())
const isSubmitting = ref(false)
const submitError = ref('')

watch(
  () => [props.modelValue, props.transfer],
  ([isOpen]) => {
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

  if (!form.otherAccountId) {
    submitError.value = 'Choisis le compte concerné.'
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

  const isCredit = form.direction === 'credit'
  const fromAccountId = isCredit ? form.otherAccountId : activeAccountId.value
  const toAccountId = isCredit ? activeAccountId.value : form.otherAccountId

  try {
    if (isEditMode.value) {
      // Ne garde que from/to/montant/date/note du formulaire — le reste
      // (pointage, copie d'origine…) vient de props.transfer pour ne pas
      // l'effacer silencieusement à l'édition.
      await store.editTransfer(props.transfer.id, {
        ...props.transfer,
        fromAccountId,
        toAccountId,
        amount: form.amount,
        date: form.date,
        note: form.note,
      })
    } else {
      await store.createTransfer({ fromAccountId, toAccountId, amount: form.amount, date: form.date, note: form.note })
    }

    emit('update:modelValue', false)
  } catch (err) {
    submitError.value =
      err instanceof Error ? err.message : isEditMode.value ? 'Impossible de modifier ce virement.' : 'Impossible de créer ce mouvement.'
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
          <p class="eyebrow">Mouvement</p>
          <h2>{{ isEditMode ? 'Modifier ce virement' : 'Ajouter un crédit ou un débit' }}</h2>
        </div>

        <button class="ghost-btn" type="button" @click="closeModal" :disabled="isSubmitting">Fermer</button>
      </div>

      <form class="quick-add-form" @submit.prevent="submitForm">
        <div class="direction-tabs">
          <button
            type="button"
            class="direction-tab direction-tab-credit"
            :class="{ 'is-active': form.direction === 'credit' }"
            @click="form.direction = 'credit'"
          >
            Crédit reçu
          </button>
          <button
            type="button"
            class="direction-tab direction-tab-debit"
            :class="{ 'is-active': form.direction === 'debit' }"
            @click="form.direction = 'debit'"
          >
            Débit envoyé
          </button>
        </div>

        <label class="form-field form-field-full">
          <span>{{ form.direction === 'credit' ? 'Compte source' : 'Compte destination' }}</span>
          <select
            v-model.number="form.otherAccountId"
            :title="otherAccountsList.find((a) => Number(a.id) === Number(form.otherAccountId))?.name || ''"
          >
            <option :value="null">Choisir un compte</option>
            <option v-for="account in otherAccountsList" :key="account.id" :value="account.id" :title="account.name">
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
            {{ isSubmitting ? 'Enregistrement...' : isEditMode ? 'Enregistrer les modifications' : 'Ajouter' }}
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
  width: min(100%, 640px);
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
  margin-bottom: 1.1rem;
}

.modal-header h2 {
  margin-top: 0.35rem;
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

.quick-add-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
}

.direction-tabs {
  grid-column: 1 / -1;
  display: flex;
  gap: 0.5rem;
}

.direction-tab {
  flex: 1;
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  border-radius: 14px;
  padding: 0.7rem;
  background: rgba(var(--tint-rgb), 0.035);
  color: var(--text-soft, #b3bbc4);
  font-weight: 700;
  font-size: 0.88rem;
  cursor: pointer;
  transition: background 140ms ease, border-color 140ms ease, color 140ms ease;
}

.direction-tab-credit.is-active {
  background: rgba(94, 203, 143, 0.16);
  border-color: rgba(94, 203, 143, 0.4);
  color: var(--positive-text, #bfe0c9);
}

.direction-tab-debit.is-active {
  background: rgba(225, 120, 106, 0.14);
  border-color: rgba(225, 120, 106, 0.4);
  color: var(--negative-text, #e1b4b4);
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
  font-size: 0.88rem;
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
  padding: 0.85rem 1rem;
  outline: none;
  font-family: inherit;
  font-size: 0.9rem;
  transition: border-color 140ms ease, background 140ms ease;
}

.form-field select {
  text-overflow: ellipsis;
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
  font-size: 0.88rem;
  grid-column: 1 / -1;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.2rem;
  grid-column: 1 / -1;
}

.ghost-btn {
  border: none;
  border-radius: 10px;
  padding: 0.55rem 0.9rem;
  background: rgba(var(--tint-rgb), 0.06);
  color: var(--text, #eef1f3);
  font-size: 0.86rem;
  font-weight: 600;
  cursor: pointer;
}
</style>
