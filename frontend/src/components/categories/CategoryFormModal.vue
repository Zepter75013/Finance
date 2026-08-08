<template>
  <div v-if="modelValue" class="modal-overlay" @click.self="closeModal">
    <section class="modal-card">
      <div class="modal-header">
        <div>
          <p class="eyebrow">
            {{ isEditMode ? 'Modifier une catégorie' : 'Nouvelle catégorie' }}
          </p>
          <h2>
            {{ isEditMode ? 'Mettre à jour la catégorie' : 'Ajouter une catégorie' }}
          </h2>
        </div>

        <button
          class="ghost-btn"
          type="button"
          @click="closeModal"
          :disabled="isSubmitting"
        >
          Fermer
        </button>
      </div>

      <form class="category-form" @submit.prevent="submitForm">
        <label class="form-field">
          <span>Nom de la catégorie</span>
          <input
            v-model.trim="form.name"
            type="text"
            placeholder="Ex: Alimentation"
            maxlength="40"
            required
          />
        </label>

        <label class="form-field">
          <span>Description</span>
          <textarea
            v-model.trim="form.description"
            rows="4"
            placeholder="Décrire l’usage de cette catégorie"
          />
        </label>

        <label v-if="!isEditMode" class="form-field">
          <span>Type</span>
          <select v-model="form.type">
            <option value="achat">Dépense (achats)</option>
            <option value="revenu">Revenu</option>
          </select>
        </label>

        <p v-else class="category-type-badge">
          Type : <strong>{{ form.type === 'revenu' ? 'Revenu' : 'Dépense (achats)' }}</strong>
        </p>

        <p v-if="submitError" class="form-error">
          {{ submitError }}
        </p>

        <div v-if="isEditMode" class="subcategory-section">
          <p class="eyebrow">Sous-catégories</p>

          <div v-if="categorySubCategories.length" class="subcategory-list">
            <span
              v-for="sub in categorySubCategories"
              :key="sub.id"
              class="subcategory-chip"
              :class="{ 'is-editing': editingSubCategoryId === sub.id }"
            >
              <template v-if="editingSubCategoryId === sub.id">
                <input
                  v-model.trim="editingSubCategoryName"
                  type="text"
                  class="subcategory-edit-input"
                  maxlength="40"
                  autofocus
                  :disabled="isSavingSubCategoryEdit"
                  @keydown.enter.prevent="handleSaveSubCategoryEdit(sub)"
                  @keydown.escape.prevent="cancelEditSubCategory"
                />
                <button
                  type="button"
                  class="subcategory-confirm"
                  :disabled="isSavingSubCategoryEdit || !editingSubCategoryName.trim()"
                  aria-label="Valider le nouveau nom"
                  title="Valider"
                  @click="handleSaveSubCategoryEdit(sub)"
                >
                  ✓
                </button>
                <button
                  type="button"
                  class="subcategory-remove"
                  :disabled="isSavingSubCategoryEdit"
                  aria-label="Annuler"
                  title="Annuler"
                  @click="cancelEditSubCategory"
                >
                  ×
                </button>
              </template>
              <template v-else>
                {{ sub.name }}
                <button
                  type="button"
                  class="subcategory-edit"
                  :disabled="removingSubCategoryId === sub.id"
                  aria-label="Modifier cette sous-catégorie"
                  title="Modifier"
                  @click="startEditSubCategory(sub)"
                >
                  ✏️
                </button>
                <button
                  type="button"
                  class="subcategory-remove"
                  :disabled="removingSubCategoryId === sub.id"
                  aria-label="Supprimer cette sous-catégorie"
                  title="Supprimer"
                  @click="handleDeleteSubCategory(sub)"
                >
                  ×
                </button>
              </template>
            </span>
          </div>

          <p v-else class="subcategory-empty">Aucune sous-catégorie pour le moment.</p>

          <div class="subcategory-add-row">
            <input
              v-model.trim="newSubCategoryName"
              type="text"
              placeholder="Nouvelle sous-catégorie"
              @keydown.enter.prevent="handleAddSubCategory"
            />
            <button
              type="button"
              class="ghost-btn subcategory-add-btn"
              :disabled="isAddingSubCategory || !newSubCategoryName.trim()"
              @click="handleAddSubCategory"
            >
              {{ isAddingSubCategory ? 'Ajout...' : '+ Ajouter' }}
            </button>
          </div>

          <p v-if="subCategoryError" class="form-error">{{ subCategoryError }}</p>
        </div>

        <div class="modal-actions">
          <button
            class="ghost-btn"
            type="button"
            @click="closeModal"
            :disabled="isSubmitting"
          >
            Annuler
          </button>

          <button class="primary-btn" type="submit" :disabled="isSubmitting">
            {{
              isSubmitting
                ? 'Enregistrement...'
                : isEditMode
                  ? 'Enregistrer les modifications'
                  : 'Ajouter la catégorie'
            }}
          </button>
        </div>
      </form>
    </section>
  </div>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { usePurchasesStore } from '../../stores/purchases'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  category: {
    type: Object,
    default: null,
  },
  defaultType: {
    type: String,
    default: 'achat',
  },
})

