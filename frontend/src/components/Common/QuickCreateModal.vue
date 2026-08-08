<template>
  <div v-if="modelValue" class="modal-overlay" @click.self="closeModal">
    <section class="modal-card quick-create-card" role="dialog" aria-modal="true">
      <div class="modal-header">
        <div>
          <p class="eyebrow">{{ eyebrow }}</p>
          <h2>{{ title }}</h2>
        </div>

        <button class="ghost-btn" type="button" @click="closeModal" :disabled="isSubmitting">
          Fermer
        </button>
      </div>

      <form class="quick-create-form" @submit.prevent="handleConfirm">
        <label class="form-field">
          <span>{{ label }}</span>
          <input
            ref="inputRef"
            v-model.trim="value"
            type="text"
            :placeholder="placeholder"
            :disabled="isSubmitting"
            autofocus
          />
        </label>

        <p v-if="errorMessage" class="form-error">{{ errorMessage }}</p>

        <div class="modal-actions">
          <button class="ghost-btn" type="button" @click="closeModal" :disabled="isSubmitting">
            Annuler
          </button>
          <button class="primary-btn" type="submit" :disabled="isSubmitting || !value.trim()">
            {{ isSubmitting ? 'Création...' : confirmLabel }}
          </button>
        </div>
      </form>
    </section>
  </div>
</template>

<script setup>
import { nextTick, ref, watch } from 'vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  eyebrow: {
    type: String,
    default: 'Création rapide',
  },
  title: {
    type: String,
    default: 'Nouvel élément',
  },
  label: {
    type: String,
    default: 'Nom',
  },
  placeholder: {
    type: String,
    default: '',
  },
  confirmLabel: {
    type: String,
    default: 'Créer',
  },
  initialValue: {
    type: String,
    default: '',
  },
  isSubmitting: {
    type: Boolean,
    default: false,
  },
  errorMessage: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['update:modelValue', 'confirm', 'cancel'])

const value = ref('')
const inputRef = ref(null)

watch(
  () => props.modelValue,
  (isOpen) => {
    if (isOpen) {
      value.value = props.initialValue || ''
      nextTick(() => inputRef.value?.focus())
    }
  },
  { immediate: true }
)

function closeModal() {
  if (props.isSubmitting) return
  emit('update:modelValue', false)
  emit('cancel')
}

function handleConfirm() {
  const trimmed = value.value.trim()
  if (!trimmed || props.isSubmitting) return
  emit('confirm', trimmed)
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

.quick-create-card {
  width: min(100%, 420px);
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
  margin-bottom: 1.1rem;
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

.quick-create-form {
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
  transition: border-color 140ms ease, background 140ms ease;
}

.form-field input::placeholder {
  color: var(--text-dim, #8a939d);
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

@media (max-width: 480px) {
  .quick-create-card {
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
