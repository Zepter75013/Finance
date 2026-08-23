import { ref, computed, watch } from 'vue'
import { defineStore } from 'pinia'
import { fetchRecurring } from '../services/recurring'
import { usePurchasesStore } from './purchases'
import { daysUntil } from '../utils/dates'

const UPCOMING_WINDOW_DAYS = 7

export const useRecurringStore = defineStore('recurring', () => {
  const purchasesStore = usePurchasesStore()
  const items = ref([])
  const isLoading = ref(false)
  const loadError = ref('')
  let suppressWatch = true

  async function loadRecurring() {
    const accountId = purchasesStore.activeAccountId
    if (!accountId) {
      items.value = []
      return
    }

    isLoading.value = true
    loadError.value = ''

    try {
      items.value = await fetchRecurring(accountId)
    } catch (err) {
      loadError.value = err instanceof Error ? err.message : 'Impossible de charger les récurrences.'
      items.value = []
    } finally {
      isLoading.value = false
      suppressWatch = false
    }
  }

  watch(
    () => purchasesStore.activeAccountId,
    (id, previousId) => {
      if (suppressWatch) return
      if (!id || id === previousId) return

      items.value = []
      loadRecurring()
    }
  )

  const overdueItems = computed(() =>
    items.value.filter((item) => item.is_active && daysUntil(item.next_run_date) <= 0)
  )

  const upcomingItems = computed(() =>
    items.value.filter((item) => {
      const days = daysUntil(item.next_run_date)
      return item.is_active && days >= 1 && days <= UPCOMING_WINDOW_DAYS
    })
  )

  return {
    items,
    isLoading,
    loadError,
    loadRecurring,
    overdueItems,
    upcomingItems,
  }
})
