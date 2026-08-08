<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useAuthStore } from '../../stores/auth'
import { requestResetCode, resetPasswordWithCode } from '../../services/auth'
import { APP_VERSION } from '../../version'
import zepterLogo from '../../assets/zepter-logo.png'

const authStore = useAuthStore()

const matrixCanvas = ref(null)
let matrixTickInterval = null
let matrixResizeHandler = null

// Grille de chiffres façon "Numbers" de Kraftwerk : pas de chute continue,
// toute la grille change en même temps, au tempo, comme un afficheur mécanique.
function startNumberGrid(canvas) {
  const ctx = canvas.getContext('2d')
  const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  if (prefersReducedMotion) return

  const fontSize = 24
  const cellWidth = fontSize * 1.5
  const cellHeight = fontSize * 1.7
  let columns = 0
  let rows = 0

  function resize() {
    canvas.width = canvas.offsetWidth
    canvas.height = canvas.offsetHeight
    columns = Math.ceil(canvas.width / cellWidth)
    rows = Math.ceil(canvas.height / cellHeight)
  }

  resize()
  matrixResizeHandler = resize
  window.addEventListener('resize', matrixResizeHandler)

  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'

  function draw() {
    ctx.clearRect(0, 0, canvas.width, canvas.height)
    ctx.font = `700 ${fontSize}px "Courier New", monospace`

    for (let row = 0; row < rows; row++) {
      for (let col = 0; col < columns; col++) {
        const digit = Math.floor(Math.random() * 10).toString()
        const x = col * cellWidth + cellWidth / 2
        const y = row * cellHeight + cellHeight / 2

        ctx.fillStyle = 'rgba(255, 255, 255, 0.16)'
        ctx.fillText(digit, x, y)
      }
    }
  }

  draw()
  matrixTickInterval = setInterval(draw, 550)
}

onMounted(() => {
  if (matrixCanvas.value) startNumberGrid(matrixCanvas.value)
})

onBeforeUnmount(() => {
  if (matrixTickInterval) clearInterval(matrixTickInterval)
  if (matrixResizeHandler) window.removeEventListener('resize', matrixResizeHandler)
})

// 'login' | 'forgot-request' | 'forgot-verify'
const mode = ref('login')

const username = ref('')
const password = ref('')
const showPassword = ref(false)
const errorMessage = ref('')

function consumeIdleLogoutFlag() {
  try {
    if (localStorage.getItem('idleLogoutMessage') === '1') {
      localStorage.removeItem('idleLogoutMessage')
      return 'Tu as été déconnecté après 15 minutes d’inactivité.'
    }
  } catch {
    // stockage indisponible : pas de message, tant pis
  }

  return ''
}

const loginSuccessMessage = ref(consumeIdleLogoutFlag())
const isSubmitting = ref(false)

const resetCode = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const showNewPassword = ref(false)
const showConfirmPassword = ref(false)
const forgotError = ref('')
const forgotInfo = ref('')
const isSendingCode = ref(false)
const isResetting = ref(false)

async function handleSubmit() {
  if (isSubmitting.value) return

  errorMessage.value = ''
  loginSuccessMessage.value = ''

  if (!username.value.trim() || !password.value) {
    errorMessage.value = 'Identifiant et mot de passe requis.'
    return
  }

  isSubmitting.value = true

  try {
    await authStore.login(username.value.trim(), password.value)
  } catch (err) {
    errorMessage.value = err instanceof Error ? err.message : 'Connexion impossible.'
  } finally {
    isSubmitting.value = false
  }
}

function switchToForgotPassword() {
  mode.value = 'forgot-request'
  forgotError.value = ''
  forgotInfo.value = ''
  resetCode.value = ''
  newPassword.value = ''
  confirmPassword.value = ''
  showNewPassword.value = false
  showConfirmPassword.value = false
}

function switchToLogin() {
  mode.value = 'login'
  errorMessage.value = ''
  showPassword.value = false
}

async function handleRequestCode() {
  if (isSendingCode.value) return

  forgotError.value = ''
  forgotInfo.value = ''

  if (!username.value.trim()) {
    forgotError.value = 'Identifiant requis.'
    return
  }

  isSendingCode.value = true

  try {
    await requestResetCode(username.value.trim())
    forgotInfo.value = 'Si ce compte existe, un code vient d’être envoyé par email.'
    mode.value = 'forgot-verify'
  } catch (err) {
    forgotError.value = err instanceof Error ? err.message : 'Impossible d’envoyer le code.'
  } finally {
    isSendingCode.value = false
  }
}

async function handleResendCode() {
  forgotError.value = ''
  forgotInfo.value = ''
  await handleRequestCode()
}

