<script setup>
import { computed, ref } from 'vue'
import { storeToRefs } from 'pinia'
import PageHero from '../Common/PageHero.vue'
import { usePurchasesStore } from '../../stores/purchases'
import { parseCsv } from '../../utils/csv'

const store = usePurchasesStore()
const { categoriesList, subCategoriesList } = storeToRefs(store)

const CATEGORY_KEYWORDS = ['categorie', 'catégorie', 'category']
const SUB_CATEGORY_KEYWORDS = ['sous categorie', 'sous-categorie', 'sous catégorie', 'souscategorie']
const TYPE_KEYWORDS = ['type']

const step = ref('upload')
const fileName = ref('')
const fileError = ref('')
const isDragging = ref(false)
const isImporting = ref(false)
const importResult = ref(null)

const groups = ref([])

function normalizeName(value) {
  return String(value ?? '')
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .trim()
}

function guessColumn(headers, keywords) {
  const normalizedHeaders = headers.map(normalizeName)

  for (const keyword of keywords) {
    const index = normalizedHeaders.findIndex((header) => header.includes(keyword))
    if (index !== -1) return headers[index]
  }

  return ''
}

function resolveType(raw) {
  const normalized = normalizeName(raw)
  return normalized.includes('revenu') ? 'revenu' : 'achat'
}

function findExistingCategory(name) {
  const normalized = normalizeName(name)
  return categoriesList.value.find((category) => normalizeName(category.name) === normalized) || null
}

function findExistingSubCategory(categoryId, name) {
  const normalized = normalizeName(name)
  return (
    subCategoriesList.value.find(
      (sc) => sc.categoryId === categoryId && normalizeName(sc.name) === normalized
    ) || null
  )
}

function resetState() {
  step.value = 'upload'
  fileName.value = ''
  fileError.value = ''
  isDragging.value = false
  isImporting.value = false
  importResult.value = null
  groups.value = []
}

async function decodeFileText(file) {
  const buffer = await file.arrayBuffer()

  try {
    return new TextDecoder('windows-1252', { fatal: true }).decode(buffer)
  } catch {
    return new TextDecoder('utf-8').decode(buffer)
  }
}

async function handleFile(file) {
  fileError.value = ''

  if (!file) return

  try {
    const text = await decodeFileText(file)
    const { headers, rows } = parseCsv(text)

    if (!headers.length || !rows.length) {
      fileError.value = 'Le fichier semble vide ou illisible.'
      return
    }

    const categoryColumn = guessColumn(headers, CATEGORY_KEYWORDS)
    const subCategoryColumn = guessColumn(headers, SUB_CATEGORY_KEYWORDS)
    const typeColumn = guessColumn(headers, TYPE_KEYWORDS)

    if (!categoryColumn || !subCategoryColumn) {
      fileError.value = "Colonnes attendues introuvables : « Categorie » et « Sous categorie »."
      return
    }

    const categoryIndex = headers.indexOf(categoryColumn)
    const subCategoryIndex = headers.indexOf(subCategoryColumn)
    const typeIndex = typeColumn ? headers.indexOf(typeColumn) : -1

    const byCategory = new Map()

    for (const row of rows) {
      const categoryName = row[categoryIndex]?.trim()
      const subCategoryName = row[subCategoryIndex]?.trim()

      if (!categoryName || !subCategoryName) continue

      const key = normalizeName(categoryName)

      if (!byCategory.has(key)) {
        const existingCategory = findExistingCategory(categoryName)
        const type = existingCategory
          ? existingCategory.type === 'revenu' ? 'revenu' : 'achat'
          : resolveType(typeIndex !== -1 ? row[typeIndex] : '')

        byCategory.set(key, {
          name: categoryName,
          type,
          existingCategoryId: existingCategory?.id ?? null,
          subCategories: new Map(),
        })
      }

      const group = byCategory.get(key)
      const subKey = normalizeName(subCategoryName)

      if (!group.subCategories.has(subKey)) {
        const existingSub = group.existingCategoryId
          ? findExistingSubCategory(group.existingCategoryId, subCategoryName)
          : null

        group.subCategories.set(subKey, {
          name: subCategoryName,
          alreadyExists: Boolean(existingSub),
        })
      }
    }

    groups.value = Array.from(byCategory.values()).map((group) => ({
      ...group,
      subCategories: Array.from(group.subCategories.values()),
    }))

    fileName.value = file.name
    step.value = 'preview'
  } catch (err) {
    fileError.value = err instanceof Error ? err.message : 'Impossible de lire ce fichier.'
  }
}

function handleFileInput(event) {
  const file = event.target?.files?.[0]
  handleFile(file)
}

