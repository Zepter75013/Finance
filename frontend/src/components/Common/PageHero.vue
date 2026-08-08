<script setup>
const props = defineProps({
  eyebrow: {
    type: String,
    default: '',
  },
  title: {
    type: String,
    required: true,
  },
  description: {
    type: String,
    default: '',
  },
})
</script>

<template>
  <section class="page-hero">
    <div class="page-hero-copy">
      <p v-if="props.eyebrow" class="eyebrow">{{ props.eyebrow }}</p>
      <h1>{{ props.title }}</h1>
      <p v-if="props.description" class="page-hero-text">
        {{ props.description }}
      </p>
    </div>

    <div v-if="$slots.actions" class="page-hero-actions">
      <slot name="actions" />
    </div>
  </section>
</template>

<style scoped>
.page-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.page-hero-copy {
  min-width: 0;
}

.page-hero h1 {
  margin: 0.2rem 0 0.35rem;
  font-size: 1.45rem;
  line-height: 1.15;
  color: var(--text, #eef1f3);
  font-weight: 700;
}

.page-hero-text {
  max-width: 640px;
  margin: 0;
  color: var(--text-soft, #b3bbc4);
  font-size: 0.88rem;
  line-height: 1.5;
}

.page-hero-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.eyebrow {
  color: var(--text-dim, #8a939d);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.74rem;
}

@media (max-width: 760px) {
  .page-hero {
    flex-direction: column;
    align-items: flex-start;
  }

  .page-hero-actions {
    width: 100%;
  }

  .page-hero-actions :deep(.primary-btn) {
    width: 100%;
  }
}
</style>