const emit = defineEmits(['update:modelValue', 'saved'])

const store = usePurchasesStore()
const { subCategoriesList } = storeToRefs(store)

const isSubmitting = ref(false)
const submitError = ref('')

const isEditMode = computed(() => Boolean(props.category?.id))

const categorySubCategories = computed(() => {
  if (!props.category?.id) return []

  return subCategoriesList.value
    .filter((sub) => Number(sub.categoryId) === Number(props.category.id))
    .sort((a, b) => a.name.localeCompare(b.name, 'fr', { sensitivity: 'base' }))
})

const newSubCategoryName = ref('')
const isAddingSubCategory = ref(false)
const removingSubCategoryId = ref(null)
const subCategoryError = ref('')

const editingSubCategoryId = ref(null)
const editingSubCategoryName = ref('')
const isSavingSubCategoryEdit = ref(false)

function startEditSubCategory(sub) {
  subCategoryError.value = ''
  editingSubCategoryId.value = sub.id
  editingSubCategoryName.value = sub.name
}

function cancelEditSubCategory() {
  editingSubCategoryId.value = null
  editingSubCategoryName.value = ''
}

async function handleSaveSubCategoryEdit(sub) {
  const name = editingSubCategoryName.value.trim()
  if (!name || isSavingSubCategoryEdit.value) return

  if (name === sub.name) {
    cancelEditSubCategory()
    return
  }

  subCategoryError.value = ''
  isSavingSubCategoryEdit.value = true

  try {
    await store.editSubCategory(sub.id, name)
    cancelEditSubCategory()
  } catch (err) {
    subCategoryError.value =
      err instanceof Error ? err.message : 'Impossible de modifier la sous-catégorie.'
  } finally {
    isSavingSubCategoryEdit.value = false
  }
}

async function handleAddSubCategory() {
  const name = newSubCategoryName.value.trim()
  if (!name || !props.category?.id || isAddingSubCategory.value) return

  subCategoryError.value = ''
  isAddingSubCategory.value = true

  try {
    await store.createSubCategory(props.category.id, name)
    newSubCategoryName.value = ''
  } catch (err) {
    subCategoryError.value =
      err instanceof Error ? err.message : 'Impossible d’ajouter la sous-catégorie.'
  } finally {
    isAddingSubCategory.value = false
  }
}

async function handleDeleteSubCategory(sub) {
  subCategoryError.value = ''
  removingSubCategoryId.value = sub.id

  try {
    await store.removeSubCategory(sub.id)
  } catch (err) {
    subCategoryError.value =
      err instanceof Error ? err.message : 'Impossible de supprimer la sous-catégorie.'
  } finally {
    removingSubCategoryId.value = null
  }
}

const initialForm = () => ({
  name: props.category?.name ?? '',
  description: props.category?.description ?? '',
  type: props.category ? (props.category.type === 'revenu' ? 'revenu' : 'achat') : props.defaultType,
})

const form = reactive(initialForm())

watch(
  () => [props.modelValue, props.category],
  ([isOpen]) => {
    if (isOpen) {
      submitError.value = ''
      Object.assign(form, initialForm())
      newSubCategoryName.value = ''
      subCategoryError.value = ''
      cancelEditSubCategory()
    }
  },
  { deep: true, immediate: true }
)

function closeModal() {
  if (isSubmitting.value) {
    return
  }

  emit('update:modelValue', false)
}

async function submitForm() {
  submitError.value = ''

  if (!form.name?.trim()) {
    submitError.value = 'Le nom de la catégorie est obligatoire.'
    return
  }

  isSubmitting.value = true

  try {
    const payload = {
      name: form.name.trim(),
      description: form.description.trim(),
      type: form.type === 'revenu' ? 'revenu' : 'achat',
    }

    emit('saved', {
      type: isEditMode.value ? 'edit' : 'create',
      category: {
        ...props.category,
        ...payload,
      },
    })

    emit('update:modelValue', false)
  } catch (err) {
    submitError.value =
      err instanceof Error
        ? err.message
        : "Impossible d'enregistrer la catégorie."
  } finally {
    isSubmitting.value = false
  }
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 80;
  display: grid;
  place-items: center;
  padding: 1.25rem;
  background: var(--modal-overlay);
  backdrop-filter: blur(6px);
}

.modal-card {
  width: min(100%, 560px);
  border-radius: 24px;
  padding: 1.4rem;
  background: var(--modal-bg);
  border: 1px solid rgba(var(--tint-rgb), 0.06);
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.34);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  margin-bottom: 1.2rem;
}

