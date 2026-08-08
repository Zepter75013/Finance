<script setup>
import { ref, watch } from 'vue'
import { APP_VERSION, APP_BUILD_TIME } from '../../version'
import { fetchDatabaseSettings } from '../../services/settings'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['update:modelValue'])

const dbDriverLabel = ref('')

function driverLabel(driver) {
  return driver === 'sqlite' ? 'SQLite (locale)' : 'MySQL'
}

// Chargé à l'ouverture plutôt qu'au montage — la modale est montée une seule
// fois au démarrage de l'app (v-if interne), pas besoin d'appeler l'API tant
// que personne n'a jamais ouvert "À propos".
watch(
  () => props.modelValue,
  (isOpen) => {
    if (!isOpen || dbDriverLabel.value) return

    fetchDatabaseSettings()
      .then((settings) => {
        dbDriverLabel.value = driverLabel(settings.active_driver)
      })
      .catch(() => {
        // Purement informatif — la modale reste utilisable sans cette ligne.
      })
  }
)

function closeModal() {
  emit('update:modelValue', false)
}

function formatBuildTime(value) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''

  return new Intl.DateTimeFormat('fr-FR', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}
</script>

<template>
  <div v-if="modelValue" class="modal-overlay" @click.self="closeModal">
    <section class="modal-card">
      <div class="modal-header">
        <div>
          <p class="eyebrow">Finance UI</p>
          <h2>À propos</h2>
        </div>

        <button class="ghost-btn" type="button" @click="closeModal">Fermer</button>
      </div>

      <div class="about-body">
        <div class="about-brand">
          <div class="about-brand-mark">F</div>
          <div>
            <strong>Finance UI</strong>
            <p>Suivi personnel de finances</p>
          </div>
        </div>

        <dl class="about-list">
          <div class="about-row">
            <dt>Version</dt>
            <dd>v{{ APP_VERSION }}</dd>
          </div>
          <div class="about-row">
            <dt>Compilé le</dt>
            <dd>{{ formatBuildTime(APP_BUILD_TIME) }}</dd>
          </div>
          <div v-if="dbDriverLabel" class="about-row">
            <dt>Base de données</dt>
            <dd>{{ dbDriverLabel }}</dd>
          </div>
        </dl>

        <p class="about-note">
          Application de suivi des achats, revenus, budgets et pointages bancaires.
        </p>
      </div>
    </section>
  </div>
</template>

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
  width: min(100%, 420px);
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
  font-size: 1.4rem;
  line-height: 1.15;
  color: var(--text, #eef1f3);
}

.eyebrow {
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.74rem;
}

.about-body {
  display: grid;
  gap: 1.1rem;
}

.about-brand {
  display: flex;
  align-items: center;
  gap: 0.85rem;
}

.about-brand-mark {
  width: 40px;
  height: 40px;
  border-radius: 13px;
  display: grid;
  place-items: center;
  background: linear-gradient(135deg, var(--accent, #728998), var(--accent-soft, #819b8d));
  color: #ffffff;
  font-weight: 700;
  font-size: 1.05rem;
  flex-shrink: 0;
}

.about-brand strong {
  display: block;
  color: var(--text, #eef1f3);
  font-size: 1rem;
}

.about-brand p {
  margin-top: 0.15rem;
  color: var(--text-dim, #8a939d);
  font-size: 0.82rem;
}

.about-list {
  margin: 0;
  display: grid;
  gap: 0.5rem;
}

.about-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.65rem 0.85rem;
  border-radius: 12px;
  background: rgba(var(--tint-rgb), 0.035);
  border: 1px solid rgba(var(--tint-rgb), 0.06);
}

.about-row dt {
  color: var(--text-soft, #b3bbc4);
  font-size: 0.86rem;
  font-weight: 600;
}

.about-row dd {
  margin: 0;
  color: var(--text, #eef1f3);
  font-size: 0.9rem;
  font-weight: 600;
}

.about-note {
  margin: 0;
  color: var(--text-dim, #8a939d);
  font-size: 0.82rem;
  line-height: 1.5;
}
</style>