function handleDrop(event) {
  isDragging.value = false
  const file = event.dataTransfer?.files?.[0]
  handleFile(file)
}

const depenseGroups = computed(() => groups.value.filter((g) => g.type !== 'revenu'))
const revenuGroups = computed(() => groups.value.filter((g) => g.type === 'revenu'))

const summary = computed(() => {
  const newCategories = groups.value.filter((g) => !g.existingCategoryId).length
  const totalSubCategories = groups.value.reduce((sum, g) => sum + g.subCategories.length, 0)
  const newSubCategories = groups.value.reduce(
    (sum, g) => sum + g.subCategories.filter((sc) => !sc.alreadyExists).length,
    0
  )

  return {
    totalCategories: groups.value.length,
    newCategories,
    totalSubCategories,
    newSubCategories,
    skippedSubCategories: totalSubCategories - newSubCategories,
  }
})

async function confirmImport() {
  isImporting.value = true

  let createdCategories = 0
  let createdSubCategories = 0
  let skippedSubCategories = 0
  let failCount = 0

  for (const group of groups.value) {
    let categoryId = group.existingCategoryId

    try {
      if (!categoryId) {
        const created = await store.createCategory({ name: group.name, type: group.type })
        categoryId = created.id
        createdCategories += 1
      }
    } catch {
      failCount += group.subCategories.length
      continue
    }

    for (const subCategory of group.subCategories) {
      try {
        const wasNew = !subCategory.alreadyExists
        await store.createSubCategory(categoryId, subCategory.name)

        if (wasNew) createdSubCategories += 1
        else skippedSubCategories += 1
      } catch {
        failCount += 1
      }
    }
  }

  isImporting.value = false
  importResult.value = { createdCategories, createdSubCategories, skippedSubCategories, failCount }
  step.value = 'done'
}
</script>

<template>
  <main class="dashboard-content import-view">
    <PageHero
      eyebrow="Récupérer des données"
      title="Import des catégories"
      description="Importe un fichier CSV de catégories et sous-catégories pour organiser tes achats."
    >
      <template #actions>
        <button
          v-if="step !== 'upload'"
          class="ghost-btn"
          type="button"
          :disabled="isImporting"
          @click="resetState"
        >
          Recommencer
        </button>
      </template>
    </PageHero>

    <section class="panel import-card">
      <section v-if="step === 'upload'" class="import-step">
        <label
          class="import-dropzone"
          :class="{ 'is-dragging': isDragging }"
          @dragover.prevent="isDragging = true"
          @dragleave.prevent="isDragging = false"
          @drop.prevent="handleDrop"
        >
          <input type="file" accept=".csv,text/csv" @change="handleFileInput" />
          <span class="import-dropzone-title">Dépose ton fichier CSV ici</span>
          <span class="import-dropzone-caption">
            Colonnes attendues : « Categorie » et « Sous categorie » (+ « Type » optionnel : Depense/Revenu)
          </span>
        </label>

        <p v-if="fileError" class="form-error">{{ fileError }}</p>
      </section>

      <section v-else-if="step === 'preview'" class="import-step">
        <p class="import-step-caption">
          <strong>{{ fileName }}</strong> ·
          {{ summary.totalCategories }} catégorie{{ summary.totalCategories > 1 ? 's' : '' }}
          ({{ summary.newCategories }} nouvelle{{ summary.newCategories > 1 ? 's' : '' }}) ·
          {{ summary.totalSubCategories }} sous-catégorie{{ summary.totalSubCategories > 1 ? 's' : '' }}
          ({{ summary.newSubCategories }} nouvelle{{ summary.newSubCategories > 1 ? 's' : '' }},
          {{ summary.skippedSubCategories }} déjà existante{{ summary.skippedSubCategories > 1 ? 's' : '' }}).
        </p>

        <div v-if="depenseGroups.length" class="import-preview-section">
          <p class="import-preview-section-title">Dépenses ({{ depenseGroups.length }})</p>

          <div class="import-preview-list">
            <article v-for="group in depenseGroups" :key="group.name" class="import-preview-group">
              <div class="import-preview-group-head">
                <strong>{{ group.name }}</strong>
                <span v-if="!group.existingCategoryId" class="import-tag import-tag-new">Nouvelle catégorie</span>
                <span v-else class="import-tag">Catégorie existante</span>
              </div>

              <div class="import-preview-subs">
                <span
                  v-for="sub in group.subCategories"
                  :key="sub.name"
                  class="import-tag"
                  :class="{ 'import-tag-new': !sub.alreadyExists }"
                >
                  {{ sub.name }}
                </span>
              </div>
            </article>
          </div>
        </div>

        <div v-if="revenuGroups.length" class="import-preview-section">
          <p class="import-preview-section-title">Revenus ({{ revenuGroups.length }})</p>

          <div class="import-preview-list">
            <article v-for="group in revenuGroups" :key="group.name" class="import-preview-group">
              <div class="import-preview-group-head">
                <strong>{{ group.name }}</strong>
                <span v-if="!group.existingCategoryId" class="import-tag import-tag-new">Nouvelle catégorie</span>
                <span v-else class="import-tag">Catégorie existante</span>
              </div>

              <div class="import-preview-subs">
                <span
                  v-for="sub in group.subCategories"
                  :key="sub.name"
                  class="import-tag"
                  :class="{ 'import-tag-new': !sub.alreadyExists }"
                >
                  {{ sub.name }}
                </span>
              </div>
            </article>
          </div>
        </div>

        <div class="modal-actions">
          <button class="ghost-btn" type="button" @click="step = 'upload'" :disabled="isImporting">
            Retour
          </button>
          <button class="primary-btn" type="button" :disabled="isImporting" @click="confirmImport">
            {{ isImporting ? 'Import en cours…' : 'Importer' }}
          </button>
        </div>
      </section>

      <section v-else class="import-step">
        <p class="import-step-caption">
          {{ importResult.createdCategories }} catégorie{{ importResult.createdCategories > 1 ? 's' : '' }} créée{{ importResult.createdCategories > 1 ? 's' : '' }},
          {{ importResult.createdSubCategories }} sous-catégorie{{ importResult.createdSubCategories > 1 ? 's' : '' }} créée{{ importResult.createdSubCategories > 1 ? 's' : '' }},
          {{ importResult.skippedSubCategories }} déjà existante{{ importResult.skippedSubCategories > 1 ? 's' : '' }} (ignorée{{ importResult.skippedSubCategories > 1 ? 's' : '' }}).
          <span v-if="importResult.failCount">{{ importResult.failCount }} échec{{ importResult.failCount > 1 ? 's' : '' }}.</span>
        </p>

        <div class="modal-actions">
          <button class="primary-btn" type="button" @click="resetState">Importer un autre fichier</button>
        </div>
      </section>
    </section>
  </main>
