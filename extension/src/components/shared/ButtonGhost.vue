<script setup lang="ts">
/**
 * The only button in the app. Variants decouple intent (primary/ghost/icon)
 * from base rules (focus ring, radius, spacing).
 */
withDefaults(
  defineProps<{
    variant?: 'primary' | 'ghost' | 'icon';
    size?: 'sm' | 'md';
    disabled?: boolean;
  }>(),
  { variant: 'ghost', size: 'md' },
);
</script>

<template>
  <button
    class="btn"
    :data-variant="variant"
    :data-size="size"
    :disabled="disabled"
  >
    <slot />
  </button>
</template>

<style scoped>
.btn {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  font-family: inherit;
  font-weight: var(--font-weight-medium);
  border-radius: var(--radius-md);
  transition:
    background var(--duration-fast) var(--ease-out),
    color var(--duration-fast) var(--ease-out),
    border-color var(--duration-fast) var(--ease-out);
  border: 1px solid transparent;
  cursor: pointer;
  user-select: none;
}

.btn[data-size='sm'] {
  height: 28px;
  padding: 0 var(--space-3);
  font-size: var(--font-size-xs);
}
.btn[data-size='md'] {
  height: 32px;
  padding: 0 var(--space-4);
  font-size: var(--font-size-sm);
}

.btn[data-variant='primary'] {
  background: var(--color-accent);
  color: var(--color-text-on-accent);
}
.btn[data-variant='primary']:hover:not(:disabled) {
  background: var(--color-accent-hover);
}
.btn[data-variant='primary']:active:not(:disabled) {
  background: var(--color-accent-active);
}

.btn[data-variant='ghost'] {
  background: transparent;
  color: var(--color-text-muted);
  border-color: var(--color-border);
}
.btn[data-variant='ghost']:hover:not(:disabled) {
  color: var(--color-text);
  border-color: var(--color-border-strong);
  background: var(--color-bg-hover);
}

.btn[data-variant='icon'] {
  width: 32px;
  height: 32px;
  padding: 0;
  justify-content: center;
  background: transparent;
  color: var(--color-text-muted);
}
.btn[data-variant='icon']:hover:not(:disabled) {
  color: var(--color-text);
  background: var(--color-bg-hover);
}

.btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
</style>
