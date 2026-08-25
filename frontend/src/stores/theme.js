import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'

const STORAGE_KEY = 'appTheme'
const VALID_THEMES = ['system', 'light', 'dark']

export const useThemeStore = defineStore('theme', () => {
  // Le thème bleu nuit fait partie de l'identité visuelle de l'app (pas
  // seulement une variante "mode sombre" parmi d'autres) — nouveau réglage
  // par défaut tant que l'utilisateur n'a pas fait de choix explicite,
  // plutôt que de dépendre de la préférence claire/sombre du système.
  // localStorage peut lever (navigation privée Safari, cookies/stockage
  // bloqués) — un thème non lisible/enregistrable ne doit pas empêcher
  // l'app de démarrer, juste retomber sur le défaut à chaque session.
  let storedTheme = null
  try {
    storedTheme = localStorage.getItem(STORAGE_KEY)
  } catch {
    storedTheme = null
  }

  const theme = ref(VALID_THEMES.includes(storedTheme) ? storedTheme : 'dark')

  const media = window.matchMedia ? window.matchMedia('(prefers-color-scheme: dark)') : null
  const systemPrefersDark = ref(media ? media.matches : true)

  if (media) {
    media.addEventListener('change', (event) => {
      systemPrefersDark.value = event.matches
    })
  }

  const resolvedTheme = computed(() => {
    return theme.value === 'system' ? (systemPrefersDark.value ? 'dark' : 'light') : theme.value
  })

  function applyTheme() {
    document.documentElement.setAttribute('data-theme', resolvedTheme.value)
  }

  function setTheme(value) {
    if (!VALID_THEMES.includes(value)) return
    theme.value = value

    try {
      localStorage.setItem(STORAGE_KEY, value)
    } catch {
      // Stockage indisponible (navigation privée, quota…) — le thème reste
      // actif pour la session en cours mais ne survivra pas à un rechargement.
    }
  }

  watch(resolvedTheme, applyTheme, { immediate: true })

  return { theme, resolvedTheme, setTheme }
})
