<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { createUser, updateUser } from '../../services/users'
import { resizeAvatarFile } from '../../utils/avatar'
import { usePurchasesStore } from '../../stores/purchases'

const { accountsList } = storeToRefs(usePurchasesStore())

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  user: {
    type: Object,
    default: null,
  },
})

const emit = defineEmits(['update:modelValue', 'saved'])

const isEditMode = computed(() => Boolean(props.user?.id))

const initialForm = () => ({
  firstName: props.user?.first_name ?? '',
  lastName: props.user?.last_name ?? '',
  username: props.user?.username ?? '',
  avatarUrl: props.user?.avatar_url ?? '',
  password: '',
  confirmPassword: '',
  isAccountRestricted: Array.isArray(props.user?.account_ids),
  accountIds: Array.isArray(props.user?.account_ids) ? [...props.user.account_ids] : [],
})

const form = reactive(initialForm())

const submitError = ref('')
const isSubmitting = ref(false)
const isProcessingPhoto = ref(false)
const photoError = ref('')
const fileInput = ref(null)

function avatarInitial() {
  const source = form.firstName || form.username
  return source?.trim().charAt(0).toUpperCase() || '?'
}

function triggerFilePicker() {
  fileInput.value?.click()
}

async function handlePhotoChange(event) {
  const file = event.target.files?.[0]
  event.target.value = ''

  if (!file) return

  photoError.value = ''
  isProcessingPhoto.value = true

  try {
    form.avatarUrl = await resizeAvatarFile(file)
  } catch (err) {
    photoError.value = err instanceof Error ? err.message : 'Impossible de traiter cette image.'
  } finally {
    isProcessingPhoto.value = false
  }
}

function removePhoto() {
  form.avatarUrl = ''
  photoError.value = ''
}

function toggleAccount(accountId) {
  const index = form.accountIds.indexOf(accountId)
  if (index === -1) {
    form.accountIds.push(accountId)
  } else {
    form.accountIds.splice(index, 1)
  }
}

watch(
  () => [props.modelValue, props.user],
  ([isOpen]) => {
    if (isOpen) {
      Object.assign(form, initialForm())
      submitError.value = ''
      photoError.value = ''
    }
  },
  { deep: true, immediate: true }
)

function closeModal() {
  if (isSubmitting.value) return
  emit('update:modelValue', false)
}

