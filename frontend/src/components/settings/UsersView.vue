<script setup>
import { computed, onMounted, ref } from 'vue'
import PageHero from '../Common/PageHero.vue'
import ConfirmModal from '../Common/ConfirmModal.vue'
import UserFormModal from './UserFormModal.vue'
import { useAuthStore } from '../../stores/auth'
import { fetchUsers, deleteUser } from '../../services/users'

const authStore = useAuthStore()

const users = ref([])
const isLoading = ref(false)
const loadError = ref('')

const isFormModalOpen = ref(false)
const editingUser = ref(null)
const deleteTarget = ref(null)
const isDeleting = ref(false)
const deleteError = ref('')

async function loadUsers() {
  isLoading.value = true
  loadError.value = ''

  try {
    users.value = await fetchUsers()
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : 'Erreur inconnue.'
  } finally {
    isLoading.value = false
  }
}

onMounted(loadUsers)

function formatDate(value) {
  if (!value) return ''

  return new Intl.DateTimeFormat('fr-FR', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  }).format(new Date(value))
}

function avatarLetter(account) {
  const source = account.first_name || account.username
  return source?.trim().charAt(0).toUpperCase() || '?'
}

function displayName(account) {
  const fullName = [account.first_name, account.last_name].filter(Boolean).join(' ')
  return fullName || account.username
}

const canDeleteLast = computed(() => users.value.length > 1)

function openCreateModal() {
  editingUser.value = null
  isFormModalOpen.value = true
}

function openEditModal(account) {
  editingUser.value = account
  isFormModalOpen.value = true
}

function handleSaved({ type, user: saved }) {
  if (type === 'edit') {
    users.value = users.value.map((account) => (account.id === saved.id ? saved : account))
  } else {
    users.value = [...users.value, saved]
  }

  users.value = [...users.value].sort((a, b) =>
    a.username.localeCompare(b.username, 'fr', { sensitivity: 'base' })
  )

  if (saved.id === authStore.id) {
    authStore.setAvatar(saved.avatar_url)
  }
}

function requestDelete(account) {
  deleteError.value = ''
  deleteTarget.value = account
}

async function confirmDelete() {
  if (!deleteTarget.value) return

  isDeleting.value = true
  deleteError.value = ''

  try {
    await deleteUser(deleteTarget.value.id)
    users.value = users.value.filter((account) => account.id !== deleteTarget.value.id)
    deleteTarget.value = null
  } catch (err) {
    deleteError.value = err instanceof Error ? err.message : 'Impossible de supprimer cet utilisateur.'
  } finally {
    isDeleting.value = false
  }
}
</script>

