<script setup lang="ts">
/**
 * Bordered surface. Every card, section, and empty state in the app is a
 * Panel. Variants change surface shade and border emphasis only; no bespoke
 * shadows or colors.
 */
withDefaults(
  defineProps<{
    title?: string;
    subtitle?: string;
    variant?: "default" | "muted" | "flush";
    compact?: boolean;
  }>(),
  { variant: "default" },
);
</script>

<template>
  <section
    class="panel"
    :data-variant="variant"
    :data-compact="compact || undefined"
  >
    <header v-if="title || $slots.header" class="head">
      <slot name="header">
        <div>
          <h3 class="title">{{ title }}</h3>
          <p v-if="subtitle" class="subtitle">{{ subtitle }}</p>
        </div>
      </slot>
      <div v-if="$slots.actions" class="actions">
        <slot name="actions" />
      </div>
    </header>
    <div class="body">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.panel {
  border: 1px solid var(--color-border-subtle);
  background: var(--color-bg-surface);
  border-radius: var(--radius-lg);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.panel[data-variant="muted"] {
  background: var(--color-bg-surface-2);
  border-color: var(--color-border-subtle);
}

.panel[data-variant="flush"] {
  background: transparent;
  border: none;
  border-radius: 0;
}

.head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--color-border-subtle);
}

.panel[data-compact] .head {
  padding: var(--space-3) var(--space-4);
}

.title {
  margin: 0;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  letter-spacing: var(--letter-wide);
  text-transform: uppercase;
  color: var(--color-text-muted);
}

.subtitle {
  margin: var(--space-1) 0 0;
  font-size: var(--font-size-xs);
  color: var(--color-text-faint);
  font-weight: var(--font-weight-normal);
}

.actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.body {
  padding: var(--space-5);
}

.panel[data-compact] .body {
  padding: var(--space-4);
}

.panel[data-variant="flush"] .body,
.panel[data-variant="flush"] .head {
  padding: 0;
}
</style>
