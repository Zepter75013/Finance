<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  backup: {
    type: Object,
    default: null,
  },
  isRestoring: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['update:modelValue', 'confirm'])

const CONFIRM_WORD = 'RESTAURER'
const confirmText = ref('')

watch(
  () => props.modelValue,
  (isOpen) => {
    if (!isOpen) confirmText.value = ''
  }
)

function closeModal() {
  if (props.isRestoring) return
  emit('update:modelValue', false)
}

function confirmRestore() {
  if (!props.backup || props.isRestoring || confirmText.value.trim() !== CONFIRM_WORD) return
  emit('confirm', props.backup)
}
</script>

<template>
  <div v-if="modelValue && backup" class="modal-overlay" @click.self="closeModal">
    <section
      class="modal-card delete-modal-card"
      role="dialog"
      aria-modal="true"
      aria-labelledby="restore-backup-title"
      aria-describedby="restore-backup-description"
    >
      <div class="modal-header">
        <div class="delete-modal-heading">
          <span class="delete-icon" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none">
              <path
                d="M4 4v6h6M20 20v-6h-6M4.5 15a8 8 0 0 0 13.9 3.2M19.5 9A8 8 0 0 0 5.6 5.8"
                stroke="currentColor"
                stroke-width="1.8"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
          </span>

          <div>
            <p class="eyebrow">Restauration</p>
            <h2 id="restore-backup-title">Restaurer cette sauvegarde ?</h2>
          </div>
        </div>
      </div>

      <p id="restore-backup-description" class="delete-modal-text">
        Toutes les données actuelles (achats, revenus, catégories, pointages…) seront
        <strong>remplacées</strong> par le contenu de la sauvegarde
        <strong>{{ backup.name }}</strong>
        . Cette action est irréversible.
      </p>

      <p class="delete-modal-note">
        Par sécurité, une sauvegarde de l’état actuel sera créée automatiquement juste
        avant la restauration — tu pourras y revenir en cas d’erreur.
      </p>

      <label class="confirm-field">
        <span>
          Tape <strong>{{ CONFIRM_WORD }}</strong> pour confirmer
        </span>
        <input
          v-model="confirmText"
          type="text"
          autocomplete="off"
          :placeholder="CONFIRM_WORD"
          :disabled="isRestoring"
        />
      </label>

      <div class="modal-actions">
        <button class="ghost-btn" type="button" :disabled="isRestoring" @click="closeModal">
          Annuler
        </button>

        <button
          class="danger-btn"
          type="button"
          :disabled="isRestoring || confirmText.trim() !== CONFIRM_WORD"
          @click="confirmRestore"
        >
          {{ isRestoring ? 'Restauration...' : 'Restaurer' }}
        </button>
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
  padding: 1.5rem;
  background: var(--modal-overlay);
  backdrop-filter: blur(6px);
  animation: overlay-fade-in 160ms ease;
}

.modal-card {
  width: min(100%, 460px);
  border-radius: 24px;
  padding: 1.35rem;
  background: var(--modal-bg);
  border: 1px solid rgba(var(--tint-rgb), 0.06);
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.34);
}

.delete-modal-card {
  animation: modal-pop-in 180ms ease;
  transform-origin: center;
}

.modal-header {
  margin-bottom: 0.4rem;
}

.delete-modal-heading {
  display: flex;
  align-items: flex-start;
  gap: 0.9rem;
}

.delete-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.75rem;
  height: 2.75rem;
  border-radius: 999px;
  background: rgba(239, 68, 68, 0.12);
  color: #dc2626;
  flex-shrink: 0;
}

.delete-icon svg {
  width: 1.25rem;
  height: 1.25rem;
}

.eyebrow {
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.74rem;
}

.modal-header h2 {
  margin-top: 0.35rem;
  font-size: 1.45rem;
  line-height: 1.15;
  color: var(--text, #eef1f3);
}

.delete-modal-text {
  margin: 1rem 0 0.75rem;
  line-height: 1.6;
  color: var(--text-soft, #b3bbc4);
}

.delete-modal-text strong {
  color: var(--text, #eef1f3);
}

.delete-modal-note {
  margin: 0 0 1rem;
  padding: 0.85rem 0.95rem;
  border-radius: 14px;
  background: rgba(var(--tint-rgb), 0.04);
  border: 1px solid rgba(var(--tint-rgb), 0.05);
  color: var(--text-dim, #8a939d);
  line-height: 1.5;
  font-size: 0.92rem;
}

.confirm-field {
  display: grid;
  gap: 0.4rem;
  margin-bottom: 1.5rem;
  font-size: 0.88rem;
  color: var(--text-soft, #b3bbc4);
}

.confirm-field strong {
  color: var(--text, #eef1f3);
}

.confirm-field input {
  border-radius: 12px;
  border: 1px solid rgba(var(--tint-rgb), 0.1);
  background: rgba(var(--tint-rgb), 0.04);
  color: var(--text, #eef1f3);
  padding: 0.7rem 0.85rem;
  font-size: 0.95rem;
}

.confirm-field input:focus {
  outline: 2px solid rgba(220, 38, 38, 0.4);
  outline-offset: 1px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}

.ghost-btn,
.danger-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 170px;
  border: none;
  border-radius: 14px;
  padding: 0.9rem 1.2rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 140ms ease, background 140ms ease, opacity 140ms ease;
}

.ghost-btn {
  background: rgba(var(--tint-rgb), 0.06);
  color: var(--text);
}

.danger-btn {
  background: #dc2626;
  color: #ffffff;
}

.ghost-btn:hover:not(:disabled),
.danger-btn:hover:not(:disabled) {
  transform: translateY(-1px);
}

.danger-btn:hover:not(:disabled) {
  background: #b91c1c;
}

.ghost-btn:disabled,
.danger-btn:disabled {
  cursor: not-allowed;
  opacity: 0.7;
  transform: none;
}

@keyframes overlay-fade-in {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes modal-pop-in {
  from {
    opacity: 0;
    transform: translateY(8px) scale(0.985);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@media (max-width: 640px) {
  .modal-card {
    padding: 1.1rem;
    border-radius: 20px;
  }

  .modal-actions {
    flex-direction: column;
    align-items: stretch;
  }

  .ghost-btn,
  .danger-btn {
    width: 100%;
    min-width: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .modal-overlay,
  .delete-modal-card,
  .ghost-btn,
  .danger-btn {
    animation: none;
    transition: none;
  }
}
</style>
