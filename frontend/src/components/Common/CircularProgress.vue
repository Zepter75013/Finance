<script setup>
import { computed } from 'vue'

const props = defineProps({
  percent: {
    type: Number,
    default: 0,
  },
  size: {
    type: Number,
    default: 128,
  },
  strokeWidth: {
    type: Number,
    default: 10,
  },
  color: {
    type: String,
    default: 'var(--accent, #3b82f6)',
  },
  trackColor: {
    type: String,
    default: 'rgba(255, 255, 255, 0.08)',
  },
  label: {
    type: String,
    default: '',
  },
})

const radius = computed(() => (props.size - props.strokeWidth) / 2)
const circumference = computed(() => 2 * Math.PI * radius.value)
// Le pourcentage affiché en chiffres peut dépasser 100 (budget dépassé),
// mais l'anneau lui-même se fige plein à 100% — un anneau qui "repart" au
//-delà d'un tour complet serait illisible.
const clampedPercent = computed(() => Math.max(0, Math.min(100, props.percent)))
const dashOffset = computed(() => circumference.value * (1 - clampedPercent.value / 100))
const center = computed(() => props.size / 2)
const rotateTransform = computed(() => `rotate(-90 ${center.value} ${center.value})`)
</script>

<template>
  <div class="circular-progress" :style="{ width: size + 'px', height: size + 'px' }">
    <svg :width="size" :height="size" :viewBox="`0 0 ${size} ${size}`">
      <circle :cx="center" :cy="center" :r="radius" fill="none" :stroke="trackColor" :stroke-width="strokeWidth" />
      <circle
        :cx="center"
        :cy="center"
        :r="radius"
        fill="none"
        :stroke="color"
        :stroke-width="strokeWidth"
        stroke-linecap="round"
        :stroke-dasharray="circumference"
        :stroke-dashoffset="dashOffset"
        :transform="rotateTransform"
        class="circular-progress-arc"
      />
    </svg>

    <div class="circular-progress-content">
      <strong>{{ Math.round(percent) }}<span class="circular-progress-unit">%</span></strong>
      <span v-if="label" class="circular-progress-label">{{ label }}</span>
    </div>
  </div>
</template>

<style scoped>
.circular-progress {
  position: relative;
  display: inline-grid;
  place-items: center;
}

.circular-progress-arc {
  transition: stroke-dashoffset 500ms ease;
}

.circular-progress-content {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  text-align: center;
  line-height: 1.15;
}

.circular-progress-content strong {
  font-size: 1.6rem;
  font-weight: 700;
  color: var(--text, #f3f5fa);
}

.circular-progress-unit {
  font-size: 1rem;
  font-weight: 600;
  margin-left: 1px;
}

.circular-progress-label {
  display: block;
  margin-top: 4px;
  font-size: 0.66rem;
  color: var(--text-dim, #7c88a6);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
</style>
