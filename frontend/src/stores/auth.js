import { ref } from 'vue'
import { defineStore } from 'pinia'
import {
  login as loginApi,
  logout as logoutApi,
  fetchCurrentUser,
  changePassword as changePasswordApi,
} from '../services/auth'

export const useAuthStore = defineStore('auth', () => {
  const id = ref(null)
  const username = ref('')
  const avatarUrl = ref('')
  const isAuthenticated = ref(false)
  const isCheckingSession = ref(true)

  function applyAccount(account) {
    id.value = account?.id ?? null
    username.value = account?.username ?? ''
    avatarUrl.value = account?.avatar_url ?? ''
  }

  async function checkSession() {
    isCheckingSession.value = true

    try {
      const account = await fetchCurrentUser()

      if (account?.username) {
        applyAccount(account)
        isAuthenticated.value = true
      } else {
        applyAccount(null)
        isAuthenticated.value = false
      }
    } finally {
      isCheckingSession.value = false
    }
  }

  async function login(usernameInput, password) {
    const account = await loginApi(usernameInput, password)
    applyAccount(account)
    isAuthenticated.value = true
  }

  async function logout() {
    try {
      await logoutApi()
    } finally {
      applyAccount(null)
      isAuthenticated.value = false
    }
  }

  async function changePassword(currentPassword, newPassword) {
    await changePasswordApi(currentPassword, newPassword)
  }

  // Updates the locally cached avatar without a round-trip to the server —
  // used right after the current user edits their own profile in Utilisateurs
  // so the sidebar reflects it instantly.
  function setAvatar(url) {
    avatarUrl.value = url ?? ''
  }

  return {
    id,
    username,
    avatarUrl,
    isAuthenticated,
    isCheckingSession,
    checkSession,
    login,
    logout,
    changePassword,
    setAvatar,
  }
})
