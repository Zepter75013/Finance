import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'

const STORAGE_KEY = 'appTheme'
const VALID_THEMES = ['system', 'light', 'dark']

export const useThemeStore = defineStore('theme', () => {
  // Le thème bleu nuit fait partie de l'identité visuelle de l'app (pas
  // seulement une variante "mode sombre" parmi d'autres) — nouveau réglage
  // par défaut tant que l'utilisateur n'a pas fait de choix explicite,
  // plutôt que de dépendre de la préférence claire/sombre du système.
  const storedTheme = localStorage.getItem(STORAGE_KEY)
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
    localStorage.setItem(STORAGE_KEY, value)
  }

  watch(resolvedTheme, applyTheme, { immediate: true })

  return { theme, resolvedTheme, setTheme }
})