async function handleResetPassword() {
  if (isResetting.value) return

  forgotError.value = ''

  if (!resetCode.value.trim() || !newPassword.value) {
    forgotError.value = 'Le code et le nouveau mot de passe sont obligatoires.'
    return
  }

  if (newPassword.value.length < 4) {
    forgotError.value = 'Le nouveau mot de passe doit contenir au moins 4 caractères.'
    return
  }

  if (newPassword.value !== confirmPassword.value) {
    forgotError.value = 'Les deux mots de passe ne correspondent pas.'
    return
  }

  isResetting.value = true

  try {
    await resetPasswordWithCode(username.value.trim(), resetCode.value.trim(), newPassword.value)
    password.value = ''
    resetCode.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    mode.value = 'login'
    errorMessage.value = ''
    forgotInfo.value = ''
    loginSuccessMessage.value = 'Mot de passe mis à jour. Tu peux te connecter.'
  } catch (err) {
    forgotError.value = err instanceof Error ? err.message : 'Impossible de modifier le mot de passe.'
  } finally {
    isResetting.value = false
  }
}
</script>

<template>
  <main class="login-shell">
    <div class="login-glow" aria-hidden="true"></div>

    <canvas ref="matrixCanvas" class="matrix-rain" aria-hidden="true"></canvas>

    <section class="login-card">
      <div class="login-brand">
        <div class="login-logo-wrap">
          <img :src="zepterLogo" alt="Zepter" class="login-logo-image" />
        </div>
      </div>

      <p class="login-version">v{{ APP_VERSION }}</p>

      <template v-if="mode === 'login'">
        <div class="login-heading">
          <p class="eyebrow">Espace privé</p>
          <h1>Connexion</h1>
          <p class="login-subtitle">
            Entre tes identifiants pour accéder à tes achats, revenus et budgets.
          </p>
        </div>

        <form class="login-form" @submit.prevent="handleSubmit">
          <label class="form-field">
            <span>Identifiant</span>
            <input
              v-model="username"
              type="text"
              autocomplete="username"
              placeholder="ton identifiant"
              :disabled="isSubmitting"
              autofocus
            />
          </label>

          <label class="form-field">
            <span>Mot de passe</span>
            <div class="password-field">
              <input
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="current-password"
                placeholder="••••••••"
                :disabled="isSubmitting"
              />
              <button
                type="button"
                class="password-toggle"
                :aria-label="showPassword ? 'Masquer le mot de passe' : 'Afficher le mot de passe'"
                :title="showPassword ? 'Masquer le mot de passe' : 'Afficher le mot de passe'"
                @click="showPassword = !showPassword"
              >
                <svg v-if="showPassword" viewBox="0 0 24 24" aria-hidden="true">
                  <path
                    d="M3 3l18 18M10.58 10.58a2 2 0 0 0 2.83 2.83M9.88 5.09A9.77 9.77 0 0 1 12 5c5 0 9 4 10 7-.32.99-1.06 2.35-2.17 3.6M6.15 6.44C3.9 7.9 2.32 9.9 2 12c1 3 5 7 10 7 1.2 0 2.35-.23 3.4-.64"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.8"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
                <svg v-else viewBox="0 0 24 24" aria-hidden="true">
                  <path
                    d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7-10-7-10-7z"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.8"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                  <circle cx="12" cy="12" r="3" fill="none" stroke="currentColor" stroke-width="1.8" />
                </svg>
              </button>
            </div>
          </label>

          <p v-if="errorMessage" class="form-error">{{ errorMessage }}</p>
          <p v-if="loginSuccessMessage" class="form-success">{{ loginSuccessMessage }}</p>

          <button class="primary-btn login-submit" type="submit" :disabled="isSubmitting">
            {{ isSubmitting ? 'Connexion…' : 'Se connecter' }}
          </button>

          <button class="login-link-btn" type="button" @click="switchToForgotPassword">
            Mot de passe oublié ?
          </button>
        </form>
      </template>

      <template v-else-if="mode === 'forgot-request'">
        <div class="login-heading">
          <p class="eyebrow">Sécurité</p>
          <h1>Mot de passe oublié</h1>
          <p class="login-subtitle">
            Indique ton identifiant : un code de vérification te sera envoyé par email.
          </p>
        </div>

        <form class="login-form" @submit.prevent="handleRequestCode">
          <label class="form-field">
            <span>Identifiant</span>
            <input
              v-model="username"
              type="text"
              autocomplete="username"
              placeholder="ton identifiant"
              :disabled="isSendingCode"
              autofocus
            />
          </label>

          <p v-if="forgotError" class="form-error">{{ forgotError }}</p>

          <button class="primary-btn login-submit" type="submit" :disabled="isSendingCode">
            {{ isSendingCode ? 'Envoi…' : 'Envoyer le code' }}
          </button>

          <button class="login-link-btn" type="button" @click="switchToLogin">
            Retour à la connexion
          </button>
        </form>
      </template>

      <template v-else>
        <div class="login-heading">
          <p class="eyebrow">Sécurité</p>
          <h1>Vérification</h1>
          <p class="login-subtitle">
            Saisis le code reçu par email ainsi que ton nouveau mot de passe.
          </p>
        </div>

        <form class="login-form" @submit.prevent="handleResetPassword">
          <p v-if="forgotInfo" class="form-success">{{ forgotInfo }}</p>

          <label class="form-field">
            <span>Code reçu par email</span>
            <input
              v-model="resetCode"
              type="text"
              inputmode="numeric"
              maxlength="6"
              placeholder="123456"
              :disabled="isResetting"
              autofocus
            />
          </label>

          <label class="form-field">
            <span>Nouveau mot de passe</span>
            <div class="password-field">
              <input
                v-model="newPassword"
                :type="showNewPassword ? 'text' : 'password'"
                autocomplete="new-password"
                placeholder="••••••••"
                :disabled="isResetting"
              />
              <button
                type="button"
                class="password-toggle"
                :aria-label="showNewPassword ? 'Masquer le mot de passe' : 'Afficher le mot de passe'"
                :title="showNewPassword ? 'Masquer le mot de passe' : 'Afficher le mot de passe'"
                @click="showNewPassword = !showNewPassword"
              >
                <svg v-if="showNewPassword" viewBox="0 0 24 24" aria-hidden="true">
                  <path
                    d="M3 3l18 18M10.58 10.58a2 2 0 0 0 2.83 2.83M9.88 5.09A9.77 9.77 0 0 1 12 5c5 0 9 4 10 7-.32.99-1.06 2.35-2.17 3.6M6.15 6.44C3.9 7.9 2.32 9.9 2 12c1 3 5 7 10 7 1.2 0 2.35-.23 3.4-.64"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.8"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
                <svg v-else viewBox="0 0 24 24" aria-hidden="true">
                  <path
                    d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7-10-7-10-7z"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.8"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                  <circle cx="12" cy="12" r="3" fill="none" stroke="currentColor" stroke-width="1.8" />
                </svg>
              </button>
            </div>
          </label>

          <label class="form-field">
            <span>Confirmer le nouveau mot de passe</span>
            <div class="password-field">
              <input
                v-model="confirmPassword"
                :type="showConfirmPassword ? 'text' : 'password'"
                autocomplete="new-password"
                placeholder="••••••••"
                :disabled="isResetting"
              />
              <button
                type="button"
                class="password-toggle"
                :aria-label="showConfirmPassword ? 'Masquer le mot de passe' : 'Afficher le mot de passe'"
                :title="showConfirmPassword ? 'Masquer le mot de passe' : 'Afficher le mot de passe'"
                @click="showConfirmPassword = !showConfirmPassword"
              >
                <svg v-if="showConfirmPassword" viewBox="0 0 24 24" aria-hidden="true">
                  <path
                    d="M3 3l18 18M10.58 10.58a2 2 0 0 0 2.83 2.83M9.88 5.09A9.77 9.77 0 0 1 12 5c5 0 9 4 10 7-.32.99-1.06 2.35-2.17 3.6M6.15 6.44C3.9 7.9 2.32 9.9 2 12c1 3 5 7 10 7 1.2 0 2.35-.23 3.4-.64"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.8"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
                <svg v-else viewBox="0 0 24 24" aria-hidden="true">
                  <path
                    d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7-10-7-10-7z"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.8"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                  <circle cx="12" cy="12" r="3" fill="none" stroke="currentColor" stroke-width="1.8" />
                </svg>
              </button>
            </div>
          </label>

          <p v-if="forgotError" class="form-error">{{ forgotError }}</p>

          <button class="primary-btn login-submit" type="submit" :disabled="isResetting">
            {{ isResetting ? 'Mise à jour…' : 'Réinitialiser le mot de passe' }}
          </button>

          <button class="login-link-btn" type="button" @click="handleResendCode" :disabled="isSendingCode">
            {{ isSendingCode ? 'Envoi…' : 'Renvoyer le code' }}
          </button>

          <button class="login-link-btn" type="button" @click="switchToLogin">
            Retour à la connexion
          </button>
        </form>
      </template>
    </section>
  </main>