<template>
  <main class="dashboard-content users-view">
    <PageHero
      eyebrow="Sécurité"
      title="Utilisateurs"
      description="Définis les personnes qui ont accès à cette application."
    >
      <template v-if="authStore.isAdmin" #actions>
        <button class="primary-btn" type="button" @click="openCreateModal">
          + Nouvel utilisateur
        </button>
      </template>
    </PageHero>

    <section v-if="isLoading" class="panel users-state">
      Chargement des utilisateurs...
    </section>

    <section v-else-if="loadError" class="panel users-state users-state--error">
      {{ loadError }}
    </section>

    <section v-else class="panel users-list-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Accès</p>
          <h2>{{ users.length }} compte{{ users.length > 1 ? 's' : '' }}</h2>
        </div>
      </div>

      <p v-if="deleteError" class="form-error users-error">{{ deleteError }}</p>

      <div class="users-list">
        <article v-for="account in users" :key="account.id" class="user-row">
          <div class="user-row-main">
            <span class="user-avatar" :class="{ 'has-photo': account.avatar_url }">
              <img v-if="account.avatar_url" :src="account.avatar_url" alt="" />
              <template v-else>{{ avatarLetter(account) }}</template>
            </span>
            <div>
              <strong>{{ displayName(account) }}</strong>
              <p class="user-email">{{ account.username }}</p>
              <p class="user-meta">Créé le {{ formatDate(account.created_at) }}</p>
            </div>
          </div>

          <div class="user-row-actions">
            <button
              v-if="authStore.isAdmin || account.username === authStore.username"
              class="icon-btn"
              type="button"
              title="Modifier cet utilisateur"
              aria-label="Modifier cet utilisateur"
              @click="openEditModal(account)"
            >
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path
                  d="M4 20h4l10.5-10.5a1.414 1.414 0 0 0-4-4L4 16v4z"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.8"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
                <path
                  d="M13.5 6.5l4 4"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.8"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </button>

            <button
              v-if="authStore.isAdmin"
              class="icon-btn icon-btn-danger"
              type="button"
              :disabled="account.username === authStore.username || !canDeleteLast"
              :title="
                account.username === authStore.username
                  ? 'Tu ne peux pas supprimer ton propre compte'
                  : !canDeleteLast
                    ? 'Impossible de supprimer le dernier compte'
                    : 'Supprimer cet utilisateur'
              "
              @click="requestDelete(account)"
            >
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path
                  d="M5 7h14"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.8"
                  stroke-linecap="round"
                />
                <path
                  d="M10 11v6M14 11v6"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.8"
                  stroke-linecap="round"
                />
                <path
                  d="M8 7l1-2h6l1 2"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.8"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
                <path
                  d="M7 7v11a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2V7"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.8"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </button>
          </div>
        </article>
      </div>
    </section>

    <UserFormModal v-model="isFormModalOpen" :user="editingUser" @saved="handleSaved" />

    <ConfirmModal
      :model-value="Boolean(deleteTarget)"
      :title="`Supprimer ${deleteTarget?.username} ?`"
      :message="`${deleteTarget?.username} n'aura plus accès à cette application.`"
      note="Cette action est irréversible."
      confirm-label="Supprimer"
      :is-processing="isDeleting"
      @update:model-value="deleteTarget = null"
      @confirm="confirmDelete"
    />
  </main>
</template>

<style scoped>
.users-view {
  display: grid;
  gap: 0.9rem;
}

.users-state {
  padding: 1.8rem 1.3rem;
  text-align: center;
  color: var(--text-soft, #b3bbc4);
}

.users-state--error {
  color: var(--negative-text);
}

.users-list-card {
  padding: 0.95rem;
}

.users-error {
  margin: 0.75rem 0 0;
}

.users-list {
  display: grid;
  gap: 0.5rem;
  margin-top: 0.9rem;
}

.user-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.9rem;
  padding: 0.7rem 0.85rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.028);
  border: 1px solid rgba(var(--tint-rgb), 0.05);
}

.user-row-main {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  min-width: 0;
}

.user-avatar {
  flex-shrink: 0;
  width: 34px;
  height: 34px;
  border-radius: 10px;
  overflow: hidden;
  display: grid;
  place-items: center;
  background: linear-gradient(135deg, var(--accent, #728998), var(--accent-soft, #819b8d));
  color: #ffffff;
  font-weight: 700;
  font-size: 0.9rem;
}

.user-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.user-row strong {
  display: block;
  color: var(--text, #eef1f3);
  font-size: 0.92rem;
}

.user-row p {
  margin: 0.15rem 0 0;
  color: var(--text-dim, #8a939d);
  font-size: 0.76rem;
}

.user-row .user-email {
  color: var(--text-soft, #b3bbc4);
}

.user-row-actions {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  flex-shrink: 0;
}

.icon-btn {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  border-radius: 10px;
  display: grid;
  place-items: center;
  background: rgba(var(--tint-rgb), 0.035);
  color: var(--text);
  cursor: pointer;
  transition: transform 140ms ease, background 140ms ease, border-color 140ms ease;
}

.icon-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  background: rgba(var(--tint-rgb), 0.085);
}

.icon-btn svg {
  width: 15px;
  height: 15px;
}

.icon-btn-danger {
  color: var(--negative-text);
  background: rgba(220, 38, 38, 0.1);
  border-color: rgba(220, 38, 38, 0.14);
}

.icon-btn-danger:hover:not(:disabled) {
  background: rgba(220, 38, 38, 0.16);
  border-color: rgba(220, 38, 38, 0.22);
}

.icon-btn-danger:disabled {
  opacity: 0.4;
  cursor: not-allowed;
  pointer-events: none;
}

.eyebrow {
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.72rem;
}
</style>
