<script setup>
import { ref } from 'vue'

const props = defineProps({
  columns: {
    type: Array,
    required: true,
  },
  isVisible: {
    type: Function,
    required: true,
  },
  isFrozen: {
    type: Function,
    required: true,
  },
})

const emit = defineEmits(['toggle-visible', 'toggle-frozen'])

const isOpen = ref(false)

function toggle() {
  isOpen.value = !isOpen.value
}

function close() {
  isOpen.value = false
}
</script>

<template>
  <div class="column-menu">
    <button class="ghost-btn column-menu-trigger" type="button" @click="toggle">
      Colonnes
    </button>

    <div v-if="isOpen" class="column-menu-backdrop" @click="close"></div>

    <div v-if="isOpen" class="column-menu-panel">
      <p class="column-menu-title">Afficher / figer les colonnes</p>

      <div v-for="column in props.columns" :key="column.key" class="column-menu-row">
        <label class="column-menu-check">
          <input
            type="checkbox"
            :checked="props.isVisible(column.key)"
            @change="emit('toggle-visible', column.key)"
          />
          <span>{{ column.label }}</span>
        </label>

        <button
          class="column-menu-pin"
          type="button"
          :class="{ 'is-active': props.isFrozen(column.key) }"
          :disabled="!props.isVisible(column.key)"
          title="Figer cette colonne"
          aria-label="Figer cette colonne"
          @click="emit('toggle-frozen', column.key)"
        >
          📌
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.column-menu {
  position: relative;
}

.column-menu-trigger {
  height: 38px;
  padding: 0 0.9rem;
  border-radius: 10px;
  font-size: 0.82rem;
}

.column-menu-backdrop {
  position: fixed;
  inset: 0;
  z-index: 40;
}

.column-menu-panel {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  z-index: 41;
  width: 240px;
  max-height: 340px;
  overflow-y: auto;
  padding: 0.7rem;
  border-radius: 14px;
  background: var(--modal-bg, #262a2f);
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.06));
  box-shadow: 0 18px 40px rgba(0, 0, 0, 0.28);
}

.column-menu-title {
  margin: 0 0 0.5rem;
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-size: 0.68rem;
}

.column-menu-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.35rem 0.2rem;
}

.column-menu-check {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
  color: var(--text, #eef1f3);
  font-size: 0.84rem;
  cursor: pointer;
}

.column-menu-check span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.column-menu-check input {
  flex-shrink: 0;
  accent-color: var(--accent, #728998);
  cursor: pointer;
}

.column-menu-pin {
  flex-shrink: 0;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 8px;
  background: transparent;
  font-size: 0.85rem;
  line-height: 1;
  cursor: pointer;
  opacity: 0.4;
  transition: opacity 140ms ease, background 140ms ease;
}

.column-menu-pin:hover:not(:disabled) {
  background: rgba(var(--tint-rgb), 0.08);
  opacity: 0.7;
}

.column-menu-pin.is-active {
  opacity: 1;
  background: rgba(114, 137, 152, 0.18);
}

.column-menu-pin:disabled {
  opacity: 0.15;
  cursor: not-allowed;
}
</style>
