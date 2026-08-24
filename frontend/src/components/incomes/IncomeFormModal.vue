<template>
  <div v-if="modelValue" class="modal-overlay" @click.self="closeModal">
    <section class="modal-card">
      <div class="modal-header">
        <div>
          <p class="eyebrow">{{ isEditMode ? 'Modifier un revenu' : 'Nouveau revenu' }}</p>
          <h2>{{ isEditMode ? 'Mettre à jour le revenu' : 'Ajouter un revenu' }}</h2>
        </div>

        <button class="ghost-btn" type="button" @click="closeModal" :disabled="isSubmitting">
          Fermer
        </button>
      </div>

      <form class="purchase-form" @submit.prevent="submitForm">
        <label class="form-field">
          <span>Source</span>
          <select v-model="form.source" required>
            <option value="">Choisir une source</option>
            <option value="Salaire">Salaire</option>
            <option value="Prime">Prime</option>
            <option value="Remboursement">Remboursement</option>
            <option value="Autre revenu">Autre revenu</option>
          </select>
        </label>

        <label class="form-field">
          <span>Compte</span>
          <select v-model.number="form.accountId" required>
            <option :value="null">Choisir un compte</option>
            <option v-for="account in accountsList" :key="account.id" :value="account.id">
              {{ account.name }}
            </option>
          </select>
        </label>

        <label class="form-field">
          <span>Montant</span>
          <input
            v-model.number="form.amount"
            type="number"
            min="0.01"
            step="0.01"
            placeholder="0.00"
            required
          />
        </label>

        <label class="form-field">
          <span>Date du revenu</span>
          <input v-model="form.income_date" type="date" required />
        </label>

        <label class="form-field form-field-full">
          <span>Note</span>
          <textarea
            v-model.trim="form.note"
            rows="4"
            placeholder="Ajouter un commentaire"
          />
        </label>

        <p class="eyebrow form-field-full form-section-label">Détails bancaires (optionnel)</p>

        <label class="form-field">
          <span>Référence</span>
          <input
            v-model.trim="form.reference"
            type="text"
            placeholder="Référence de l'opération"
            readonly
            title="Référence bancaire — non modifiable (sert à détecter les doublons à l'import)"
          />
        </label>

        <label class="form-field">
          <span>Type d'opération</span>
          <input v-model.trim="form.operationType" type="text" placeholder="Ex: Virement reçu" />
        </label>

        <label class="form-field">
          <span>Catégorie</span>
          <select :value="form.category" @change="handleCategoryChange">
            <option value="">Choisir une catégorie</option>
            <option v-for="cat in revenuCategories" :key="cat.id" :value="cat.name">
              {{ cat.name }}
            </option>
            <option value="__new__">+ Créer une nouvelle catégorie…</option>
          </select>
        </label>

        <label class="form-field">
          <span>Sous-catégorie</span>
          <select :value="form.subCategory" @change="handleSubCategoryChange" :disabled="!selectedCategoryId">
            <option value="">Aucune sous-catégorie</option>
            <option v-for="sub in matchingSubCategories" :key="sub.id" :value="sub.name">
              {{ sub.name }}
            </option>
            <option value="__new__">+ Créer une nouvelle sous-catégorie…</option>
          </select>
        </label>

        <label class="form-field form-field-full">
          <span>Libellé opération</span>
          <input v-model.trim="form.operationLabel" type="text" placeholder="Libellé complet de l'opération bancaire" />
        </label>

        <label class="form-field form-field-full">
          <span>Informations complémentaires</span>
          <textarea
            v-model.trim="form.additionalInfo"
            rows="2"
            placeholder="Informations complémentaires de la banque"
          />
        </label>

        <label class="form-field">
          <span>Date opération</span>
          <input v-model="form.operationDate" type="date" />
        </label>

        <label class="form-field">
          <span>Date de valeur</span>
          <input v-model="form.valueDate" type="date" />
        </label>

        <label class="form-field form-field-full form-field-checkbox">
          <input v-model="form.isReconciled" type="checkbox" />
          <span>Opération pointée</span>
        </label>

        <label v-if="form.isReconciled" class="form-field form-field-full">
          <span>Numéro de relevé</span>
          <input v-model.trim="form.statementReference" type="text" placeholder="Ex: 2026-01" />
        </label>

        <div v-if="isEditMode" class="form-field form-field-full transfer-section">
          <label class="transfer-toggle">
            <input v-model="isTransfer" type="checkbox" />
            <span>C'est en réalité un virement depuis un autre compte</span>
          </label>

          <label v-if="isTransfer" class="form-field transfer-account-field">
            <span>Compte source</span>
            <select v-model.number="transferAccountId">
              <option :value="null">Choisir un compte</option>
              <option v-for="account in otherAccountsList" :key="account.id" :value="account.id">
                {{ account.name }}
              </option>
            </select>
          </label>
        </div>

        <p v-if="submitError" class="form-error">
          {{ submitError }}
        </p>

        <div class="modal-actions">
          <button class="ghost-btn" type="button" @click="closeModal" :disabled="isSubmitting">
            Annuler
          </button>
          <button
            class="primary-btn"
            type="submit"
            :disabled="isSubmitting || (isTransfer && !transferAccountId)"
          >
            {{
              isSubmitting
                ? 'Enregistrement...'
                : isTransfer
                  ? 'Transférer'
                  : isEditMode
                    ? 'Enregistrer les modifications'
                    : 'Ajouter le revenu'
            }}
          </button>
        </div>
      </form>
    </section>

    <QuickCreateModal
      v-model="isCategoryModalOpen"
      title="Nouvelle catégorie"
      label="Nom de la catégorie"
      placeholder="Ex: Salaire"
      confirm-label="Créer la catégorie"
      :is-submitting="isCreatingCategoryInline"
      :error-message="categoryModalError"
      @confirm="confirmCreateCategory"
    />

    <QuickCreateModal
      v-model="isSubCategoryModalOpen"
      title="Nouvelle sous-catégorie"
      label="Nom de la sous-catégorie"
      placeholder="Ex: Salaire net"
      confirm-label="Créer la sous-catégorie"
      :is-submitting="isCreatingSubCategoryInline"
      :error-message="subCategoryModalError"
      @confirm="confirmCreateSubCategory"
    />
  </div>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { usePurchasesStore } from '../../stores/purchases'
