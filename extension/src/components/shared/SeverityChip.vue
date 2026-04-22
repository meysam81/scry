<script setup lang="ts">
/**
 * One chip to render every severity tag across the app. Binds to
 * --color-{critical,warning,info} tokens, no inline colors ever.
 */
import type { Severity } from "@/schemas/audit";

defineProps<{
  severity: Severity;
  compact?: boolean;
}>();

const LABEL: Record<Severity, string> = {
  critical: "Critical",
  warning: "Warning",
  info: "Info",
};
</script>

<template>
  <span
    class="chip"
    :data-severity="severity"
    :data-compact="compact || undefined"
  >
    <span class="dot" />
    {{ compact ? LABEL[severity][0] : LABEL[severity] }}
  </span>
</template>

<style scoped>
.chip {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: 2px var(--space-2);
  border-radius: var(--radius-full);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  letter-spacing: var(--letter-wide);
  text-transform: uppercase;
  border: 1px solid transparent;
  background: var(--_bg);
  color: var(--_fg);
  border-color: var(--_border);
  white-space: nowrap;
}

.chip[data-severity="critical"] {
  --_bg: var(--color-critical-soft);
  --_fg: var(--color-critical);
  --_border: color-mix(in oklab, var(--color-critical) 30%, transparent);
}

.chip[data-severity="warning"] {
  --_bg: var(--color-warning-soft);
  --_fg: var(--color-warning);
  --_border: color-mix(in oklab, var(--color-warning) 30%, transparent);
}

.chip[data-severity="info"] {
  --_bg: var(--color-info-soft);
  --_fg: var(--color-info);
  --_border: color-mix(in oklab, var(--color-info) 30%, transparent);
}

.chip[data-compact] {
  justify-content: center;
  width: 18px;
  height: 18px;
  padding: 0;
  font-size: 10px;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: var(--radius-full);
  background: currentColor;
}

.chip[data-compact] .dot {
  display: none;
}
</style>
