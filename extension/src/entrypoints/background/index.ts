// The service worker is the only place that owns a WasmRuntime instance.
// It:
//   1. Listens to chrome.webRequest to cache real response headers per tab.
//   2. Answers UI panels with audit results via chrome.runtime.sendMessage.
//   3. Requests a DOM snapshot from the active tab's content script.
//
// Rule of thumb: never store derived state here — only raw captures. The
// UI derives scores/categories/etc. off the raw issue list.
import { WasmRuntime } from '@/lib/wasm-runtime';
import { PageSnapshotSchema } from '@/schemas/page';
import type { MsgAuditResult, MsgAuditError } from '@/schemas/messages';

type Headers = Record<string, string[]>;

interface HeaderCapture {
  url: string;
  statusCode: number;
  headers: Headers;
  capturedAt: number;
}

// Per-tab last-known real response headers. Populated by webRequest,
// consumed at audit time. Cleared on tab removal.
const headerCache = new Map<number, HeaderCapture>();

const wasm = new WasmRuntime({
  wasmUrl: chrome.runtime.getURL('scry.wasm'),
  shimUrl: chrome.runtime.getURL('wasm_exec.js'),
});

// -----------------------------------------------------------------------------
// Side panel plumbing — tapping the toolbar icon opens the panel.
// -----------------------------------------------------------------------------
chrome.sidePanel
  .setPanelBehavior({ openPanelOnActionClick: true })
  .catch((err) => console.warn('[scry] setPanelBehavior failed', err));

// -----------------------------------------------------------------------------
// Header capture — chrome.webRequest.onResponseStarted fires before the page
// has finished loading, giving us the real response headers as the browser
// received them (including HSTS, CSP, cache-control, etc.).
// -----------------------------------------------------------------------------
chrome.webRequest.onResponseStarted.addListener(
  (details) => {
    if (details.type !== 'main_frame') return;
    if (details.tabId < 0) return;

    const headers: Headers = {};
    for (const h of details.responseHeaders ?? []) {
      const key = h.name.toLowerCase();
      (headers[key] ??= []).push(h.value ?? '');
    }

    headerCache.set(details.tabId, {
      url: details.url,
      statusCode: details.statusCode,
      headers,
      capturedAt: Date.now(),
    });
  },
  { urls: ['<all_urls>'] },
  ['responseHeaders', 'extraHeaders'],
);

chrome.tabs.onRemoved.addListener((tabId) => headerCache.delete(tabId));

// -----------------------------------------------------------------------------
// Snapshot request helper. Injects the content script on demand if needed
// (handles edge cases where the content script hasn't loaded yet — e.g. when
// the user opens the side panel before the page's `document_idle` event).
// -----------------------------------------------------------------------------
async function requestSnapshot(tabId: number): Promise<unknown> {
  try {
    return await chrome.tabs.sendMessage(tabId, { kind: 'bg:request-snapshot' });
  } catch {
    // Content script not loaded: inject it programmatically.
    await chrome.scripting.executeScript({
      target: { tabId },
      files: ['src/entrypoints/content/index.ts'],
    });
    return chrome.tabs.sendMessage(tabId, { kind: 'bg:request-snapshot' });
  }
}

// -----------------------------------------------------------------------------
// Audit pipeline
// -----------------------------------------------------------------------------
async function runAudit(tabId: number): Promise<MsgAuditResult | MsgAuditError> {
  const t0 = performance.now();

  let snapshotRaw: unknown;
  try {
    snapshotRaw = await requestSnapshot(tabId);
  } catch (e) {
    return { kind: 'bg:audit-error', tabId, error: `snapshot failed: ${String(e)}` };
  }

  // Trust-but-verify.
  const envelope =
    snapshotRaw && typeof snapshotRaw === 'object' && 'snapshot' in snapshotRaw
      ? (snapshotRaw as { snapshot: unknown }).snapshot
      : null;

  const parsed = PageSnapshotSchema.safeParse(envelope);
  if (!parsed.success) {
    return {
      kind: 'bg:audit-error',
      tabId,
      error: `invalid snapshot shape: ${parsed.error.issues[0]?.message ?? 'unknown'}`,
    };
  }

  const snapshot = parsed.data;

  // Overlay real headers / status onto the snapshot.
  const capture = headerCache.get(tabId);
  if (capture && capture.url === snapshot.page.url) {
    snapshot.page.headers = capture.headers;
    snapshot.page.status_code = capture.statusCode;
  }

  const auditData = await wasm.auditPage(snapshot.page, snapshot.body);
  if (!auditData) {
    return {
      kind: 'bg:audit-error',
      tabId,
      error: 'WASM audit returned invalid data',
    };
  }

  return {
    kind: 'bg:audit-result',
    tabId,
    url: auditData.url || snapshot.page.url,
    issues: auditData.issues,
    snapshot,
    ran_at: new Date().toISOString(),
    duration_ms: Math.round(performance.now() - t0),
  };
}

// -----------------------------------------------------------------------------
// Message router
// -----------------------------------------------------------------------------
chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  if (msg?.kind === 'ui:request-audit' || msg?.kind === 'ui:refresh') {
    const tabId = Number(msg.tabId);
    if (!Number.isFinite(tabId)) {
      sendResponse({
        kind: 'bg:audit-error',
        tabId: 0,
        error: 'missing tabId',
      } satisfies MsgAuditError);
      return false;
    }
    runAudit(tabId)
      .then(sendResponse)
      .catch((err) =>
        sendResponse({
          kind: 'bg:audit-error',
          tabId,
          error: String(err),
        } satisfies MsgAuditError),
      );
    return true; // async
  }
  return false;
});

// Warm the WASM runtime on install so the first audit feels snappy.
chrome.runtime.onInstalled.addListener(() => {
  wasm.boot().catch((err) => console.warn('[scry] WASM boot failed', err));
});