</template>

<style scoped>
.login-shell {
  position: relative;
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 1.5rem;
  /* Dégradé "hero" volontairement fixe (indépendant du thème clair/sombre) —
     l'écran de connexion garde la même signature visuelle affirmée, comme
     une page d'accroche. La carte de connexion reste donc toujours sur fond
     sombre : on fige ici les variables de texte à des teintes claires pour
     que tout le reste de ce fichier (qui lit var(--text)/var(--text-soft)/
     var(--text-dim)) reste lisible même quand le thème choisi est "clair". */
  --text: #f3f5fa;
  --text-soft: #c9d2e6;
  --text-dim: #93a1c2;
  --tint-rgb: 255, 255, 255;
  --line: rgba(255, 255, 255, 0.18);
  background: var(--hero-gradient);
  overflow: hidden;
}

.login-glow {
  position: absolute;
  inset: -20%;
  background:
    radial-gradient(circle at 20% 20%, rgba(59, 130, 246, 0.22), transparent 45%),
    radial-gradient(circle at 80% 75%, rgba(242, 168, 120, 0.22), transparent 40%);
  pointer-events: none;
}

.matrix-rain {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  opacity: 0.5;
  pointer-events: none;
}

.login-card {
  position: relative;
  width: min(100%, 420px);
  padding: 2rem 1.9rem;
  border-radius: var(--radius-xl, 28px);
  background: rgba(22, 29, 46, 0.62);
  border: 1px solid rgba(255, 255, 255, 0.14);
  box-shadow: 0 30px 70px rgba(5, 8, 16, 0.45);
  backdrop-filter: blur(22px);
  -webkit-backdrop-filter: blur(22px);
}