async function submitForm() {
  submitError.value = ''

  const firstName = form.firstName.trim()
  const lastName = form.lastName.trim()
  const username = form.username.trim()

  if (!firstName || !lastName || !username) {
    submitError.value = 'Tous les champs sont obligatoires.'
    return
  }

  if (!isEditMode.value) {
    if (!form.password) {
      submitError.value = 'Le mot de passe est obligatoire.'
      return
    }

    if (form.password.length < 4) {
      submitError.value = 'Le mot de passe doit contenir au moins 4 caractères.'
      return
    }

    if (form.password !== form.confirmPassword) {
      submitError.value = 'Les deux mots de passe ne correspondent pas.'
      return
    }
  }

  isSubmitting.value = true

  const accountIds = form.isAccountRestricted ? [...form.accountIds] : null

  try {
    let saved

    if (isEditMode.value) {
      saved = await updateUser(props.user.id, {
        username,
        first_name: firstName,
        last_name: lastName,
        avatar_url: form.avatarUrl,
        account_ids: accountIds,
      })
    } else {
      saved = await createUser({
        username,
        first_name: firstName,
        last_name: lastName,
        avatar_url: form.avatarUrl,
        password: form.password,
        account_ids: accountIds,
      })
    }

    emit('saved', { type: isEditMode.value ? 'edit' : 'create', user: saved })
    emit('update:modelValue', false)
  } catch (err) {
    submitError.value = err instanceof Error ? err.message : 'Impossible d’enregistrer l’utilisateur.'
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
          <h2>{{ isEditMode ? 'Modifier l’utilisateur' : 'Nouvel utilisateur' }}</h2>
        </div>

        <button class="ghost-btn" type="button" @click="closeModal" :disabled="isSubmitting">
          Fermer
        </button>
      </div>

      <form class="user-form" @submit.prevent="submitForm">
        <div class="avatar-field">
          <div class="avatar-preview" :class="{ 'is-empty': !form.avatarUrl }">
            <img v-if="form.avatarUrl" :src="form.avatarUrl" alt="" />
            <span v-else>{{ avatarInitial() }}</span>
          </div>

          <div class="avatar-field-actions">
            <input
              ref="fileInput"
              type="file"
              accept="image/*"
              class="avatar-file-input"
              @change="handlePhotoChange"
            />
            <button
              class="ghost-btn"
              type="button"
              :disabled="isProcessingPhoto"
              @click="triggerFilePicker"
            >
              {{ isProcessingPhoto ? 'Chargement...' : form.avatarUrl ? 'Changer la photo' : 'Ajouter une photo' }}
            </button>
            <button
              v-if="form.avatarUrl"
              class="ghost-btn ghost-btn-danger"
              type="button"
              @click="removePhoto"
            >
              Retirer
            </button>
          </div>

          <p v-if="photoError" class="form-error avatar-error">{{ photoError }}</p>
        </div>

        <div class="user-form-row">
          <label class="form-field">
            <span>Prénom</span>
            <input
              v-model.trim="form.firstName"
              type="text"
              placeholder="Prénom"
              autocomplete="off"
            />
          </label>

          <label class="form-field">
            <span>Nom</span>
            <input
              v-model.trim="form.lastName"
              type="text"
              placeholder="Nom"
              autocomplete="off"
            />
          </label>
        </div>

        <label class="form-field">
          <span>Email</span>
          <input
            v-model.trim="form.username"
            type="text"
            placeholder="ex: prenom@email.com"
            autocomplete="off"
          />
        </label>

        <div class="accounts-field">
          <label class="accounts-toggle">
            <input type="checkbox" v-model="form.isAccountRestricted" />
            <span>Restreindre l'accès à certains comptes</span>
          </label>

          <p class="accounts-hint">
            Non coché : cet utilisateur voit tous les comptes, y compris ceux créés plus tard.
          </p>

          <div v-if="form.isAccountRestricted" class="accounts-checklist">
            <label
              v-for="account in accountsList"
              :key="account.id"
              class="accounts-checklist-item"
            >
              <input
                type="checkbox"
                :checked="form.accountIds.includes(account.id)"
                @change="toggleAccount(account.id)"
              />
              <span>{{ account.name }}</span>
            </label>

            <p v-if="!accountsList.length" class="accounts-hint">Aucun compte disponible.</p>
          </div>
        </div>

        <template v-if="!isEditMode">
          <label class="form-field">
            <span>Mot de passe</span>
            <input
              v-model="form.password"
              type="password"
              autocomplete="new-password"
              placeholder="••••••••"
            />
          </label>

          <label class="form-field">
            <span>Confirmer le mot de passe</span>
            <input
              v-model="form.confirmPassword"
              type="password"
              autocomplete="new-password"
              placeholder="••••••••"
            />
          </label>
        </template>

        <p v-if="submitError" class="form-error">{{ submitError }}</p>

        <div class="modal-actions">
          <button class="ghost-btn" type="button" @click="closeModal" :disabled="isSubmitting">
            Annuler
          </button>

          <button class="primary-btn" type="submit" :disabled="isSubmitting">
            {{
              isSubmitting
                ? 'Enregistrement...'
                : isEditMode
                  ? 'Enregistrer les modifications'
                  : 'Créer l’utilisateur'
            }}
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

.user-form {
  display: grid;
  gap: 1rem;
}

.avatar-field {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}

.avatar-preview {
  flex-shrink: 0;
  width: 64px;
  height: 64px;
  border-radius: 18px;
  overflow: hidden;
  display: grid;
  place-items: center;
  background: linear-gradient(135deg, var(--accent, #728998), var(--accent-soft, #819b8d));
  color: #ffffff;
  font-weight: 700;
  font-size: 1.3rem;
}

.avatar-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-preview.is-empty {
  background: rgba(var(--tint-rgb), 0.08);
  color: var(--text-dim, #8a939d);
}

.avatar-field-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.avatar-file-input {
  display: none;
}

.ghost-btn-danger {
  color: var(--negative-text);
}

.avatar-error {
  flex-basis: 100%;
  margin: 0;
}

.user-form-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
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

.accounts-field {
  display: grid;
  gap: 0.5rem;
}

.accounts-toggle {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.92rem;
  font-weight: 600;
  cursor: pointer;
}

.accounts-hint {
  margin: 0;
  color: var(--text-dim, #8a939d);
  font-size: 0.82rem;
}

.accounts-checklist {
  display: grid;
  gap: 0.5rem;
  padding: 0.85rem 1rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.035);
  border: 1px solid rgba(var(--tint-rgb), 0.07);
}

.accounts-checklist-item {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  color: var(--text, #eef1f3);
  font-size: 0.92rem;
  cursor: pointer;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.4rem;
}
</style>
