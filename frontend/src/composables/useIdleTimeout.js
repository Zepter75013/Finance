import { onBeforeUnmount, ref, watch } from 'vue'

const ACTIVITY_EVENTS = ['mousemove', 'mousedown', 'keydown', 'scroll', 'touchstart', 'wheel']

// Déconnexion automatique après un délai d'inactivité — n'écoute les
// interactions (souris/clavier/scroll/tactile) que pendant que `isActive` est
// vrai, pour ne rien surveiller sur l'écran de connexion.
export function useIdleTimeout(isActive, timeoutMs, onTimeout) {
  const isTracking = ref(false)
  let timer = null

  function resetTimer() {
    if (timer) clearTimeout(timer)
    timer = setTimeout(onTimeout, timeoutMs)
  }

  function startTracking() {
    if (isTracking.value) return
    isTracking.value = true

    for (const eventName of ACTIVITY_EVENTS) {
      window.addEventListener(eventName, resetTimer, { passive: true })
    }

    resetTimer()
  }

  function stopTracking() {
    if (!isTracking.value) return
    isTracking.value = false

    if (timer) {
      clearTimeout(timer)
      timer = null
    }

    for (const eventName of ACTIVITY_EVENTS) {
      window.removeEventListener(eventName, resetTimer)
    }
  }

  watch(
    isActive,
    (active) => {
      if (active) startTracking()
      else stopTracking()
    },
    { immediate: true }
  )

  onBeforeUnmount(stopTracking)
}
