<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useAuditStore } from "@/stores/audit";
import { useActiveTab } from "@/composables/useActiveTab";

import Panel from "@/components/shared/Panel.vue";
import ButtonGhost from "@/components/shared/ButtonGhost.vue";
import ScryIcon, { type IconName } from "@/components/shared/ScryIcon.vue";

import ScoreDial from "@/components/home/ScoreDial.vue";
import CategoryBar from "@/components/home/CategoryBar.vue";
import CriticalList from "@/components/home/CriticalList.vue";
import PageFacts from "@/components/home/PageFacts.vue";

import TechStack from "@/components/tech/TechStack.vue";
import IssueList from "@/components/issues/IssueList.vue";
import HeadersTable from "@/components/headers/HeadersTable.vue";
import SchemaView from "@/components/schema/SchemaView.vue";

const store = useAuditStore();
const { activeTab } = useActiveTab();

type TabKey = "home" | "issues" | "tech" | "headers" | "schema";

const TABS: Array<{ key: TabKey; label: string; icon: IconName }> = [
  { key: "home", label: "Overview", icon: "zap" },
  { key: "issues", label: "Issues", icon: "list" },
  { key: "tech", label: "Tech", icon: "layers" },
  { key: "headers", label: "Headers", icon: "code" },
  { key: "schema", label: "Schema", icon: "database" },
];

const current = ref<TabKey>("home");

const hostname = computed(() => {
  try {
    return new URL(store.url || activeTab.value?.url || "").hostname;
  } catch {
    return "";
  }
});

const isAuditable = computed(() => {
  const url = activeTab.value?.url ?? "";
  return url.startsWith("http://") || url.startsWith("https://");
});

async function runAudit() {
  if (!activeTab.value?.id || !isAuditable.value) return;
  await store.request(activeTab.value.id);
}

onMounted(() => {
  if (activeTab.value?.id && isAuditable.value) void runAudit();
});

watch(
  () => activeTab.value?.id,
  (id) => {
    if (id && isAuditable.value) {
      store.reset();
      void runAudit();
    }
  },
);
</script>

<template>
  <div class="shell">
    <header class="topbar">
      <div class="brand">
        <div class="mark">
          <ScryIcon name="eye" :size="18" :stroke="1.8" />
        </div>
        <div class="brand-text">
          <span class="brand-name">Scry</span>
          <span v-if="hostname" class="brand-host">{{ hostname }}</span>
        </div>
      </div>
      <ButtonGhost
        variant="icon"
        :disabled="!isAuditable || store.status === 'loading'"
        @click="runAudit"
        title="Re-run audit"
      >
        <ScryIcon name="refresh" :size="16" />
      </ButtonGhost>
    </header>

    <nav class="tabs" role="tablist">
      <button
        v-for="tab in TABS"
        :key="tab.key"
        class="tab"
        role="tab"
        :data-active="current === tab.key || undefined"
        :aria-selected="current === tab.key"
        @click="current = tab.key"
      >
        <ScryIcon :name="tab.icon" :size="14" />
        <span>{{ tab.label }}</span>
      </button>
    </nav>

    <main class="content">
      <div v-if="!isAuditable" class="state">
        <Panel variant="muted">
          <p class="state-text">
            Scry can only audit pages on <code>http://</code> or
            <code>https://</code>. Navigate to a real website to run an audit.
          </p>
        </Panel>
      </div>

      <div v-else-if="store.status === 'loading'" class="state">
        <Panel variant="muted">
          <div class="loading">
            <span class="spinner" />
            <span>Auditing {{ hostname || "page" }}…</span>
          </div>
        </Panel>
      </div>

      <div v-else-if="store.status === 'error'" class="state">
        <Panel title="Audit failed" variant="default">
          <p class="err-msg">{{ store.error }}</p>
          <p class="hint">
            Try refreshing the page (so chrome.webRequest sees the response
            headers) and then re-running the audit.
          </p>
          <ButtonGhost variant="primary" size="sm" @click="runAudit">
            <ScryIcon name="refresh" :size="14" />
            Retry
          </ButtonGhost>
        </Panel>
      </div>

      <template v-else-if="store.status === 'ready'">
        <section v-show="current === 'home'" class="view">
          <Panel variant="flush">
            <ScoreDial
              :score="store.score.overall"
              :grade="store.score.grade"
              :critical="store.score.counts.critical"
              :warning="store.score.counts.warning"
              :info="store.score.counts.info"
            />
          </Panel>

          <Panel
            title="Critical & warnings"
            subtitle="Highest-impact issues on this page"
          >
            <CriticalList :issues="store.issues" :max="4" />
          </Panel>

          <Panel title="Page facts" subtitle="At-a-glance numerics">
            <PageFacts :snapshot="store.snapshot" />
          </Panel>

          <Panel title="Categories" subtitle="Per-family health score">
            <div class="cat-list">
              <CategoryBar
                v-for="entry in store.score.byCategory"
                :key="entry.key"
                :entry="entry"
                @click="current = 'issues'"
              />
            </div>
          </Panel>

          <Panel
            v-if="(store.snapshot?.technologies.length ?? 0) > 0"
            title="Detected stack"
          >
            <TechStack :technologies="store.snapshot?.technologies ?? []" />
          </Panel>

          <footer class="footer">
            Audited in {{ store.durationMs }}ms ·
            {{ new Date(store.ranAt).toLocaleTimeString() }}
          </footer>
        </section>

        <section v-show="current === 'issues'" class="view">
          <IssueList :issues="store.issues" />
        </section>

        <section v-show="current === 'tech'" class="view">
          <Panel
            title="Technologies"
            subtitle="Detected from DOM, headers & scripts"
          >
            <TechStack :technologies="store.snapshot?.technologies ?? []" />
          </Panel>
        </section>

        <section v-show="current === 'headers'" class="view">
          <Panel
            title="Response headers"
            subtitle="Captured from chrome.webRequest"
          >
            <HeadersTable :headers="store.snapshot?.page.headers ?? {}" />
          </Panel>
        </section>

        <section v-show="current === 'schema'" class="view">
          <Panel title="Structured data" subtitle="OpenGraph, Twitter, JSON-LD">
            <SchemaView :snapshot="store.snapshot" />
          </Panel>
        </section>
      </template>

      <div v-else class="state">
        <Panel variant="muted">
          <p class="state-text">Preparing audit…</p>
        </Panel>
      </div>
    </main>
  </div>