.login-brand {
  display: flex;
  justify-content: center;
  margin-bottom: 1.6rem;
}

/* Fond du logo transparent (le noir d'origine a été détouré) — un halo doux
   aux couleurs du bouton "Se connecter" (var(--accent)/var(--accent-soft))
   le met en valeur sans jamais former de bloc/plaque visible. */
.login-logo-wrap {
  position: relative;
  width: 100%;
  max-width: 210px;
}

.login-logo-wrap::before {
  content: '';
  position: absolute;
  inset: -12%;
  background: radial-gradient(circle, var(--accent, #728998) 0%, var(--accent-soft, #819b8d) 45%, transparent 72%);
  opacity: 0.32;
  filter: blur(18px);
  pointer-events: none;
}

.login-logo-image {
  position: relative;
  display: block;
  width: 100%;
  height: auto;
}

.login-heading {
  margin-bottom: 1.6rem;
}

.eyebrow {
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.74rem;
}

.login-heading h1 {
  margin: 0.3rem 0 0;
  color: var(--text, #eef1f3);
  font-size: 1.6rem;
  line-height: 1.2;
}

.login-subtitle {
  margin: 0.6rem 0 0;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.9rem;
  line-height: 1.5;
}

.login-form {
  display: grid;
  gap: 1rem;
}

.password-field {
  position: relative;
}

.password-field input {
  padding-right: 44px;
}

.password-toggle {
  position: absolute;
  top: 50%;
  right: 6px;
  transform: translateY(-50%);
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-dim, #8a939d);
  cursor: pointer;
  transition: background 140ms ease, color 140ms ease;
}

.password-toggle:hover {
  background: rgba(var(--tint-rgb), 0.06);
  color: var(--text, #eef1f3);
}

.password-toggle svg {
  width: 18px;
  height: 18px;
}

.login-submit {
  width: 100%;
  height: 46px;
  margin-top: 0.3rem;
  font-size: 0.95rem;
}

.login-link-btn {
  width: 100%;
  border: none;
  background: transparent;
  color: var(--text-dim, #8a939d);
  font-size: 0.84rem;
  font-weight: 600;
  cursor: pointer;
  padding: 0.3rem;
  text-align: center;
  transition: color 140ms ease;
}

.login-link-btn:hover {
  color: var(--text-soft, #b3bbc4);
  text-decoration: underline;
}

.form-success {
  margin: 0;
  padding: 0.85rem 1rem;
  border-radius: 14px;
  background: rgba(143, 168, 160, 0.14);
  color: var(--positive-text, #bfe0c9);
  border: 1px solid rgba(143, 168, 160, 0.24);
  font-size: 0.92rem;
}

.login-version {
  position: absolute;
  top: 1.4rem;
  right: 1.9rem;
  margin: 0;
  color: var(--text-dim, #8a939d);
  font-size: 0.76rem;
  letter-spacing: 0.02em;
}

@media (max-width: 480px) {
  .login-card {
    padding: 1.6rem 1.3rem;
  }
}
</style>
