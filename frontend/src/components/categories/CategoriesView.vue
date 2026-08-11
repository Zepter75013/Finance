<script setup>
import { computed, reactive, ref } from 'vue'
import { storeToRefs } from 'pinia'

import PageHero from '../Common/PageHero.vue'
import ConfirmModal from '../Common/ConfirmModal.vue'
import CategoryRowCard from './CategoryRowCard.vue'
import { usePurchasesStore } from '../../stores/purchases'
import { formatCurrency } from '../../utils/format'

const props = defineProps({
  isLoading: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['create', 'edit', 'delete', 'open-purchases', 'bulk-delete'])

const purchasesStore = usePurchasesStore()

const { categoriesList, purchases, subCategoriesList } = storeToRefs(purchasesStore)

const checkedCategoryIds = reactive(new Set())
const isBulkDeleting = ref(false)
const isBulkDeleteConfirmOpen = ref(false)
const activeType = ref('achat')

function setActiveType(type) {
  if (activeType.value === type) return
  activeType.value = type
  checkedCategoryIds.clear()
}

const categoryCards = computed(() => {
  return categoriesList.value
    .filter((category) => (category.type === 'revenu' ? 'revenu' : 'achat') === activeType.value)
    .map((category) => {
    const linkedPurchases = purchases.value.filter(
      (purchase) => Number(purchase.categoryId) === Number(category.id)
    )

    const totalAmount = linkedPurchases.reduce((sum, purchase) => {
      return sum + Number(purchase.amount || 0)
    }, 0)

    const subCategories = subCategoriesList.value.filter(
      (subCategory) => Number(subCategory.categoryId) === Number(category.id)
    )

    return {
      ...category,
      purchaseCount: linkedPurchases.length,
      totalAmount,
      subCategories,
    }
  })
})

const depenseCategoryCount = computed(
  () => categoriesList.value.filter((category) => (category.type || 'achat') !== 'revenu').length
)

const revenuCategoryCount = computed(
  () => categoriesList.value.filter((category) => category.type === 'revenu').length
)

const totalCategories = computed(() => categoryCards.value.length)

const usedCategories = computed(() => {
  return categoryCards.value.filter((category) => category.purchaseCount > 0).length
})

const emptyCategories = computed(() => {
  return categoryCards.value.filter((category) => category.purchaseCount === 0).length
})

const totalTrackedAmount = computed(() => {
  return categoryCards.value.reduce((sum, category) => sum + category.totalAmount, 0)
})


function handleCreate() {
  emit('create', activeType.value)
}

function handleEdit(category) {
  emit('edit', category)
}

function handleDelete(category) {
  emit('delete', category)
}

function handleOpenPurchases(category) {
  if (!category?.name || category.purchaseCount === 0) {
    return
  }

  emit('open-purchases', category)
}

const selectableCategories = computed(() =>
  categoryCards.value.filter((category) => category.purchaseCount === 0)
)

function isCategoryChecked(id) {
  return checkedCategoryIds.has(id)
}

function toggleCategoryChecked(category) {
  if (category.purchaseCount > 0) return

  if (checkedCategoryIds.has(category.id)) checkedCategoryIds.delete(category.id)
  else checkedCategoryIds.add(category.id)
}

const hasCategorySelection = computed(() => checkedCategoryIds.size > 0)

const isAllCategoriesSelected = computed(() => {
  return (
    selectableCategories.value.length > 0 &&
    selectableCategories.value.every((category) => checkedCategoryIds.has(category.id))
  )
})

function selectAllCategories() {
  for (const category of selectableCategories.value) checkedCategoryIds.add(category.id)
}

function clearCategorySelection() {
  checkedCategoryIds.clear()
}

function toggleSelectAllCategories() {
  if (isAllCategoriesSelected.value) clearCategorySelection()
  else selectAllCategories()
}

function requestDeleteSelectedCategories() {
  if (!checkedCategoryIds.size) return
  isBulkDeleteConfirmOpen.value = true
}

async function deleteSelectedCategories() {
  const ids = Array.from(checkedCategoryIds)
  if (!ids.length) return

  isBulkDeleting.value = true

  let successCount = 0
  let failCount = 0

  for (const id of ids) {
    try {
      await purchasesStore.removeCategory(id)
      checkedCategoryIds.delete(id)
      successCount += 1
    } catch {
      failCount += 1
    }
  }

  isBulkDeleting.value = false
  isBulkDeleteConfirmOpen.value = false
  emit('bulk-delete', { successCount, failCount })
}
</script>

<template>
  <main class="dashboard-content categories-view">
    <PageHero
      eyebrow="Organisation"
      title="Catégories"
      :description="
        activeType === 'revenu'
          ? 'Gérez vos catégories de revenus pour garder une structure claire.'
          : 'Gérez vos catégories d’achats pour garder une structure claire et mieux suivre vos dépenses.'
      "
    >
      <template #actions>
        <button class="primary-btn" type="button" @click="handleCreate">
          + Nouvelle catégorie
        </button>
      </template>
    </PageHero>

    <div class="category-type-tabs">
      <button
        class="category-type-tab"
        :class="{ 'is-active': activeType === 'achat' }"
        type="button"
        @click="setActiveType('achat')"
      >
        Dépenses <span class="category-type-tab-count">{{ depenseCategoryCount }}</span>
      </button>
      <button
        class="category-type-tab"
        :class="{ 'is-active': activeType === 'revenu' }"
        type="button"
        @click="setActiveType('revenu')"
      >
        Revenus <span class="category-type-tab-count">{{ revenuCategoryCount }}</span>
      </button>
    </div>

    <section class="categories-stats-grid">
      <article class="panel stat-card">
        <p class="eyebrow">Total catégories</p>
        <h2>{{ totalCategories }}</h2>
        <p>{{ activeType === 'revenu' ? 'Base disponible pour classer les revenus' : 'Base disponible pour classer les achats' }}</p>
      </article>

      <article class="panel stat-card">
        <p class="eyebrow">Catégories utilisées</p>
        <h2>{{ usedCategories }}</h2>
        <p>Au moins un achat enregistré</p>
      </article>

      <article class="panel stat-card">
        <p class="eyebrow">Sans achat</p>
        <h2>{{ emptyCategories }}</h2>
        <p>Peuvent être nettoyées ou renommées</p>
      </article>
    </section>

    <section v-if="props.isLoading" class="panel categories-state">
      Chargement des catégories...
    </section>

    <section v-else-if="!categoryCards.length" class="panel empty-state">
      <div class="empty-state-icon">#</div>
      <h2>Aucune catégorie pour le moment</h2>
      <p>
        Crée une première catégorie pour organiser tes achats plus proprement.
      </p>
      <button class="primary-btn" type="button" @click="handleCreate">
        Créer une catégorie
      </button>
    </section>

    <section v-else class="categories-layout">
      <article class="panel categories-list-card">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Activité</p>
            <h2>Liste des catégories</h2>
          </div>
        </div>

        <div class="categories-selection-bar">
          <button class="ghost-btn" type="button" @click="toggleSelectAllCategories">
            {{ isAllCategoriesSelected ? 'Tout désélectionner' : 'Tout sélectionner' }}
          </button>

          <span v-if="hasCategorySelection" class="selection-count">
            {{ checkedCategoryIds.size }} sélectionnée{{ checkedCategoryIds.size > 1 ? 's' : '' }}
          </span>

          <button
            v-if="hasCategorySelection"
            class="danger-btn"
            type="button"
            :disabled="isBulkDeleting"
            @click="requestDeleteSelectedCategories"
          >
            {{ isBulkDeleting ? 'Suppression…' : 'Supprimer la sélection' }}
          </button>

          <span v-else class="selection-hint">
            Seules les catégories sans achat peuvent être sélectionnées.
          </span>
        </div>

        <div class="categories-list">
          <CategoryRowCard
            v-for="category in categoryCards"
            :key="category.id"
            :category="category"
            :checked="isCategoryChecked(category.id)"
            @edit="handleEdit"
            @delete="handleDelete"
            @toggle-check="toggleCategoryChecked"
            @open-purchases="handleOpenPurchases"
          />
        </div>
      </article>
    </section>

    <ConfirmModal
      v-model="isBulkDeleteConfirmOpen"
      :title="`Supprimer ${checkedCategoryIds.size} catégorie${checkedCategoryIds.size > 1 ? 's' : ''} ?`"
      :message="`Tu es sur le point de supprimer définitivement ${checkedCategoryIds.size} catégorie${checkedCategoryIds.size > 1 ? 's' : ''} sélectionnée${checkedCategoryIds.size > 1 ? 's' : ''}.`"
      note="Vérifie ta sélection avant de confirmer : cette action est irréversible."
      confirm-label="Supprimer la sélection"
      :is-processing="isBulkDeleting"
      @confirm="deleteSelectedCategories"
    />
  </main>
</template>

<style scoped>
.categories-view {
  display: grid;
  gap: 0.9rem;
}

.category-type-tabs {
  display: flex;
  gap: 0.5rem;
}

.category-type-tab {
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  border-radius: 999px;
  padding: 0.5rem 1rem;
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--text-soft, #b3bbc4);
  font-size: 0.88rem;
  font-weight: 600;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  transition: background 140ms ease, color 140ms ease, border-color 140ms ease;
}

.category-type-tab:hover {
  color: var(--text, #eef1f3);
}

.category-type-tab.is-active {
  background: rgba(219, 230, 223, 0.14);
  border-color: rgba(219, 230, 223, 0.3);
  color: #dbe6df;
}

.category-type-tab-count {
  border-radius: 999px;
  padding: 0.05rem 0.5rem;
  background: rgba(var(--tint-rgb), 0.08);
  font-size: 0.78rem;
}

.categories-stats-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.7rem;
}

.stat-card h2 {
  margin: 0.22rem 0;
}

.stat-card p:last-child {
  color: var(--text-muted, #94a3b8);
}

.categories-layout {
  display: grid;
  grid-template-columns: 1fr;
}

.categories-list-card {
  min-height: 100%;
  padding: 0.95rem;
}

.categories-list-card .panel-header {
  padding-inline: 0.15rem;
}

.categories-selection-bar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-top: 0.75rem;
  padding-top: 0.75rem;
  border-top: 1px solid rgba(var(--tint-rgb), 0.05);
}

.selection-count {
  color: var(--text-soft, #b3bbc4);
  font-size: 0.84rem;
}

.selection-hint {
  color: var(--text-dim, #8a939d);
  font-size: 0.8rem;
}

.categories-list {
  display: grid;
  gap: 0.5rem;
  margin-top: 0.75rem;
}

.ghost-btn,
.danger-btn {
  border: none;
  border-radius: 10px;
  padding: 0.55rem 0.85rem;
  font-size: 0.84rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 140ms ease, opacity 140ms ease, background 140ms ease;
}

.ghost-btn:hover,
.danger-btn:hover {
  transform: translateY(-1px);
}

.ghost-btn {
  background: rgba(var(--tint-rgb), 0.06);
  color: var(--text);
}

.danger-btn {
  background: rgba(220, 38, 38, 0.14);
  color: var(--negative-text);
  border: 1px solid rgba(220, 38, 38, 0.18);
}

.danger-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.categories-state,
.empty-state {
  padding: 2rem 1.4rem;
  text-align: center;
}

.empty-state h2 {
  margin: 0.8rem 0 0.55rem;
  color: var(--text, #eef1f3);
}

.empty-state p,
.categories-state {
  color: var(--text-soft, #b3bbc4);
}

.empty-state .primary-btn {
  margin-top: 1rem;
}

.empty-state-icon {
  width: 56px;
  height: 56px;
  margin: 0 auto;
  border-radius: 18px;
  display: grid;
  place-items: center;
  background: rgba(var(--tint-rgb), 0.06);
  color: #dbe6df;
  font-weight: 700;
  font-size: 1.1rem;
}

.eyebrow {
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.74rem;
}

@media (max-width: 1100px) {
  .categories-stats-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .categories-stats-grid {
    grid-template-columns: 1fr;
  }

  .categories-selection-bar {
    flex-wrap: wrap;
  }
}
</style>