import QuickCreateModal from '../Common/QuickCreateModal.vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  income: {
    type: Object,
    default: null,
  },
})

const emit = defineEmits(['update:modelValue', 'saved'])

const store = usePurchasesStore()
const { categoriesList, subCategoriesList, accountsList, activeAccountId } = storeToRefs(store)

const isSubmitting = ref(false)
const submitError = ref('')

const isCategoryModalOpen = ref(false)
const isCreatingCategoryInline = ref(false)
const categoryModalError = ref('')

const isSubCategoryModalOpen = ref(false)
const isCreatingSubCategoryInline = ref(false)
const subCategoryModalError = ref('')

const isEditMode = computed(() => Boolean(props.income?.id))

const isTransfer = ref(false)
const transferAccountId = ref(null)

const otherAccountsList = computed(() =>
  accountsList.value.filter((account) => Number(account.id) !== Number(form.accountId))
)

const revenuCategories = computed(() =>
  categoriesList.value
    .filter((category) => category.type === 'revenu')
    .sort((a, b) => a.name.localeCompare(b.name, 'fr', { sensitivity: 'base' }))
)

const selectedCategoryId = computed(() => {
  const selected = revenuCategories.value.find(
    (category) => category.name.toLowerCase() === (form.category || '').trim().toLowerCase()
  )

  return selected ? selected.id : null
})

const matchingSubCategories = computed(() => {
  const list = selectedCategoryId.value
    ? subCategoriesList.value.filter((sub) => Number(sub.categoryId) === Number(selectedCategoryId.value))
    : []

  return list.slice().sort((a, b) => a.name.localeCompare(b.name, 'fr', { sensitivity: 'base' }))
})

const initialForm = () => ({
  source: props.income?.source ?? '',
  accountId: props.income?.accountId ?? activeAccountId.value ?? accountsList.value?.[0]?.id ?? null,
  amount: props.income?.amount ?? null,
  income_date: props.income?.income_date ?? new Date().toISOString().slice(0, 10),
  note: props.income?.note ?? '',
  reference: props.income?.reference ?? '',
  operationLabel: props.income?.operationLabel ?? '',
  additionalInfo: props.income?.additionalInfo ?? '',
  operationType: props.income?.operationType ?? '',
  category: props.income?.category ?? '',
  subCategory: props.income?.subCategory ?? '',
  operationDate: props.income?.operationDate ?? '',
  valueDate: props.income?.valueDate ?? '',
  isReconciled: props.income?.isReconciled ?? false,
  statementReference: props.income?.statementReference ?? '',
})

