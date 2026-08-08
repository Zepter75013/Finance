<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  label: {
    type: String,
    required: true,
  },
  options: {
    type: Array,
    default: () => [], // [{ value, label }] — mode liste plate
  },
  groups: {
    type: Array,
    default: () => [], // [{ key, label, options: [{ value, label }] }] — mode groupé
  },
  modelValue: {
    type: Array,
    default: () => [],
  },
  allLabel: {
    type: String,
    default: 'Tous',
  },
})

const emit = defineEmits(['update:modelValue'])

const isOpen = ref(false)

function toggle() {
  isOpen.value = !isOpen.value
}

function close() {
  isOpen.value = false
}

const isGrouped = computed(() => props.groups.length > 0)

// Toutes les options, groupées ou non — sert de base commune au décompte et
// au "Tout sélectionner".
const flatOptions = computed(() => {
  if (isGrouped.value) return props.groups.flatMap((group) => group.options)
  return props.options
})

const triggerLabel = computed(() => {
  if (!props.modelValue.length) return props.allLabel

  if (props.modelValue.length === 1) {
    const match = flatOptions.value.find((option) => option.value === props.modelValue[0])
    if (!match) return props.modelValue[0]

    if (isGrouped.value) {
      const group = props.groups.find((g) => g.options.includes(match))
      return group ? `${group.label} : ${match.label}` : match.label
    }

    return match.label
  }

  return `${props.modelValue.length} sélectionnées`
})

function isChecked(value) {
  return props.modelValue.includes(value)
}

function toggleValue(value) {
  const next = isChecked(value)
    ? props.modelValue.filter((v) => v !== value)
    : [...props.modelValue, value]
  emit('update:modelValue', next)
}

const isAllSelected = computed(
  () => flatOptions.value.length > 0 && props.modelValue.length === flatOptions.value.length
)

function toggleSelectAll() {
  emit('update:modelValue', isAllSelected.value ? [] : flatOptions.value.map((option) => option.value))
}

function groupSelectedCount(group) {
  return group.options.filter((option) => isChecked(option.value)).length
}

function isGroupFullySelected(group) {
  return group.options.length > 0 && groupSelectedCount(group) === group.options.length
}

function isGroupPartiallySelected(group) {
  const count = groupSelectedCount(group)
  return count > 0 && count < group.options.length
}

function toggleGroup(group) {
  const groupValues = group.options.map((option) => option.value)

  if (isGroupFullySelected(group)) {
    emit(
      'update:modelValue',
      props.modelValue.filter((v) => !groupValues.includes(v))
    )
    return
  }

  const next = new Set(props.modelValue)
  groupValues.forEach((v) => next.add(v))
  emit('update:modelValue', Array.from(next))
}

// Reflète l'état "partiellement coché" sur la case native de la catégorie —
// pas exposable via un attribut HTML, seulement via la propriété DOM.
function applyGroupIndeterminate(el, group) {
  if (el) el.indeterminate = isGroupPartiallySelected(group)
}
</script>

<template>
  <div class="multi-select">
    <span class="sort-label">{{ props.label }}</span>
    <button class="sort-select multi-select-trigger" type="button" @click="toggle">
      {{ triggerLabel }}
    </button>

    <div v-if="isOpen" class="multi-select-backdrop" @click="close"></div>

    <div v-if="isOpen" class="multi-select-panel">
      <div class="multi-select-row multi-select-all-row">
        <label class="multi-select-check">
          <input type="checkbox" :checked="isAllSelected" @change="toggleSelectAll" />
          <span>Tout sélectionner</span>
        </label>
      </div>

      <template v-if="isGrouped">
        <div v-for="group in props.groups" :key="group.key" class="multi-select-group">
          <div class="multi-select-row">
            <label class="multi-select-check multi-select-group-check">
              <input
                type="checkbox"
                :checked="isGroupFullySelected(group)"
                :ref="(el) => applyGroupIndeterminate(el, group)"
                @change="toggleGroup(group)"
              />
              <span>{{ group.label }}</span>
            </label>
          </div>

          <div v-for="option in group.options" :key="option.value" class="multi-select-row multi-select-subrow">
            <label class="multi-select-check">
              <input
                type="checkbox"
                :checked="isChecked(option.value)"
                @change="toggleValue(option.value)"
              />
              <span>{{ option.label }}</span>
            </label>
          </div>
        </div>

        <p v-if="!props.groups.length" class="multi-select-empty">Aucune option disponible</p>
      </template>

      <template v-else>
        <div v-for="option in props.options" :key="option.value" class="multi-select-row">
          <label class="multi-select-check">
            <input
              type="checkbox"
              :checked="isChecked(option.value)"
              @change="toggleValue(option.value)"
            />
            <span>{{ option.label }}</span>
          </label>
        </div>

        <p v-if="!props.options.length" class="multi-select-empty">Aucune option disponible</p>
      </template>
    </div>
  </div>
</template>

<style scoped>
.multi-select {
  position: relative;
  display: flex;
  align-items: center;
  gap: 0.55rem;
}

.sort-label {
  color: var(--text-dim, #8a939d);
  font-size: 0.76rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.multi-select-trigger {
  height: 32px;
  min-width: 168px;
  border-radius: 10px;
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--text, #eef1f3);
  padding: 0 0.7rem;
  font-size: 0.8rem;
  text-align: left;
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.multi-select-trigger:hover {
  background: rgba(var(--tint-rgb), 0.07);
}

.multi-select-trigger:focus-visible {
  outline: none;
  border-color: rgba(143, 168, 160, 0.34);
  box-shadow: 0 0 0 3px rgba(143, 168, 160, 0.12);
}

.multi-select-backdrop {
  position: fixed;
  inset: 0;
  z-index: 40;
}

.multi-select-panel {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  z-index: 41;
  width: max-content;
  min-width: 240px;
  max-width: 480px;
  max-height: 340px;
  overflow-y: auto;
  padding: 0.55rem;
  border-radius: 14px;
  background: var(--modal-bg, #262a2f);
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.06));
  box-shadow: 0 18px 40px rgba(0, 0, 0, 0.28);
}

.multi-select-all-row {
  padding-bottom: 0.35rem;
  margin-bottom: 0.35rem;
  border-bottom: 1px solid rgba(var(--tint-rgb), 0.08);
}

.multi-select-group {
  padding-bottom: 0.2rem;
}

.multi-select-group:not(:last-child) {
  margin-bottom: 0.25rem;
  border-bottom: 1px solid rgba(var(--tint-rgb), 0.05);
}

.multi-select-group-check {
  font-weight: 600;
}

.multi-select-row {
  padding: 0.3rem 0.2rem;
}

.multi-select-subrow {
  padding-left: 1.5rem;
}

.multi-select-check {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
  color: var(--text, #eef1f3);
  font-size: 0.84rem;
  cursor: pointer;
}

.multi-select-check span {
  white-space: normal;
  word-break: break-word;
}

.multi-select-check input {
  flex-shrink: 0;
  accent-color: var(--accent, #728998);
  cursor: pointer;
}

.multi-select-empty {
  margin: 0.3rem 0.2rem;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.82rem;
}
</style>
