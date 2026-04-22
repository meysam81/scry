// The single source of truth for the UI. Each tab gets one AuditState slot;
// we never store more than one because the side panel only ever shows the
// active tab. Cached snapshots are timestamped so "stale" badges can fire.
import { defineStore } from 'pinia';
import { ref, computed, shallowRef } from 'vue';
import type { Issue } from '@/schemas/audit';
import type { PageSnapshot } from '@/schemas/page';
import { summarize } from '@/lib/scoring';
import { parseMessage } from '@/schemas/messages';

export type AuditStatus = 'idle' | 'loading' | 'ready' | 'error';

export const useAuditStore = defineStore('audit', () => {
  const tabId = ref<number | null>(null);
  const url = ref<string>('');
  const status = ref<AuditStatus>('idle');
  const error = ref<string>('');
  const issues = shallowRef<Issue[]>([]);
  const snapshot = shallowRef<PageSnapshot | null>(null);
  const ranAt = ref<string>('');
  const durationMs = ref<number>(0);

  const score = computed(() => summarize(issues.value));

  async function request(currentTabId: number) {
    tabId.value = currentTabId;
    status.value = 'loading';
    error.value = '';

    try {
      const response = await chrome.runtime.sendMessage({
        kind: 'ui:request-audit',
        tabId: currentTabId,
      });
      applyResponse(response);
    } catch (e) {
      status.value = 'error';
      error.value = `no response from background: ${String(e)}`;
    }
  }

  function applyResponse(raw: unknown) {
    const msg = parseMessage(raw);
    if (!msg) {
      status.value = 'error';
      error.value = 'invalid message from background';
      return;
    }
    if (msg.kind === 'bg:audit-error') {
      status.value = 'error';
      error.value = msg.error;
      return;
    }
    if (msg.kind === 'bg:audit-result') {
      url.value = msg.url;
      issues.value = msg.issues;
      snapshot.value = msg.snapshot;
      ranAt.value = msg.ran_at;
      durationMs.value = msg.duration_ms;
      status.value = 'ready';
    }
  }

  function reset() {
    issues.value = [];
    snapshot.value = null;
    status.value = 'idle';
    error.value = '';
    url.value = '';
    ranAt.value = '';
    durationMs.value = 0;
  }

  return {
    tabId,
    url,
    status,
    error,
    issues,
    snapshot,
    ranAt,
    durationMs,
    score,
    request,
    reset,
  };
});
