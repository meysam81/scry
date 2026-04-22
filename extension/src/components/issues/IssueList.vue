<script setup lang="ts">
/**
 * Full issues table. Filters by severity and category. Issues grouped by
 * check name so repeated hits on the same rule collapse.
 */
import { computed, ref } from 'vue';
import type { Issue, Severity } from '@/schemas/audit';
import { categoryFor, CATEGORIES, CATEGORY_ORDER, type CategoryKey } from '@/lib/categories';
import SeverityChip from '@/components/shared/SeverityChip.vue';
import ScryIcon from '@/components/shared/ScryIcon.vue';

const props = defineProps<{ issues: Issue[] }>();

const severityFilter = ref<Severity | 'all'>('all');
const categoryFilter = ref<CategoryKey | 'all'>('all');

const filtered = computed(() =>
  props.issues.filter((i) => {
    if (severityFilter.value !== 'all' && i.severity !== severityFilter.value)
      return false;
    if (categoryFilter.value !== 'all' && categoryFor(i.check_name) !== categoryFilter.value)
      return false;
    return true;
  }),
);

const availableCategories = computed(() => {
  const seen = new Set<CategoryKey>();
  for (const i of props.issues) seen.add(categoryFor(i.check_name));
  return CATEGORY_ORDER.filter((c) => seen.has(c));
});

const SEVERITY_OPTIONS: Array<{ value: Severity | 'all'; label: string }> = [
  { value: 'all', label: 'All' },
  { value: 'critical', label: 'Critical' },
  { value: 'warning', label: 'Warning' },
  { value: 'info', label: 'Info' },
];

const expanded = ref<Set<string>>(new Set());

function toggle(key: string) {
  if (expanded.value.has(key)) expanded.value.delete(key);
  else expanded.value.add(key);
  // trigger reactivity (Set identity)
  expanded.value = new Set(expanded.value);
}
</script>

<template>
  <div class="wrap">
    <div class="filters">
      <div class="group">
        <label class="label">Severity</label>
        <div class="seg">
          <button
            v-for="opt in SEVERITY_OPTIONS"
            :key="opt.value"
            class="seg-btn"
            :data-active="severityFilter === opt.value || undefined"
            @click="severityFilter = opt.value"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>
      <div v-if="availableCategories.length > 1" class="group">
        <label class="label">Category</label>
        <select class="select" v-model="categoryFilter">
          <option value="all">All categories</option>
          <option v-for="c in availableCategories" :key="c" :value="c">
            {{ CATEGORIES[c].label }}
          </option>
        </select>
      </div>
    </div>

    <ul v-if="filtered.length" class="list">
      <li
        v-for="(issue, idx) in filtered"
        :key="`${issue.check_name}-${issue.url}-${idx}`"
        class="row"
        :data-open="expanded.has(`${idx}`) || undefined"
      >
        <button class="head" @click="toggle(`${idx}`)">
          <ScryIcon
            :name="expanded.has(`${idx}`) ? 'chevron-down' : 'chevron-right'"
            :size="14"
          />
          <SeverityChip :severity="issue.severity" compact />
          <span class="msg">{{ issue.message }}</span>
          <span class="name">{{ issue.check_name }}</span>
        </button>
        <div v-if="expanded.has(`${idx}`) && issue.detail" class="detail">
          <pre>{{ issue.detail }}</pre>
        </div>
      </li>
    </ul>
    <p v-else class="empty">
      Nothing matches these filters.
    </p>
  </div>
</template>

<style scoped>
.wrap {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.filters {
  display: flex;
  gap: var(--space-4);
  flex-wrap: wrap;
}

.group {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.label {
  font-size: var(--font-size-xs);
  color: var(--color-text-faint);
  text-transform: uppercase;
  letter-spacing: var(--letter-wide);
}

.seg {
  display: flex;
  background: var(--color-bg-surface-2);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-md);
  padding: 2px;
  gap: 2px;
}

.seg-btn {
  padding: 4px var(--space-3);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  border: none;
  background: transparent;
  cursor: pointer;
}

.seg-btn[data-active] {
  background: var(--color-bg-hover);
  color: var(--color-text);
}

.select {
  height: 28px;
  padding: 0 var(--space-3);
  background: var(--color-bg-surface-2);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-md);
  color: var(--color-text);
  font-size: var(--font-size-sm);
  font-family: inherit;
}

.list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.row {
  background: var(--color-bg-surface-2);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.head {
  display: grid;
  grid-template-columns: auto auto 1fr auto;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  width: 100%;
  text-align: left;
  cursor: pointer;
  color: var(--color-text);
  transition: background var(--duration-fast) var(--ease-out);
}

.head:hover {
  background: var(--color-bg-hover);
}

.msg {
  font-size: var(--font-size-sm);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.name {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-faint);
  white-space: nowrap;
}

.detail {
  border-top: 1px solid var(--color-border-subtle);
  padding: var(--space-3) var(--space-3) var(--space-3) var(--space-8);
}

.detail pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.empty {
  padding: var(--space-4);
  background: var(--color-bg-surface-2);
  border-radius: var(--radius-md);
  border: 1px dashed var(--color-border-subtle);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
  margin: 0;
}
</style>
