<script setup>
import { reactive, ref, watch } from 'vue'
import { useAuthStore } from '../../stores/auth'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['update:modelValue'])

const authStore = useAuthStore()

const form = reactive({
  currentPassword: '',
  newPassword: '',
  confirmPassword: '',
})

const submitError = ref('')
const isSubmitting = ref(false)
const successMessage = ref('')

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      form.currentPassword = ''
      form.newPassword = ''
      form.confirmPassword = ''
      submitError.value = ''
      successMessage.value = ''
    }
  }
)

function closeModal() {
  if (isSubmitting.value) return
  emit('update:modelValue', false)
}

async function submitForm() {
  submitError.value = ''
  successMessage.value = ''

  if (!form.currentPassword || !form.newPassword) {
    submitError.value = 'Tous les champs sont obligatoires.'
    return
  }

  if (form.newPassword.length < 4) {
    submitError.value = 'Le nouveau mot de passe doit contenir au moins 4 caractères.'
    return
  }

  if (form.newPassword !== form.confirmPassword) {
    submitError.value = 'Les deux mots de passe ne correspondent pas.'
    return
  }

  isSubmitting.value = true

  try {
    await authStore.changePassword(form.currentPassword, form.newPassword)
    successMessage.value = 'Mot de passe mis à jour.'
    form.currentPassword = ''
    form.newPassword = ''
    form.confirmPassword = ''
  } catch (err) {
    submitError.value = err instanceof Error ? err.message : 'Impossible de modifier le mot de passe.'
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
          <p class="eyebrow">Sécurité</p>
          <h2>Modifier le mot de passe</h2>
        </div>

        <button class="ghost-btn" type="button" @click="closeModal" :disabled="isSubmitting">
          Fermer
        </button>
      </div>

      <form class="password-form" @submit.prevent="submitForm">
        <label class="form-field">
          <span>Mot de passe actuel</span>
          <input
            v-model="form.currentPassword"
            type="password"
            autocomplete="current-password"
            placeholder="••••••••"
          />
        </label>

        <label class="form-field">
          <span>Nouveau mot de passe</span>
          <input
            v-model="form.newPassword"
            type="password"
            autocomplete="new-password"
            placeholder="••••••••"
          />
        </label>

        <label class="form-field">
          <span>Confirmer le nouveau mot de passe</span>
          <input
            v-model="form.confirmPassword"
            type="password"
            autocomplete="new-password"
            placeholder="••••••••"
          />
        </label>

        <p v-if="submitError" class="form-error">{{ submitError }}</p>
        <p v-if="successMessage" class="form-success">{{ successMessage }}</p>

        <div class="modal-actions">
          <button class="ghost-btn" type="button" @click="closeModal" :disabled="isSubmitting">
            Annuler
          </button>

          <button class="primary-btn" type="submit" :disabled="isSubmitting">
            {{ isSubmitting ? 'Enregistrement...' : 'Mettre à jour' }}
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
  width: min(100%, 460px);
  border-radius: 24px;
  padding: 1.4rem;
  background: var(--modal-bg);
  border: 1px solid rgba(var(--tint-rgb), 0.06);
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.34);
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

.password-form {
  display: grid;
  gap: 1rem;
}

.form-field {
  display: grid;
  gap: 0.45rem;
}

.form-field span {
  color: var(--text-soft, #b3bbc4);
  font-size: 0.92rem;
  font-weight: 600;
}

.form-field input {
  width: 100%;
  border: 1px solid rgba(var(--tint-rgb), 0.07);
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.035);
  color: var(--text, #eef1f3);
  padding: 0.9rem 1rem;
  outline: none;
  transition: border-color 140ms ease, background 140ms ease;
}

.form-field input::placeholder {
  color: rgba(var(--tint-rgb), 0.34);
}

.form-field input:focus {
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

.form-success {
  margin: 0;
  padding: 0.85rem 1rem;
  border-radius: 14px;
  background: rgba(143, 168, 160, 0.14);
  color: var(--positive-text);
  border: 1px solid rgba(143, 168, 160, 0.24);
  font-size: 0.92rem;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.4rem;
}
</style>