.modal-header h2 {
  margin-top: 0.35rem;
  font-size: 1.55rem;
  line-height: 1.15;
  color: var(--text, #eef1f3);
}

.eyebrow {
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.74rem;
}

.category-form {
  display: grid;
  gap: 1rem;
}

.form-field {
  display: grid;
  gap: 0.45rem;
}

.form-field span {
  color: var(--text-soft, #b3bbc4);
  font-size: 0.92rem;
  font-weight: 600;
}

.form-field input,
.form-field textarea {
  width: 100%;
  border: 1px solid rgba(var(--tint-rgb), 0.07);
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.035);
  color: var(--text, #eef1f3);
  padding: 0.9rem 1rem;
  outline: none;
  transition: border-color 140ms ease, background 140ms ease;
}

.form-field input::placeholder,
.form-field textarea::placeholder {
  color: rgba(var(--tint-rgb), 0.34);
}

.form-field input:focus,
.form-field textarea:focus {
  border-color: rgba(219, 230, 223, 0.35);
  background: rgba(var(--tint-rgb), 0.05);
}

.form-field textarea {
  resize: vertical;
  min-height: 120px;
}

.form-error {
  margin: 0;
  padding: 0.85rem 1rem;
  border-radius: 14px;
  background: rgba(239, 68, 68, 0.12);
  color: var(--negative-text);
  border: 1px solid rgba(239, 68, 68, 0.18);
  font-size: 0.92rem;
}

.category-type-badge {
  margin: 0;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.88rem;
}

.category-type-badge strong {
  color: var(--text, #eef1f3);
}

.subcategory-section {
  display: grid;
  gap: 0.65rem;
  padding-top: 0.6rem;
  border-top: 1px solid rgba(var(--tint-rgb), 0.06);
}

.subcategory-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}

.subcategory-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.3rem 0.3rem 0.3rem 0.65rem;
  border-radius: 999px;
  background: rgba(var(--tint-rgb), 0.06);
  color: var(--text-soft, #b3bbc4);
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  font-size: 0.8rem;
  font-weight: 600;
}

.subcategory-chip.is-editing {
  padding-left: 0.35rem;
}

.subcategory-edit-input {
  width: 120px;
  border: 1px solid rgba(var(--tint-rgb), 0.14);
  border-radius: 999px;
  background: rgba(var(--tint-rgb), 0.06);
  color: var(--text, #eef1f3);
  padding: 0.25rem 0.6rem;
  font-size: 0.8rem;
  font-weight: 600;
  outline: none;
}

.subcategory-edit-input:focus {
  border-color: rgba(219, 230, 223, 0.4);
}

.subcategory-edit,
.subcategory-confirm,
.subcategory-remove {
  display: grid;
  place-items: center;
  width: 18px;
  height: 18px;
  border: none;
  border-radius: 999px;
  background: rgba(var(--tint-rgb), 0.08);
  color: var(--text-soft, #b3bbc4);
  font-size: 0.85rem;
  line-height: 1;
  cursor: pointer;
  transition: background 140ms ease, color 140ms ease;
  flex-shrink: 0;
}

.subcategory-edit {
  font-size: 0.68rem;
}

.subcategory-edit:hover:not(:disabled) {
  background: rgba(var(--tint-rgb), 0.14);
  color: var(--text, #eef1f3);
}

.subcategory-confirm:hover:not(:disabled) {
  background: rgba(143, 168, 160, 0.24);
  color: var(--positive-text);
}

.subcategory-remove:hover:not(:disabled) {
  background: rgba(220, 38, 38, 0.2);
  color: var(--negative-text);
}

.subcategory-edit:disabled,
.subcategory-confirm:disabled,
.subcategory-remove:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.subcategory-empty {
  margin: 0;
  color: var(--text-dim, #8a939d);
  font-size: 0.84rem;
}

.subcategory-add-row {
  display: flex;
  gap: 0.5rem;
}

.subcategory-add-row input {
  flex: 1;
  border: 1px solid rgba(var(--tint-rgb), 0.07);
  border-radius: 12px;
  background: rgba(var(--tint-rgb), 0.035);
  color: var(--text, #eef1f3);
  padding: 0.6rem 0.8rem;
  outline: none;
  font-size: 0.88rem;
}

.subcategory-add-row input:focus {
  border-color: rgba(219, 230, 223, 0.35);
  background: rgba(var(--tint-rgb), 0.05);
}

.subcategory-add-btn {
  flex-shrink: 0;
  padding: 0 0.9rem;
  font-size: 0.84rem;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.4rem;
}

.primary-btn,
.ghost-btn {
  border: none;
  border-radius: 14px;
  padding: 0.9rem 1.1rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 140ms ease, opacity 140ms ease, background 140ms ease;
}

.primary-btn:hover,
.ghost-btn:hover {
  transform: translateY(-1px);
}

.primary-btn:disabled,
.ghost-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
  transform: none;
}

.primary-btn {
  background: #dbe6df;
  color: #1c2421;
}

.ghost-btn {
  background: rgba(var(--tint-rgb), 0.06);
  color: var(--text);
}

@media (max-width: 640px) {
  .modal-card {
    padding: 1.1rem;
    border-radius: 20px;
  }

  .modal-header,
  .modal-actions {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