const form = reactive(initialForm())

watch(
  () => [props.modelValue, props.income],
  ([isOpen]) => {
    if (isOpen) {
      submitError.value = ''
      Object.assign(form, initialForm())
      isCategoryModalOpen.value = false
      isSubCategoryModalOpen.value = false
      categoryModalError.value = ''
      subCategoryModalError.value = ''
      isTransfer.value = false
      transferAccountId.value = null
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

function handleCategoryChange(event) {
  const rawValue = event.target.value

  if (rawValue === '__new__') {
    event.target.value = form.category || ''
    categoryModalError.value = ''
    isCategoryModalOpen.value = true
    return
  }

  form.category = rawValue
  form.subCategory = ''
}

async function confirmCreateCategory(name) {
  isCreatingCategoryInline.value = true
  categoryModalError.value = ''

  try {
    const created = await store.createCategory({ name, type: 'revenu' })
    form.category = created.name
    form.subCategory = ''
    isCategoryModalOpen.value = false
  } catch (err) {
    categoryModalError.value = err instanceof Error ? err.message : 'Impossible de créer la catégorie.'
  } finally {
    isCreatingCategoryInline.value = false
  }
}

function handleSubCategoryChange(event) {
  const rawValue = event.target.value

  if (rawValue === '__new__') {
    event.target.value = form.subCategory || ''
    if (!selectedCategoryId.value) return
    subCategoryModalError.value = ''
    isSubCategoryModalOpen.value = true
    return
  }

  form.subCategory = rawValue
}

async function confirmCreateSubCategory(name) {
  if (!selectedCategoryId.value) return

  isCreatingSubCategoryInline.value = true
  subCategoryModalError.value = ''

  try {
    const created = await store.createSubCategory(selectedCategoryId.value, name)
    form.subCategory = created.name
    isSubCategoryModalOpen.value = false
  } catch (err) {
    subCategoryModalError.value = err instanceof Error ? err.message : 'Impossible de créer la sous-catégorie.'
  } finally {
    isCreatingSubCategoryInline.value = false
  }
}

async function submitForm() {
  submitError.value = ''
  isSubmitting.value = true

  try {
    const payload = {
      source: form.source,
      accountId: form.accountId,
      amount: form.amount,
      income_date: form.income_date,
      note: form.note,
      reference: form.reference,
      operationLabel: form.operationLabel,
      additionalInfo: form.additionalInfo,
      operationType: form.operationType,
      category: form.category,
      subCategory: form.subCategory,
      operationDate: form.operationDate,
      valueDate: form.valueDate,
      isReconciled: form.isReconciled,
      statementReference: form.isReconciled ? form.statementReference : '',
    }

    if (isEditMode.value && isTransfer.value && transferAccountId.value) {
      // Un revenu vient toujours du compte choisi (débit) vers son propre
      // compte (crédit) — sens inverse d'un achat. Copie de la ligne (avec
      // ses modifications) gardée dans origin_payload pour l'annulation et
      // l'affichage dans Pointage (voir même logique côté achat).
      await store.createTransfer({
        fromAccountId: transferAccountId.value,
        toAccountId: payload.accountId,
        amount: payload.amount,
        date: payload.income_date,
        note: payload.source,
        fromIsReconciled: false,
        fromStatementReference: '',
        toIsReconciled: payload.isReconciled,
        toStatementReference: payload.statementReference,
        originType: 'revenu',
        originPayload: JSON.stringify(payload),
      })

      await store.removeIncome(props.income.id)
      emit('saved', { type: 'transfer', source: payload.source })
    } else if (isEditMode.value) {
      await store.editIncome(props.income.id, payload)
      emit('saved', { type: 'edit', source: payload.source })
    } else {
      await store.createIncome(payload)
      emit('saved', { type: 'create', source: payload.source })
    }

    emit('update:modelValue', false)
  } catch (err) {
    submitError.value =
      err instanceof Error
        ? err.message
        : isEditMode.value
          ? 'Impossible de modifier le revenu.'
          : 'Impossible de créer le revenu.'
  } finally {
    isSubmitting.value = false
  }
}
</script>
