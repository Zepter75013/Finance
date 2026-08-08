<script setup>
const props = defineProps({
  income: {
    type: Object,
    required: true,
  },
  selected: {
    type: Boolean,
    default: false,
  },
  checked: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['select', 'edit', 'delete', 'toggle-check'])

function formatCurrency(value) {
  return new Intl.NumberFormat('fr-FR', {
    style: 'currency',
    currency: 'EUR',
    minimumFractionDigits: 2,
  }).format(Number(value || 0))
}

function formatIncomeDate(value) {
  if (!value) return ''

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''

  const day = String(date.getDate()).padStart(2, '0')
  const month = String(date.getMonth() + 1).padStart(2, '0')

  return `${day}/${month}/${date.getFullYear()}`
}

function getSourceIcon(source) {
  const normalized = String(source || '').toLowerCase()

  if (normalized.includes('salaire')) return '💼'
  if (normalized.includes('rembours')) return '💸'
  if (normalized.includes('prime')) return '🎁'
  if (normalized.includes('remise') || normalized.includes('cashback')) return '🏷️'
  if (normalized.includes('aide') || normalized.includes('caf')) return '🤝'
  if (normalized.includes('vente')) return '🏷️'

  return '💰'
}
</script>

<template>
  <article
    class="income-row"
    :class="{ 'income-row-selected': selected }"
    @click="emit('select', props.income)"
    @dblclick="emit('edit', props.income)"
  >
    <div class="income-row-top">
      <div class="income-row-main">
        <input
          type="checkbox"
          class="row-checkbox"
          :checked="checked"
          aria-label="Sélectionner ce revenu"
          @click.stop
          @dblclick.stop
          @change="emit('toggle-check', props.income)"
        />

        <span class="row-icon" :title="income.source" aria-hidden="true">
          {{ getSourceIcon(income.source) }}
        </span>

        <div class="income-amount-inline">
          <p class="eyebrow income-inline-label">
            <span>Revenu</span>
            <span v-if="income.note" class="income-note-inline">
              — {{ income.note }}
            </span>
          </p>

          <div class="income-value-line">
            <h3>{{ formatCurrency(income.amount) }}</h3>
            <span class="income-date-inline">
              {{ formatIncomeDate(income.income_date) }}
            </span>
            <span
              class="reconciled-badge-inline"
              :class="income.isReconciled ? 'is-reconciled' : 'is-pending'"
              :title="income.isReconciled && income.statementReference ? `Relevé ${income.statementReference}` : ''"
            >
              {{ income.isReconciled ? '✓ Pointé' : 'Non pointé' }}
            </span>
          </div>
        </div>
      </div>

      <div class="income-row-actions" @click.stop @dblclick.stop>
        <button
          class="icon-btn"
          type="button"
          aria-label="Modifier ce revenu"
          title="Modifier"
          @click="emit('edit', props.income)"
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
          class="icon-btn icon-btn-danger"
          type="button"
          aria-label="Supprimer ce revenu"
          title="Supprimer"
          @click="emit('delete', props.income)"
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
    </div>
  </article>
</template>

<style scoped>
.income-row {
  padding: 0.6rem 0.85rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.028);
  border: 1px solid rgba(var(--tint-rgb), 0.05);
  cursor: pointer;
  transition:
    background 160ms ease,
    border-color 160ms ease,
    transform 160ms ease,
    box-shadow 160ms ease;
}

.income-row:hover {
  background: rgba(var(--tint-rgb), 0.05);
  border-color: rgba(143, 168, 160, 0.18);
  transform: translateY(-1px);
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.16);
}

.income-row-selected {
  background: rgba(143, 168, 160, 0.08);
  border-color: rgba(143, 168, 160, 0.28);
  box-shadow: 0 0 0 1px rgba(143, 168, 160, 0.12);
}

.income-row-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.9rem;
}

.income-row-main {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  min-width: 0;
  flex: 1;
}

.row-checkbox {
  flex-shrink: 0;
  width: 16px;
  height: 16px;
  accent-color: var(--accent, #728998);
  cursor: pointer;
}

.row-icon {
  flex-shrink: 0;
  width: 26px;
  height: 26px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  background: rgba(var(--tint-rgb), 0.045);
  font-size: 0.85rem;
  line-height: 1;
}

.income-amount-inline {
  min-width: 0;
}

.income-inline-label {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  min-width: 0;
}

.income-value-line {
  display: flex;
  align-items: baseline;
  gap: 0.7rem;
  min-width: 0;
}

.income-row h3 {
  margin: 0.16rem 0 0;
  color: var(--text, #eef1f3);
  font-size: 0.92rem;
  line-height: 1.15;
}

.income-note-inline {
  color: var(--text-soft, #b3bbc4);
  font-size: 0.7rem;
  line-height: 1;
  text-transform: none;
  letter-spacing: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 320px;
}

.income-date-inline {
  color: var(--text-dim, #8a939d);
  font-size: 0.72rem;
  white-space: nowrap;
}

.reconciled-badge-inline {
  display: inline-block;
  padding: 0.15rem 0.5rem;
  border-radius: 999px;
  font-size: 0.68rem;
  font-weight: 600;
  white-space: nowrap;
}

.reconciled-badge-inline.is-reconciled {
  background: color-mix(in srgb, var(--bg-elevated, #2c3137) 78%, #22c55e 22%);
  color: var(--positive-text, #bfe0c9);
}

.reconciled-badge-inline.is-pending {
  color: var(--text-dim, #8a939d);
}

.income-row-actions {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  margin-left: auto;
  flex-shrink: 0;
}

.icon-btn {
  width: 30px;
  height: 30px;
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  border-radius: 10px;
  display: grid;
  place-items: center;
  background: rgba(var(--tint-rgb), 0.035);
  color: var(--text);
  cursor: pointer;
  transition:
    transform 140ms ease,
    background 140ms ease,
    border-color 140ms ease,
    box-shadow 140ms ease;
}

.icon-btn:hover {
  transform: translateY(-1px);
  background: rgba(var(--tint-rgb), 0.085);
  border-color: rgba(var(--tint-rgb), 0.14);
}

.icon-btn:focus-visible {
  outline: none;
  border-color: rgba(143, 168, 160, 0.42);
  box-shadow: 0 0 0 3px rgba(143, 168, 160, 0.14);
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

.icon-btn-danger:hover {
  background: rgba(220, 38, 38, 0.16);
  border-color: rgba(220, 38, 38, 0.22);
}

.icon-btn-danger:focus-visible {
  border-color: rgba(220, 38, 38, 0.32);
  box-shadow: 0 0 0 3px rgba(220, 38, 38, 0.12);
}

.eyebrow {
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.72rem;
}

@media (max-width: 720px) {
  .income-row-top {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
  }

  .income-inline-label,
  .income-value-line {
    flex-wrap: wrap;
  }

  .income-note-inline {
    max-width: 100%;
    white-space: normal;
    line-height: 1.35;
  }

  .income-row-actions {
    margin-left: 0;
    padding-top: 0.1rem;
  }
}
</style>
