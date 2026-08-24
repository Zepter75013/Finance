<script setup>
import { formatCurrency, formatDate as formatPurchaseDate } from '../../utils/format'

const props = defineProps({
  purchase: {
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

const emit = defineEmits(['select', 'edit', 'delete', 'toggle-check', 'undo'])

// Un virement créé directement (pas via la conversion d'un achat) n'a pas de
// ligne d'origine à restaurer.
function canUndoTransfer(purchase) {
  return Boolean(purchase.raw?.originType) && Boolean(purchase.raw?.originPayload)
}

function formatMerchantLabel(value) {
  return value?.trim() || 'Sans commerçant'
}

function getCategoryIcon(category) {
  const normalized = String(category || '').toLowerCase()

  if (normalized.includes('course')) return '🛒'
  if (normalized.includes('transport')) return '🚗'
  if (normalized.includes('loisir')) return '🎮'
  if (normalized.includes('sant')) return '💊'
  if (normalized.includes('maison')) return '🏠'
  if (normalized.includes('resto') || normalized.includes('restau')) return '🍽️'
  if (normalized.includes('abonnement') || normalized.includes('facture')) return '🧾'

  return '💳'
}


</script>

<template>
  <article
    class="purchase-row-card"
    :class="{ 'purchase-row-selected': selected }"
    @click="emit('select', props.purchase)"
    @dblclick="emit('edit', props.purchase)"
  >
    <div class="purchase-row-top">
      <div class="purchase-row-main">
        <input
          type="checkbox"
          class="row-checkbox"
          :checked="checked"
          :disabled="purchase.isTransfer"
          :title="purchase.isTransfer ? 'Les virements se gèrent depuis leur propre bouton Supprimer' : ''"
          aria-label="Sélectionner cet achat"
          @click.stop
          @dblclick.stop
          @change="emit('toggle-check', props.purchase)"
        />

        <span class="row-icon" :title="purchase.isTransfer ? 'Virement' : purchase.category" aria-hidden="true">
          {{ purchase.isTransfer ? '🔀' : getCategoryIcon(purchase.category) }}
        </span>

        <div class="purchase-amount-inline">
          <p class="eyebrow purchase-inline-label">
            <span>{{ formatMerchantLabel(purchase.merchant) }}</span>
            <span v-if="purchase.note" class="purchase-note-inline">
              — {{ purchase.note }}
            </span>
          </p>

          <div class="purchase-value-line">
            <h3>{{ formatCurrency(purchase.amount) }}</h3>
            <span class="purchase-date-inline">
              {{ formatPurchaseDate(purchase.date) }}
            </span>
            <span v-if="purchase.isTransfer" class="transfer-badge-inline">
              Virement
            </span>
            <span
              v-else
              class="reconciled-badge-inline"
              :class="purchase.isReconciled ? 'is-reconciled' : 'is-pending'"
              :title="purchase.isReconciled && purchase.statementReference ? `Relevé ${purchase.statementReference}` : ''"
            >
              {{ purchase.isReconciled ? '✓ Pointé' : 'Non pointé' }}
            </span>
          </div>
        </div>
      </div>

      <div class="purchase-row-actions" @click.stop @dblclick.stop>
        <button
          v-if="purchase.isTransfer && canUndoTransfer(purchase)"
          class="icon-btn"
          type="button"
          aria-label="Annuler ce transfert et revenir à l'achat d'origine"
          title="Annuler ce transfert et revenir à l'achat d'origine"
          @click="emit('undo', props.purchase)"
        >
          ↩️
        </button>

        <button
          v-else
          class="icon-btn"
          type="button"
          :aria-label="purchase.isTransfer ? 'Modifier ce virement' : 'Modifier cet achat'"
          :title="purchase.isTransfer ? 'Virement — modifier' : 'Modifier'"
          @click="emit('edit', props.purchase)"
        >
          <span v-if="purchase.isTransfer" aria-hidden="true">🔀</span>
          <svg v-else viewBox="0 0 24 24" aria-hidden="true">
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
          aria-label="Supprimer cet achat"
          title="Supprimer"
          @click="emit('delete', props.purchase)"
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
.purchase-row-card {
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

.purchase-row-card:hover {
  background: rgba(var(--tint-rgb), 0.05);
  border-color: rgba(143, 168, 160, 0.18);
  transform: translateY(-1px);
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.16);
}

.purchase-row-selected {
  background: rgba(143, 168, 160, 0.08);
  border-color: rgba(143, 168, 160, 0.28);
  box-shadow: 0 0 0 1px rgba(143, 168, 160, 0.12);
}

.purchase-row-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.9rem;
}

.purchase-row-main {
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

.purchase-amount-inline {
  min-width: 0;
}

.purchase-inline-label {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  min-width: 0;
}

.purchase-value-line {
  display: flex;
  align-items: baseline;
  gap: 0.7rem;
  min-width: 0;
}

.purchase-row-card h3 {
  margin: 0.16rem 0 0;
  color: var(--text, #eef1f3);
  font-size: 0.92rem;
  line-height: 1.15;
}

.purchase-note-inline {
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

.purchase-date-inline {
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

.transfer-badge-inline {
  display: inline-block;
  padding: 0.15rem 0.5rem;
  border-radius: 999px;
  font-size: 0.68rem;
  font-weight: 600;
  white-space: nowrap;
  color: #f0a95e;
  background: rgba(240, 169, 94, 0.16);
}

.purchase-row-actions {
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

@media (max-width: 1100px) {
  .purchase-note-inline {
    max-width: 240px;
  }
}

@media (max-width: 720px) {
  .purchase-row-top {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
  }

  .purchase-inline-label,
  .purchase-value-line {
    flex-wrap: wrap;
  }

  .purchase-note-inline {
    max-width: 100%;
    white-space: normal;
    line-height: 1.35;
  }

  .purchase-row-actions {
    margin-left: 0;
    padding-top: 0.1rem;
  }
}
</style>
