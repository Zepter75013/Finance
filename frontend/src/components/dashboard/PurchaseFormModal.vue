<template>
  <div v-if="modelValue" class="modal-overlay" @click.self="closeModal">
    <section class="modal-card">
      <div class="modal-header">
        <div>
          <p class="eyebrow">{{ isEditMode ? 'Modifier un achat' : 'Nouvel achat' }}</p>
          <h2>{{ isEditMode ? 'Mettre à jour la dépense' : 'Ajouter une dépense' }}</h2>
        </div>

        <button class="ghost-btn" type="button" @click="closeModal" :disabled="isSubmitting">
          Fermer
        </button>
      </div>

      <form class="purchase-form" @submit.prevent="submitForm">
        <label class="form-field">
          <span>Commerçant</span>
          <input
            v-model.trim="form.merchant"
            type="text"
            placeholder="Ex: Monoprix"
            required
          />
        </label>

        <label class="form-field">
          <span>Catégorie</span>
          <select :value="form.categoryId" @change="handleCategoryChange" required>
            <option :value="null">Choisir une catégorie</option>
            <option
              v-for="category in availableCategories"
              :key="category.id"
              :value="category.id"
            >
              {{ category.name }}
            </option>
            <option value="__new__">+ Créer une nouvelle catégorie…</option>
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
          <span>Montant ({{ currency }})</span>
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
          <span>Date</span>
          <input v-model="form.date" type="date" required />
        </label>

        <label class="form-field">
          <span>Moyen de paiement</span>
          <select v-model="form.paymentMethod" required>
            <option value="Carte bancaire">Carte bancaire</option>
            <option value="Visa">Visa</option>
            <option value="Mastercard">Mastercard</option>
            <option value="Espèces">Espèces</option>
            <option value="Virement">Virement</option>
            <option value="Prélèvement">Prélèvement</option>
          </select>
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
          <span>Sous-catégorie</span>
          <select :value="form.subCategory" @change="handleSubCategoryChange" :disabled="!form.categoryId">
            <option value="">Aucune sous-catégorie</option>
            <option v-for="sub in subCategoriesForCategory" :key="sub.id" :value="sub.name">
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
            <span>C'est en réalité un virement vers un autre compte</span>
          </label>

          <label v-if="isTransfer" class="form-field transfer-account-field">
            <span>Compte destination</span>
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
                    : "Ajouter l'achat"
            }}
          </button>
        </div>
      </form>
    </section>

    <QuickCreateModal
      v-model="isCategoryModalOpen"
      title="Nouvelle catégorie"
      label="Nom de la catégorie"
      placeholder="Ex: Alimentation"
      confirm-label="Créer la catégorie"
      :is-submitting="isCreatingCategoryInline"
      :error-message="categoryModalError"
      @confirm="confirmCreateCategory"
    />

    <QuickCreateModal
      v-model="isSubCategoryModalOpen"
      title="Nouvelle sous-catégorie"
      label="Nom de la sous-catégorie"
      placeholder="Ex: Hyper/supermarché"
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
import { currencySymbol } from '../../utils/format'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  purchase: {
    type: Object,
    default: null,
  },
})

const emit = defineEmits(['update:modelValue', 'saved'])

const store = usePurchasesStore()
const storeRefs = storeToRefs(store)
const { subCategoriesList, accountsList, activeAccountId } = storeRefs

const isSubmitting = ref(false)
const submitError = ref('')

const isCategoryModalOpen = ref(false)
const isCreatingCategoryInline = ref(false)
const categoryModalError = ref('')

const isSubCategoryModalOpen = ref(false)
const isCreatingSubCategoryInline = ref(false)
const subCategoryModalError = ref('')

const isEditMode = computed(() => Boolean(props.purchase?.id))
const currency = computed(() => currencySymbol())

const isTransfer = ref(false)
const transferAccountId = ref(null)

const otherAccountsList = computed(() =>
  accountsList.value.filter((account) => Number(account.id) !== Number(form.accountId))
)

const availableCategories = computed(() => {
  return (storeRefs.categoriesList?.value ?? storeRefs.categories?.value ?? []).filter(
    (category) => (category.type || 'achat') === 'achat'
  )
})

const subCategoriesForCategory = computed(() => {
  if (!form.categoryId) return []

  return subCategoriesList.value
    .filter((sub) => Number(sub.categoryId) === Number(form.categoryId))
    .sort((a, b) => a.name.localeCompare(b.name, 'fr', { sensitivity: 'base' }))
})

