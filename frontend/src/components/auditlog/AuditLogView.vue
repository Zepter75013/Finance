<script setup>
import { onMounted, ref } from 'vue'
import PageHero from '../Common/PageHero.vue'
import { fetchAuditLog } from '../../services/auditlog'
import { formatDate } from '../../utils/format'

const entries = ref([])
const isLoading = ref(false)
const loadError = ref('')

async function loadEntries() {
  isLoading.value = true
  loadError.value = ''

  try {
    entries.value = await fetchAuditLog()
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : 'Erreur inconnue.'
  } finally {
    isLoading.value = false
  }
}

onMounted(loadEntries)

const ACTION_LABELS = {
  POST: 'Création',
  PUT: 'Modification',
  DELETE: 'Suppression',
}

function actionLabel(method) {
  return ACTION_LABELS[method] || method
}

function formatTime(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })
}

function entityLabel(entry) {
  const type = entry.entity_type || '—'
  return entry.entity_id ? `${type} #${entry.entity_id}` : type
}

// Seules les requêtes POST/PUT/DELETE sont journalisées — les codes
// possibles restent donc un petit ensemble connu, couvert explicitement
// plutôt que d'afficher le numéro brut.
const STATUS_LABELS = {
  200: 'Succès',
  201: 'Créé',
  204: 'Succès',
  400: 'Requête invalide',
  401: 'Non authentifié',
  403: 'Refusé',
  404: 'Introuvable',
  405: 'Méthode non autorisée',
  409: 'Conflit',
  500: 'Erreur serveur',
}

function statusLabel(code) {
  return STATUS_LABELS[code] || (code >= 400 ? `Échec (${code})` : `Succès (${code})`)
}
</script>

<template>
  <main class="dashboard-content audit-log-view">
    <PageHero
      eyebrow="Sécurité"
      title="Journal d’audit"
      description="Qui a créé, modifié ou supprimé quoi, et quand."
    />

    <section v-if="isLoading" class="panel audit-log-state">Chargement...</section>

    <section v-else-if="loadError" class="panel audit-log-state audit-log-state--error">
      {{ loadError }}
    </section>

    <section v-else class="panel audit-log-card">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Historique</p>
          <h2>{{ entries.length }} action{{ entries.length > 1 ? 's' : '' }} récente{{ entries.length > 1 ? 's' : '' }}</h2>
        </div>
      </div>

      <p v-if="!entries.length" class="audit-log-empty">
        Aucune action enregistrée pour l’instant.
      </p>

      <div v-else class="audit-log-table-wrapper">
        <table class="audit-log-table">
          <thead>
            <tr>
              <th>Date</th>
              <th>Utilisateur</th>
              <th>Action</th>
              <th>Élément</th>
              <th>Résultat</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="entry in entries" :key="entry.id">
              <td>{{ formatDate(entry.created_at) }} · {{ formatTime(entry.created_at) }}</td>
              <td>{{ entry.username }}</td>
              <td>{{ actionLabel(entry.method) }}</td>
              <td>{{ entityLabel(entry) }}</td>
              <td :class="entry.status_code >= 400 ? 'audit-log-status-error' : 'audit-log-status-ok'">
                {{ statusLabel(entry.status_code) }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </main>
</template>

<style scoped>
.audit-log-view {
  display: grid;
  gap: 0.9rem;
}

.audit-log-state {
  padding: 1.1rem;
  color: var(--text-dim, #8a939d);
}

.audit-log-state--error {
  color: var(--negative-text);
}

.audit-log-card {
  padding: 1.1rem;
}

.audit-log-empty {
  margin: 0.9rem 0 0;
  color: var(--text-dim, #8a939d);
  font-size: 0.88rem;
}

.audit-log-table-wrapper {
  margin-top: 0.9rem;
  overflow-x: auto;
}

.audit-log-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.88rem;
}

.audit-log-table th {
  text-align: left;
  padding: 0.6rem 0.75rem;
  color: var(--text-dim, #8a939d);
  font-weight: 600;
  font-size: 0.76rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  border-bottom: 1px solid rgba(var(--tint-rgb), 0.08);
  white-space: nowrap;
}

.audit-log-table td {
  padding: 0.6rem 0.75rem;
  border-bottom: 1px solid rgba(var(--tint-rgb), 0.045);
  color: var(--text, #eef1f3);
  white-space: nowrap;
}

.audit-log-table tbody tr:last-child td {
  border-bottom: none;
}

.audit-log-status-ok {
  color: var(--positive-text, #86efac);
}

.audit-log-status-error {
  color: var(--negative-text, #fca5a5);
  font-weight: 700;
}
</style>
