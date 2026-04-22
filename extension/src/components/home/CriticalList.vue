<script setup lang="ts">
/**
 * The "top N issues you should fix right now" list on Home. Surfaces the
 * most punishing issues with a one-tap copy-check-name affordance.
 */
import { computed } from "vue";
import type { Issue } from "@/schemas/audit";
import SeverityChip from "@/components/shared/SeverityChip.vue";

const props = withDefaults(defineProps<{ issues: Issue[]; max?: number }>(), {
  max: 4,
});

const SEVERITY_RANK = { critical: 0, warning: 1, info: 2 };
const top = computed(() =>
  [...props.issues]
    .sort((a, b) => SEVERITY_RANK[a.severity] - SEVERITY_RANK[b.severity])
    .slice(0, props.max),
);
</script>

<template>
  <ul v-if="top.length" class="list">
    <li
      v-for="issue in top"
      :key="`${issue.check_name}-${issue.url}`"
      class="item"
    >
      <SeverityChip :severity="issue.severity" compact />
      <div class="content">
        <p class="msg">{{ issue.message }}</p>
        <p class="name">{{ issue.check_name }}</p>
      </div>
    </li>
  </ul>
  <p v-else class="empty">
    No issues found. Either this site is genuinely immaculate, or the page
    hasn't reloaded since the extension was installed — try refreshing.
  </p>
</template>

<style scoped>
.list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.item {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: var(--space-3);
  align-items: start;
  padding: var(--space-3);
  border-radius: var(--radius-md);
  background: var(--color-bg-surface-2);
  border: 1px solid var(--color-border-subtle);
}

.content {
  min-width: 0;
}

.msg {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-text);
  line-height: var(--line-height-snug);
}

.name {
  margin: var(--space-1) 0 0;
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-faint);
  overflow-wrap: anywhere;
}

.empty {
  margin: 0;
  padding: var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
  background: var(--color-bg-surface-2);
  border-radius: var(--radius-md);
  border: 1px dashed var(--color-border-subtle);
  line-height: var(--line-height-snug);
}
</style>