const initialForm = () => ({
  merchant: props.purchase?.merchant ?? '',
  categoryId: props.purchase?.categoryId ?? availableCategories.value?.[0]?.id ?? null,
  accountId: props.purchase?.accountId ?? activeAccountId.value ?? accountsList.value?.[0]?.id ?? null,
  amount: props.purchase?.amount ?? null,
  date: props.purchase?.date ?? new Date().toISOString().slice(0, 10),
  paymentMethod: props.purchase?.paymentMethod ?? 'Carte bancaire',
  note: props.purchase?.note ?? '',
  reference: props.purchase?.reference ?? '',
  operationLabel: props.purchase?.operationLabel ?? '',
  additionalInfo: props.purchase?.additionalInfo ?? '',
  subCategory: props.purchase?.subCategory ?? '',
  operationDate: props.purchase?.operationDate ?? '',
  valueDate: props.purchase?.valueDate ?? '',
  isReconciled: props.purchase?.isReconciled ?? false,
  statementReference: props.purchase?.statementReference ?? '',
})

const form = reactive(initialForm())

watch(
  () => [props.modelValue, props.purchase, availableCategories.value],
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
    event.target.value = form.categoryId ?? ''
    categoryModalError.value = ''
    isCategoryModalOpen.value = true
    return
  }

  form.categoryId = rawValue ? Number(rawValue) : null
  form.subCategory = ''
}

async function confirmCreateCategory(name) {
  isCreatingCategoryInline.value = true
  categoryModalError.value = ''

  try {
    const created = await store.createCategory({ name, type: 'achat' })
    form.categoryId = created.id
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
    if (!form.categoryId) return
    subCategoryModalError.value = ''
    isSubCategoryModalOpen.value = true
    return
  }

  form.subCategory = rawValue
}

async function confirmCreateSubCategory(name) {
  if (!form.categoryId) return

  isCreatingSubCategoryInline.value = true
  subCategoryModalError.value = ''

  try {
    const created = await store.createSubCategory(form.categoryId, name)
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
      merchant: form.merchant,
      categoryId: form.categoryId,
      accountId: form.accountId,
      amount: form.amount,
      date: form.date,
      paymentMethod: form.paymentMethod,
      note: form.note,
      reference: form.reference,
      operationLabel: form.operationLabel,
      additionalInfo: form.additionalInfo,
      subCategory: form.subCategory,
      operationDate: form.operationDate,
      valueDate: form.valueDate,
      isReconciled: form.isReconciled,
      statementReference: form.isReconciled ? form.statementReference : '',
    }

    if (isEditMode.value && isTransfer.value && transferAccountId.value) {
      // Un achat sort toujours de son propre compte (débit) vers le compte
      // choisi (crédit) — copie de la ligne (avec ses modifications) gardée
      // dans origin_payload pour pouvoir annuler le virement plus tard (voir
      // store.undoTransfer) et pour l'affichage dans Pointage (catégorie/
      // référence/dates), le nom de catégorie n'étant pas dans le payload API.
      const categoryName = availableCategories.value.find(
        (category) => Number(category.id) === Number(payload.categoryId)
      )?.name || ''

      await store.createTransfer({
        fromAccountId: payload.accountId,
        toAccountId: transferAccountId.value,
        amount: payload.amount,
        date: payload.date,
        note: payload.merchant,
        fromIsReconciled: payload.isReconciled,
        fromStatementReference: payload.statementReference,
        toIsReconciled: false,
        toStatementReference: '',
        originType: 'achat',
        originPayload: JSON.stringify({ ...payload, category: categoryName }),
      })

      await store.removePurchase(props.purchase.id)
      emit('saved', { type: 'transfer', merchant: payload.merchant })
    } else if (isEditMode.value) {
      await store.editPurchase(props.purchase.id, payload)
      emit('saved', { type: 'edit', merchant: payload.merchant })
    } else {
      await store.submitPurchase(payload)
      emit('saved', { type: 'create', merchant: payload.merchant })
    }

    emit('update:modelValue', false)
  } catch (err) {
    submitError.value =
      err instanceof Error
        ? err.message
        : isEditMode.value
          ? 'Impossible de modifier l’achat.'
          : 'Impossible de créer l’achat.'
  } finally {
    isSubmitting.value = false
  }
}
</script>
