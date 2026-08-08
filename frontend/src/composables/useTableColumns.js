import { computed, reactive } from 'vue'

export function useTableColumns(storageKey, columns) {
  function loadState() {
    try {
      return JSON.parse(localStorage.getItem(storageKey) || '{}')
    } catch {
      return {}
    }
  }

  const stored = loadState()

  const state = reactive({
    hidden: new Set(stored.hidden || []),
    frozen: new Set(stored.frozen || columns.filter((c) => c.defaultFrozen).map((c) => c.key)),
  })

  function persist() {
    localStorage.setItem(
      storageKey,
      JSON.stringify({
        hidden: Array.from(state.hidden),
        frozen: Array.from(state.frozen),
      })
    )
  }

  function isVisible(key) {
    return !state.hidden.has(key)
  }

  function isFrozen(key) {
    return state.frozen.has(key)
  }

  function toggleVisible(key) {
    if (state.hidden.has(key)) state.hidden.delete(key)
    else state.hidden.add(key)
    persist()
  }

  function toggleFrozen(key) {
    if (state.frozen.has(key)) state.frozen.delete(key)
    else state.frozen.add(key)
    persist()
  }

  const visibleColumns = computed(() => columns.filter((column) => isVisible(column.key)))

  const frozenLeftOffsets = computed(() => {
    const offsets = {}
    let cumulative = 0

    for (const column of visibleColumns.value) {
      if (isFrozen(column.key)) {
        offsets[column.key] = cumulative
        cumulative += column.width || 140
      }
    }

    return offsets
  })

  function columnStyle(key) {
    if (!isFrozen(key)) return {}

    return {
      position: 'sticky',
      left: `${frozenLeftOffsets.value[key] || 0}px`,
      zIndex: 2,
    }
  }

  return {
    columns,
    isVisible,
    isFrozen,
    toggleVisible,
    toggleFrozen,
    visibleColumns,
    columnStyle,
  }
}
