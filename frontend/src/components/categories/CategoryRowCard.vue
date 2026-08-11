<script setup>
import { formatCurrency } from '../../utils/format'
const props = defineProps({
  category: {
    type: Object,
    required: true,
  },
  checked: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['edit', 'delete', 'toggle-check', 'open-purchases'])


function handleOpenPurchases() {
  if (props.category.purchaseCount === 0) return
  emit('open-purchases', props.category)
}
</script>

<template>
  <article class="category-row-card" @dblclick="emit('edit', props.category)">
    <div class="category-row-top">
      <div class="category-row-main">
        <input
          type="checkbox"
          class="row-checkbox"
          :checked="checked"
          :disabled="category.purchaseCount > 0"
          :title="
            category.purchaseCount > 0
              ? 'Catégorie utilisée par des achats — suppression impossible'
              : 'Sélectionner cette catégorie'
          "
          aria-label="Sélectionner cette catégorie"
          @click.stop
          @dblclick.stop
          @change="emit('toggle-check', props.category)"
        />

        <span class="row-icon" aria-hidden="true">#</span>

        <div class="category-amount-inline">
          <p class="eyebrow category-inline-label">
            <span>{{ category.name }}</span>
          </p>

          <div class="category-value-line">
            <h3>{{ formatCurrency(category.totalAmount) }}</h3>
            <button
              class="category-count-badge"
              type="button"
              :disabled="category.purchaseCount === 0"
              :title="
                category.purchaseCount === 0
                  ? 'Aucun achat dans cette catégorie'
                  : `Voir les achats de ${category.name}`
              "
              @click.stop="handleOpenPurchases"
              @dblclick.stop
            >
              {{ category.purchaseCount }} achat<span v-if="category.purchaseCount > 1">s</span>
            </button>
          </div>
        </div>
      </div>

      <div class="category-row-actions" @click.stop @dblclick.stop>
        <button
          class="icon-btn"
          type="button"
          aria-label="Modifier cette catégorie"
          title="Modifier"
          @click="emit('edit', props.category)"
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
          :disabled="category.purchaseCount > 0"
          :aria-label="
            category.purchaseCount > 0
              ? 'Catégorie utilisée par des achats — suppression impossible'
              : 'Supprimer cette catégorie'
          "
          :title="
            category.purchaseCount > 0
              ? 'Catégorie utilisée par des achats — suppression impossible'
              : 'Supprimer'
          "
          @click="emit('delete', props.category)"
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

    <div v-if="category.subCategories?.length" class="category-subs">
      <span v-for="sub in category.subCategories" :key="sub.id" class="category-sub-tag">
        {{ sub.name }}
      </span>
    </div>
  </article>
</template>

<style scoped>
.category-subs {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  margin-top: 0.55rem;
  padding-left: 2.35rem;
}

.category-sub-tag {
  padding: 0.18rem 0.55rem;
  border-radius: 999px;
  background: rgba(var(--tint-rgb), 0.045);
  color: var(--text-dim, #8a939d);
  border: 1px solid rgba(var(--tint-rgb), 0.06);
  font-size: 0.7rem;
  font-weight: 600;
}

.category-row-card {
  padding: 0.6rem 0.85rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.028);
  border: 1px solid rgba(var(--tint-rgb), 0.05);
  transition:
    background 160ms ease,
    border-color 160ms ease;
}

.category-row-card:hover {
  background: rgba(var(--tint-rgb), 0.05);
  border-color: rgba(143, 168, 160, 0.18);
}

.category-row-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.9rem;
}

.category-row-main {
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

.row-checkbox:disabled {
  cursor: not-allowed;
  opacity: 0.35;
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
  color: var(--text-dim, #8a939d);
  font-weight: 700;
}

.category-amount-inline {
  min-width: 0;
  flex: 1;
}

.category-inline-label {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  min-width: 0;
}

.category-value-line {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  min-width: 0;
}

.category-row-card h3 {
  margin: 0.16rem 0 0;
  color: var(--text, #eef1f3);
  font-size: 0.92rem;
  line-height: 1.15;
}

.category-count-badge {
  flex-shrink: 0;
  padding: 0.2rem 0.55rem;
  border-radius: 999px;
  background: rgba(143, 168, 160, 0.14);
  color: var(--positive-text);
  border: 1px solid rgba(143, 168, 160, 0.2);
  font-size: 0.72rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 140ms ease, background 140ms ease;
}

.category-count-badge:hover:not(:disabled) {
  transform: translateY(-1px);
  background: rgba(143, 168, 160, 0.22);
}

.category-count-badge:disabled {
  background: rgba(var(--tint-rgb), 0.06);
  color: var(--text-soft, #b3bbc4);
  border-color: rgba(var(--tint-rgb), 0.08);
  cursor: not-allowed;
}

.category-row-actions {
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

.icon-btn-danger:disabled {
  opacity: 0.4;
  cursor: not-allowed;
  pointer-events: none;
}

.eyebrow {
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.72rem;
}

@media (max-width: 720px) {
  .category-row-top {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
  }

  .category-row-actions {
    margin-left: 0;
    padding-top: 0.1rem;
  }
}
</style>