</template>

<style scoped>
.import-view {
  display: grid;
  gap: 0.9rem;
}

.import-card {
  padding: 1.1rem;
}

.import-step {
  display: grid;
  gap: 1rem;
}

.import-step-caption {
  margin: 0;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.88rem;
  line-height: 1.5;
}

.import-dropzone {
  position: relative;
  display: grid;
  place-items: center;
  gap: 0.4rem;
  padding: 2.4rem 1.2rem;
  border-radius: 18px;
  border: 1px dashed rgba(var(--tint-rgb), 0.16);
  background: rgba(var(--tint-rgb), 0.03);
  text-align: center;
  cursor: pointer;
  transition: border-color 140ms ease, background 140ms ease;
}

.import-dropzone.is-dragging {
  border-color: rgba(143, 168, 160, 0.5);
  background: rgba(143, 168, 160, 0.08);
}

.import-dropzone input[type='file'] {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}

.import-dropzone-title {
  color: var(--text, #eef1f3);
  font-weight: 600;
}

.import-dropzone-caption {
  color: var(--text-dim, #8a939d);
  font-size: 0.84rem;
}

.import-preview-section {
  display: grid;
  gap: 0.5rem;
}

.import-preview-section-title {
  margin: 0;
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-size: 0.72rem;
  font-weight: 700;
}

.import-preview-list {
  display: grid;
  gap: 0.6rem;
  max-height: 320px;
  overflow-y: auto;
  padding-right: 0.2rem;
}

.import-preview-group {
  padding: 0.7rem 0.85rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.028);
  border: 1px solid rgba(var(--tint-rgb), 0.05);
  display: grid;
  gap: 0.5rem;
}

.import-preview-group-head {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  color: var(--text, #eef1f3);
  font-size: 0.92rem;
}

.import-preview-subs {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.import-tag {
  padding: 0.2rem 0.55rem;
  border-radius: 999px;
  background: rgba(var(--tint-rgb), 0.06);
  color: var(--text-soft, #b3bbc4);
  border: 1px solid rgba(var(--tint-rgb), 0.08);
  font-size: 0.72rem;
  font-weight: 600;
}

.import-tag-new {
  background: rgba(143, 168, 160, 0.14);
  color: var(--positive-text);
  border-color: rgba(143, 168, 160, 0.24);
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

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.4rem;
}
</style>