</template>

<style scoped>
.shell {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: var(--color-bg-base);
}

.topbar {
  position: sticky;
  top: 0;
  z-index: var(--z-sticky);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-5);
  background: var(--color-bg-base);
  border-bottom: 1px solid var(--color-border-subtle);
  height: var(--layout-header-height);
}

.brand {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-width: 0;
}

.mark {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  background: var(--color-accent-soft);
  color: var(--color-accent);
  border-radius: var(--radius-md);
  border: 1px solid color-mix(in oklab, var(--color-accent) 25%, transparent);
}

.brand-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
  line-height: 1.1;
}

.brand-name {
  font-family: var(--font-display);
  font-size: var(--font-size-lg);
  letter-spacing: var(--letter-tight);
}

.brand-host {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-faint);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tabs {
  position: sticky;
  top: var(--layout-header-height);
  z-index: var(--z-sticky);
  display: flex;
  gap: 2px;
  padding: var(--space-2) var(--space-4);
  background: var(--color-bg-base);
  border-bottom: 1px solid var(--color-border-subtle);
  height: var(--layout-tab-height);
}

.tab {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: 0 var(--space-3);
  height: 28px;
  border-radius: var(--radius-md);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  background: transparent;
  border: 1px solid transparent;
  cursor: pointer;
  white-space: nowrap;
}

.tab:hover {
  background: var(--color-bg-hover);
  color: var(--color-text);
}

.tab[data-active] {
  background: var(--color-bg-surface-2);
  color: var(--color-text);
  border-color: var(--color-border-subtle);
}

.content {
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  flex: 1;
}

.view {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.cat-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.state {
  padding-top: var(--space-6);
}

.state-text {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
  line-height: var(--line-height-snug);
}

.loading {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  color: var(--color-text-muted);
  font-size: var(--font-size-sm);
}

.spinner {
  width: 14px;
  height: 14px;
  border-radius: var(--radius-full);
  border: 1.5px solid var(--color-border);
  border-top-color: var(--color-accent);
  animation: spin var(--duration-slow) linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.err-msg {
  margin: 0 0 var(--space-2);
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-critical);
}

.hint {
  margin: 0 0 var(--space-3);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
  line-height: var(--line-height-snug);
}

.footer {
  font-size: var(--font-size-xs);
  color: var(--color-text-faint);
  padding: var(--space-3) 0 var(--space-4);
  text-align: center;
  font-variant-numeric: tabular-nums;
}

code {
  background: var(--color-bg-surface-3);
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-size: 0.9em;
  color: var(--color-text);
}
</style>
