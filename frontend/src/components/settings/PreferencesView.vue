<script setup>
import { onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import PageHero from '../Common/PageHero.vue'
import RestoreBackupModal from './RestoreBackupModal.vue'
import { useThemeStore } from '../../stores/theme'
import { usePreferencesStore } from '../../stores/preferences'
import {
  fetchBackups,
  createBackup,
  downloadBackup,
  restoreBackup,
  restoreUploadedBackup,
  deleteBackup,
  fetchBackupSettings,
  updateBackupSettings,
  pickBackupDirectory,
} from '../../services/backups'
import { fetchDatabaseSettings, updateDatabaseSettings } from '../../services/settings'

const themeStore = useThemeStore()
const { theme } = storeToRefs(themeStore)

const THEME_OPTIONS = [
  { value: 'dark', label: 'Sombre', emoji: '🌙', description: 'Fond sombre, comme aujourd’hui.' },
  { value: 'light', label: 'Clair', emoji: '☀️', description: 'Fond clair pour un usage de jour.' },
  { value: 'system', label: 'Système', emoji: '🖥️', description: 'Suit le réglage de ton appareil.' },
]

function selectTheme(value) {
  themeStore.setTheme(value)
}

const preferencesStore = usePreferencesStore()
const { currencyCode, currencyPosition, numberFormatStyle, dateFormatStyle } = storeToRefs(preferencesStore)

const CURRENCY_OPTIONS = [
  { value: 'EUR', label: 'Euro — €' },
  { value: 'USD', label: 'Dollar US — $' },
  { value: 'GBP', label: 'Livre Sterling — £' },
  { value: 'CHF', label: 'Franc Suisse — CHF' },
]

const CURRENCY_POSITION_OPTIONS = [
  { value: 'after', label: 'Après le montant', example: '1 234,56 €' },
  { value: 'before', label: 'Avant le montant', example: '€1 234,56' },
]

const NUMBER_FORMAT_OPTIONS = [
  { value: 'fr', label: 'Français', example: '1 234,56' },
  { value: 'us', label: 'Anglais US', example: '1,234.56' },
  { value: 'ch', label: 'Suisse', example: "1'234.56" },
]

const DATE_FORMAT_OPTIONS = [
  { value: 'dmy', label: 'Jour/Mois/Année', example: '11/08/2026' },
  { value: 'mdy', label: 'Mois/Jour/Année', example: '08/11/2026' },
  { value: 'iso', label: 'Année-Mois-Jour', example: '2026-08-11' },
]

const backups = ref([])
const isLoadingBackups = ref(false)
const isCreatingBackup = ref(false)
const isRestoring = ref(false)
const backupError = ref('')
const backupSuccess = ref('')
const restoreTarget = ref(null)
const isRestoreModalOpen = ref(false)
const uploadedFile = ref(null)
const fileInput = ref(null)

const backupDirectory = ref('')
const backupDirectoryInput = ref('')
const isSavingDirectory = ref(false)
const isPickingDirectory = ref(false)

const autoBackupEnabled = ref(true)
const retentionDaysInput = ref(30)
const isSavingAutoBackup = ref(false)

async function loadBackupSettings() {
  try {
    const settings = await fetchBackupSettings()
    backupDirectory.value = settings.directory
    backupDirectoryInput.value = settings.directory
    autoBackupEnabled.value = settings.auto_backup_enabled
    retentionDaysInput.value = settings.retention_days
  } catch (err) {
    backupError.value = err instanceof Error ? err.message : 'Impossible de charger le dossier de sauvegarde.'
  }
}

// Envoie toujours l'état réactif complet des trois réglages (directory,
// autoBackupEnabled, retentionDays) — le backend réécrit le fichier de
// configuration en entier à chaque appel, donc n'envoyer que le champ
// modifié réinitialiserait silencieusement les autres.
async function handleSaveDirectory() {
  if (isSavingDirectory.value) return

  isSavingDirectory.value = true
  backupError.value = ''
  backupSuccess.value = ''

  try {
    const settings = await updateBackupSettings({
      directory: backupDirectoryInput.value,
      autoBackupEnabled: autoBackupEnabled.value,
      retentionDays: retentionDaysInput.value,
    })
    backupDirectory.value = settings.directory
    backupDirectoryInput.value = settings.directory
    backupSuccess.value = 'Dossier de sauvegarde mis à jour.'
    await loadBackups()
  } catch (err) {
    backupError.value = err instanceof Error ? err.message : 'Impossible de changer le dossier de sauvegarde.'
  } finally {
    isSavingDirectory.value = false
  }
}

async function handleSaveAutoBackupSettings() {
  if (isSavingAutoBackup.value) return

  isSavingAutoBackup.value = true
  backupError.value = ''
  backupSuccess.value = ''

  try {
    const settings = await updateBackupSettings({
      directory: backupDirectory.value,
      autoBackupEnabled: autoBackupEnabled.value,
      retentionDays: retentionDaysInput.value,
    })
    autoBackupEnabled.value = settings.auto_backup_enabled
    retentionDaysInput.value = settings.retention_days
    backupSuccess.value = 'Réglages de sauvegarde automatique mis à jour.'
  } catch (err) {
    backupError.value = err instanceof Error ? err.message : 'Impossible de mettre à jour les réglages de sauvegarde automatique.'
  } finally {
    isSavingAutoBackup.value = false
  }
}

function isAutoBackup(backup) {
  return backup.name.startsWith('finance-backup-auto-')
}

async function handlePickDirectory() {
  if (isPickingDirectory.value) return

  isPickingDirectory.value = true
  backupError.value = ''
  backupSuccess.value = ''

  try {
    const picked = await pickBackupDirectory()
    if (picked) {
      backupDirectoryInput.value = picked
      await handleSaveDirectory()
    }
  } catch (err) {
    backupError.value = err instanceof Error ? err.message : 'Impossible d’ouvrir le sélecteur de dossier.'
  } finally {
    isPickingDirectory.value = false
  }
}

function formatBackupDate(value) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''

  return new Intl.DateTimeFormat('fr-FR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function formatFileSize(bytes) {
  const value = Number(bytes || 0)
  if (value < 1024) return `${value} o`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} Ko`
  return `${(value / (1024 * 1024)).toFixed(1)} Mo`
}

async function loadBackups() {
  isLoadingBackups.value = true
  backupError.value = ''

  try {
    backups.value = await fetchBackups()
  } catch (err) {
    backupError.value = err instanceof Error ? err.message : 'Impossible de charger les sauvegardes.'
  } finally {
    isLoadingBackups.value = false
  }
}

async function handleCreateBackup() {
  if (isCreatingBackup.value) return

  isCreatingBackup.value = true
  backupError.value = ''
  backupSuccess.value = ''

  try {
    await createBackup()
    backupSuccess.value = 'Sauvegarde créée avec succès.'
    await loadBackups()
  } catch (err) {
    backupError.value = err instanceof Error ? err.message : 'Impossible de créer la sauvegarde.'
  } finally {
    isCreatingBackup.value = false
  }
}

async function handleDownload(backup) {
  backupError.value = ''

  try {
    await downloadBackup(backup.name)
  } catch (err) {
    backupError.value = err instanceof Error ? err.message : 'Impossible de télécharger la sauvegarde.'
  }
}

function requestRestore(backup) {
  restoreTarget.value = backup
  isRestoreModalOpen.value = true
}

function triggerFileInput() {
  fileInput.value?.click()
}

function handleFileChange(event) {
  const file = event.target.files?.[0]
  if (!file) return

  uploadedFile.value = file
  restoreTarget.value = { name: file.name, isUpload: true }
  isRestoreModalOpen.value = true
}

function closeRestoreModal() {
  if (isRestoring.value) return

  isRestoreModalOpen.value = false
  restoreTarget.value = null
  uploadedFile.value = null
  if (fileInput.value) fileInput.value.value = ''
}

async function confirmRestore(backup) {
  isRestoring.value = true
  backupError.value = ''
  backupSuccess.value = ''

  try {
    if (backup.isUpload && uploadedFile.value) {
      await restoreUploadedBackup(uploadedFile.value)
    } else {
      await restoreBackup(backup.name)
    }

    backupSuccess.value = 'Restauration effectuée avec succès.'
    isRestoreModalOpen.value = false
    restoreTarget.value = null
    uploadedFile.value = null
    if (fileInput.value) fileInput.value.value = ''
    await loadBackups()
  } catch (err) {
    backupError.value = err instanceof Error ? err.message : 'Impossible de restaurer cette sauvegarde.'
  } finally {
    isRestoring.value = false
  }
}

async function handleDelete(backup) {
  backupError.value = ''
  backupSuccess.value = ''

  try {
    await deleteBackup(backup.name)
    await loadBackups()
  } catch (err) {
    backupError.value = err instanceof Error ? err.message : 'Impossible de supprimer la sauvegarde.'
  }
}

const dbSettings = ref(null)
const isLoadingDbSettings = ref(false)
const isSavingDbDriver = ref(false)
const dbSettingsError = ref('')
const dbSettingsSuccess = ref('')

async function loadDatabaseSettings() {
  isLoadingDbSettings.value = true
  dbSettingsError.value = ''

  try {
    dbSettings.value = await fetchDatabaseSettings()
  } catch (err) {
    dbSettingsError.value =
      err instanceof Error ? err.message : 'Impossible de charger le réglage de base de données.'
  } finally {
    isLoadingDbSettings.value = false
  }
}

async function selectDatabaseDriver(driver) {
  if (isSavingDbDriver.value || dbSettings.value?.configured_driver === driver) return

  isSavingDbDriver.value = true
  dbSettingsError.value = ''
  dbSettingsSuccess.value = ''

  try {
    dbSettings.value = await updateDatabaseSettings(driver)
    dbSettingsSuccess.value = 'Réglage enregistré.'
  } catch (err) {
    dbSettingsError.value =
      err instanceof Error ? err.message : 'Impossible de changer le réglage de base de données.'
  } finally {
    isSavingDbDriver.value = false
  }
}

function driverLabel(driver) {
  return driver === 'sqlite' ? 'SQLite (locale)' : 'MySQL'
}

onMounted(() => {
  loadBackupSettings()
  loadBackups()
  loadDatabaseSettings()
})
</script>

<template>
  <main class="dashboard-content preferences-view">
    <PageHero
      eyebrow="Personnalisation"
      title="Préférences"
      description="Ajuste l’apparence de l’application selon tes goûts."
    />

    <section class="panel preferences-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Apparence</p>
          <h2>Thème de l’application</h2>
        </div>
      </div>

      <div class="theme-options">
        <button
          v-for="option in THEME_OPTIONS"
          :key="option.value"
          type="button"
          class="theme-option"
          :class="{ 'is-active': theme === option.value }"
          @click="selectTheme(option.value)"
        >
          <span class="theme-option-emoji" aria-hidden="true">{{ option.emoji }}</span>
          <span class="theme-option-label">{{ option.label }}</span>
          <span class="theme-option-description">{{ option.description }}</span>
        </button>
      </div>
    </section>

    <section class="panel preferences-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Personnalisation</p>
          <h2>Format d’affichage</h2>
        </div>
      </div>

      <div class="preferences-field-grid">
        <label class="form-field">
          <span>Devise</span>
          <select :value="currencyCode" @change="preferencesStore.setCurrencyCode($event.target.value)">
            <option v-for="option in CURRENCY_OPTIONS" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </label>

        <label class="form-field">
          <span>Placement de la devise</span>
          <select :value="currencyPosition" @change="preferencesStore.setCurrencyPosition($event.target.value)">
            <option v-for="option in CURRENCY_POSITION_OPTIONS" :key="option.value" :value="option.value">
              {{ option.label }} ({{ option.example }})
            </option>
          </select>
        </label>

        <label class="form-field">
          <span>Format des nombres</span>
          <select
            :value="numberFormatStyle"
            @change="preferencesStore.setNumberFormatStyle($event.target.value)"
          >
            <option v-for="option in NUMBER_FORMAT_OPTIONS" :key="option.value" :value="option.value">
              {{ option.label }} ({{ option.example }})
            </option>
          </select>
        </label>

        <label class="form-field">
          <span>Format de date</span>
          <select :value="dateFormatStyle" @change="preferencesStore.setDateFormatStyle($event.target.value)">
            <option v-for="option in DATE_FORMAT_OPTIONS" :key="option.value" :value="option.value">
              {{ option.label }} ({{ option.example }})
            </option>
          </select>
        </label>
      </div>
    </section>

    <section class="panel preferences-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Stockage</p>
          <h2>Base de données</h2>
        </div>
      </div>

      <p v-if="dbSettingsError" class="form-error">{{ dbSettingsError }}</p>
      <p v-if="dbSettingsSuccess" class="form-success">{{ dbSettingsSuccess }}</p>

      <p v-if="dbSettings?.restart_required" class="db-restart-warning">
        Réglage enregistré pour « {{ driverLabel(dbSettings.configured_driver) }} », mais ce serveur tourne
        encore avec « {{ driverLabel(dbSettings.active_driver) }} » — redémarre le backend pour appliquer
        le changement.
      </p>

      <div class="theme-options">
        <button
          type="button"
          class="theme-option"
          :class="{ 'is-active': dbSettings?.configured_driver === 'mysql' }"
          :disabled="isSavingDbDriver || isLoadingDbSettings"
          @click="selectDatabaseDriver('mysql')"
        >
          <span class="theme-option-emoji" aria-hidden="true">🗄️</span>
          <span class="theme-option-label">MySQL</span>
          <span class="theme-option-description">Base partagée, comme aujourd’hui.</span>
        </button>

        <button
          type="button"
          class="theme-option"
          :class="{ 'is-active': dbSettings?.configured_driver === 'sqlite' }"
          :disabled="isSavingDbDriver || isLoadingDbSettings"
          @click="selectDatabaseDriver('sqlite')"
        >
          <span class="theme-option-emoji" aria-hidden="true">💾</span>
          <span class="theme-option-label">SQLite (locale)</span>
          <span class="theme-option-description">
            Fichier local, sans serveur MySQL — indépendante des données actuelles.
          </span>
        </button>
      </div>
    </section>

    <section class="panel preferences-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Données</p>
          <h2>Sauvegarde et restauration</h2>
        </div>
        <button class="primary-btn" type="button" :disabled="isCreatingBackup" @click="handleCreateBackup">
          {{ isCreatingBackup ? 'Création...' : '+ Créer une sauvegarde' }}
        </button>
      </div>

      <p v-if="backupError" class="form-error">{{ backupError }}</p>
      <p v-if="backupSuccess" class="form-success">{{ backupSuccess }}</p>

      <div class="backup-directory-field">
        <label>
          <span>Dossier de stockage des sauvegardes</span>
          <div class="backup-directory-input-row">
            <input
              v-model="backupDirectoryInput"
              type="text"
              placeholder="/Users/toi/Sauvegardes/Finance"
              :disabled="isSavingDirectory || isPickingDirectory"
            />
            <button
              class="ghost-btn"
              type="button"
              :disabled="isPickingDirectory || isSavingDirectory"
              @click="handlePickDirectory"
            >
              {{ isPickingDirectory ? 'Ouverture du Finder...' : '📁 Parcourir...' }}
            </button>
            <button
              class="ghost-btn"
              type="button"
              :disabled="isSavingDirectory || isPickingDirectory || backupDirectoryInput.trim() === backupDirectory"
              @click="handleSaveDirectory"
            >
              {{ isSavingDirectory ? 'Enregistrement...' : 'Enregistrer' }}
            </button>
          </div>
        </label>
        <p class="backup-directory-hint">
          Chemin absolu, sur la machine qui fait tourner le serveur. « Parcourir » ouvre
          le sélecteur Finder directement sur cette machine. Changer ce dossier ne
          déplace pas les sauvegardes déjà existantes.
        </p>
      </div>

      <div class="backup-auto-field">
        <label class="backup-auto-toggle">
          <input type="checkbox" v-model="autoBackupEnabled" />
          <span>Sauvegarde automatique quotidienne</span>
        </label>

        <div v-if="autoBackupEnabled" class="backup-retention-row">
          <label>
            <span>Conserver</span>
            <input
              v-model.number="retentionDaysInput"
              type="number"
              min="1"
              step="1"
              class="backup-retention-input"
            />
            <span>jours</span>
          </label>
        </div>

        <p class="backup-directory-hint">
          Une sauvegarde est créée automatiquement chaque jour. Les sauvegardes automatiques
          plus anciennes que la rétention choisie sont supprimées ; les sauvegardes créées
          manuellement ne sont jamais supprimées automatiquement.
        </p>

        <button
          class="ghost-btn"
          type="button"
          :disabled="isSavingAutoBackup"
          @click="handleSaveAutoBackupSettings"
        >
          {{ isSavingAutoBackup ? 'Enregistrement...' : 'Enregistrer' }}
        </button>
      </div>

      <p v-if="isLoadingBackups" class="backups-empty">Chargement des sauvegardes...</p>

      <p v-else-if="!backups.length" class="backups-empty">
        Aucune sauvegarde pour l’instant — clique sur « Créer une sauvegarde » pour en générer une.
      </p>

      <div v-else class="backups-list">
        <article v-for="backup in backups" :key="backup.name" class="backup-row">
          <div class="backup-row-main">
            <strong>
              {{ formatBackupDate(backup.created_at) }}
              <span v-if="isAutoBackup(backup)" class="backup-auto-badge">Automatique</span>
            </strong>
            <span class="backup-row-meta">{{ backup.name }} · {{ formatFileSize(backup.size_bytes) }}</span>
          </div>

          <div class="backup-row-actions">
            <button class="ghost-btn" type="button" @click="handleDownload(backup)">Télécharger</button>
            <button class="ghost-btn" type="button" @click="requestRestore(backup)">Restaurer</button>
            <button class="ghost-btn ghost-btn-danger" type="button" @click="handleDelete(backup)">
              Supprimer
            </button>
          </div>
        </article>
      </div>

      <div class="backup-upload">
        <p class="backup-upload-label">Restaurer depuis un fichier de sauvegarde (.sql)</p>
        <div class="backup-upload-control">
          <button class="ghost-btn" type="button" @click="triggerFileInput">Choisir un fichier</button>
          <span class="backup-upload-filename">{{ uploadedFile?.name || 'Aucun fichier choisi' }}</span>
          <input
            ref="fileInput"
            class="backup-upload-input"
            type="file"
            accept=".sql"
            @change="handleFileChange"
          />
        </div>
      </div>
    </section>
  </main>

  <RestoreBackupModal
    :model-value="isRestoreModalOpen"
    :backup="restoreTarget"
    :is-restoring="isRestoring"
    @update:model-value="closeRestoreModal"
    @confirm="confirmRestore"
  />
</template>

<style scoped>
.preferences-view {
  display: grid;
  gap: 0.9rem;
}

.preferences-card {
  padding: 1.1rem;
}

.theme-options {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
  margin-top: 0.9rem;
}

.preferences-field-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.9rem;
  margin-top: 0.9rem;
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

.form-field select {
  width: 100%;
  border: 1px solid rgba(var(--tint-rgb), 0.07);
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.035);
  color: var(--text, #eef1f3);
  padding: 0.9rem 1rem;
  outline: none;
  font-size: 0.92rem;
  font-family: inherit;
  cursor: pointer;
  transition: border-color 140ms ease, background 140ms ease;
}

.form-field select:focus {
  border-color: rgba(219, 230, 223, 0.35);
  background: rgba(var(--tint-rgb), 0.05);
}

@media (max-width: 640px) {
  .preferences-field-grid {
    grid-template-columns: 1fr;
  }
}

.theme-option {
  display: grid;
  justify-items: center;
  gap: 0.4rem;
  padding: 1.1rem 0.9rem;
  border-radius: 16px;
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
  background: rgba(var(--tint-rgb), 0.03);
  color: var(--text-soft, #b3bbc4);
  cursor: pointer;
  text-align: center;
  transition: background 140ms ease, border-color 140ms ease, transform 140ms ease, color 140ms ease;
}

.theme-option:hover {
  transform: translateY(-1px);
  background: rgba(var(--tint-rgb), 0.06);
}

.theme-option.is-active {
  border-color: rgba(114, 137, 152, 0.5);
  background: rgba(114, 137, 152, 0.14);
  color: var(--text, #eef1f3);
}

.theme-option-emoji {
  font-size: 1.6rem;
  line-height: 1;
}

.theme-option-label {
  font-weight: 700;
  font-size: 0.95rem;
  color: var(--text, #eef1f3);
}

.theme-option-description {
  font-size: 0.78rem;
  color: var(--text-dim, #8a939d);
}

@media (max-width: 640px) {
  .theme-options {
    grid-template-columns: 1fr;
  }
}

.form-success {
  margin: 0;
  padding: 0.85rem 1rem;
  border-radius: 14px;
  background: rgba(143, 168, 160, 0.14);
  color: var(--positive-text, #bfe0c9);
  border: 1px solid rgba(143, 168, 160, 0.24);
  font-size: 0.92rem;
}

.db-restart-warning {
  margin: 0 0 0.9rem;
  padding: 0.85rem 1rem;
  border-radius: 14px;
  background: rgba(242, 168, 120, 0.14);
  color: var(--accent-sand, #f2a878);
  border: 1px solid rgba(242, 168, 120, 0.28);
  font-size: 0.88rem;
  line-height: 1.5;
}

.ghost-btn-danger {
  color: var(--negative-text);
}

.backup-directory-field {
  margin-top: 0.9rem;
  padding: 0.9rem 1rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.03);
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
}

.backup-directory-field label {
  display: grid;
  gap: 0.45rem;
  font-size: 0.85rem;
  color: var(--text-soft, #b3bbc4);
}

.backup-directory-input-row {
  display: flex;
  gap: 0.6rem;
}

.backup-directory-input-row input {
  flex: 1;
  min-width: 0;
  border-radius: 12px;
  border: 1px solid rgba(var(--tint-rgb), 0.1);
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--text, #eef1f3);
  padding: 0.6rem 0.8rem;
  font-size: 0.9rem;
  font-family: inherit;
}

.backup-directory-input-row .ghost-btn {
  flex-shrink: 0;
}

.backup-directory-hint {
  margin: 0.5rem 0 0;
  color: var(--text-dim, #8a939d);
  font-size: 0.76rem;
  line-height: 1.4;
}

@media (max-width: 640px) {
  .backup-directory-input-row {
    flex-direction: column;
  }
}

.backup-auto-field {
  margin-top: 0.7rem;
  padding: 0.9rem 1rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.03);
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
  display: grid;
  gap: 0.6rem;
}

.backup-auto-toggle {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.92rem;
  font-weight: 600;
  cursor: pointer;
}

.backup-retention-row label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.85rem;
}

.backup-retention-input {
  width: 70px;
  border-radius: 10px;
  border: 1px solid rgba(var(--tint-rgb), 0.1);
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--text, #eef1f3);
  padding: 0.4rem 0.6rem;
  font-size: 0.9rem;
  font-family: inherit;
}

.backup-auto-badge {
  display: inline-block;
  margin-left: 0.5rem;
  padding: 0.1rem 0.5rem;
  border-radius: 999px;
  background: rgba(94, 203, 143, 0.16);
  color: #5ecb8f;
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.02em;
}

.backups-empty {
  margin: 0.9rem 0 0;
  color: var(--text-dim, #8a939d);
  font-size: 0.88rem;
}

.backups-list {
  display: grid;
  gap: 0.6rem;
  margin-top: 0.9rem;
}

.backup-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.85rem 1rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.03);
  border: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
  flex-wrap: wrap;
}

.backup-row-main {
  display: grid;
  gap: 0.2rem;
  min-width: 0;
}

.backup-row-main strong {
  color: var(--text, #eef1f3);
  font-size: 0.92rem;
}

.backup-row-meta {
  color: var(--text-dim, #8a939d);
  font-size: 0.78rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.backup-row-actions {
  display: flex;
  gap: 0.5rem;
  flex-shrink: 0;
}

.backup-row-actions .ghost-btn {
  padding: 0.5rem 0.85rem;
  font-size: 0.82rem;
}

.backup-upload {
  margin-top: 1.2rem;
  padding-top: 1rem;
  border-top: 1px solid var(--line-soft, rgba(255, 255, 255, 0.05));
}

.backup-upload-label {
  margin: 0 0 0.5rem;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.88rem;
}

.backup-upload-control {
  display: flex;
  align-items: center;
  gap: 0.7rem;
}

.backup-upload-filename {
  color: var(--text-dim, #7c88a6);
  font-size: 0.86rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Le vrai <input type="file"> reste dans le DOM (accessible, ciblable par
   .click()) mais invisible — c'est le bouton .ghost-btn juste à côté qui
   déclenche le sélecteur natif, pour ne jamais montrer le contrôle
   navigateur par défaut (qui ne peut pas être re-stylé de façon fiable
   entre navigateurs). */
.backup-upload-input {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

@media (max-width: 640px) {
  .backup-row {
    flex-direction: column;
    align-items: stretch;
  }

  .backup-row-actions {
    flex-wrap: wrap;
  }
}
</style>
