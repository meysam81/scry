// A thin wrapper around the Go WASM module. Hides the one-shot `go.run()`
// bootstrap and exposes a typed, Zod-validated surface. Designed to be
// instantiated once per service worker (or UI context) and reused.
//
// The shim file `wasm_exec.js` defines `globalThis.Go`. We dynamic-import it
// so that environments without a DOM (service workers) get a clean bootstrap.
import {
  parseEnvelope,
  AuditDataSchema,
  VersionSchema,
  ChecksListSchema,
  type AuditData,
} from '@/schemas/audit';
import type { Page } from '@/schemas/page';

declare global {
  // eslint-disable-next-line no-var
  var Go: undefined | (new () => GoInstance);
  // Functions Go exports once `go.run()` starts.
  // eslint-disable-next-line no-var
  var scryAuditPage: undefined | ((input: string) => string);
  // eslint-disable-next-line no-var
  var scryListChecks: undefined | (() => string);
  // eslint-disable-next-line no-var
  var scryVersion: undefined | (() => string);
}

interface GoInstance {
  importObject: WebAssembly.Imports;
  run(instance: WebAssembly.Instance): Promise<void>;
}

export interface WasmRuntimeOptions {
  /** URL to the compiled Go WASM binary. Must be a chrome-extension:// URL. */
  wasmUrl: string;
  /** URL to the Go wasm_exec.js shim. */
  shimUrl: string;
}

export class WasmRuntime {
  #booted: Promise<void> | null = null;
  #opts: WasmRuntimeOptions;

  constructor(opts: WasmRuntimeOptions) {
    this.#opts = opts;
  }

  /** Idempotent. Returns the same promise on every call until boot resolves. */
  async boot(): Promise<void> {
    this.#booted ??= this.#bootOnce();
    return this.#booted;
  }

  async #bootOnce(): Promise<void> {
    if (typeof globalThis.Go !== 'function') {
      await import(/* @vite-ignore */ this.#opts.shimUrl);
    }
    if (typeof globalThis.Go !== 'function') {
      throw new Error('wasm_exec.js did not register globalThis.Go');
    }

    const go = new globalThis.Go();
    const wasm = await fetch(this.#opts.wasmUrl).then((r) => r.arrayBuffer());
    const { instance } = await WebAssembly.instantiate(wasm, go.importObject);

    // Intentionally do not await — go.run blocks until the Go program exits,
    // and our program blocks forever on `select {}` to keep js.FuncOf wrappers
    // alive. We just let it run in the background.
    void go.run(instance);

    // Poll briefly for the exported globals to appear. This is pragmatic:
    // the Go runtime sets them synchronously in main(), but "synchronously"
    // here means "on the first microtask after go.run() starts".
    for (let i = 0; i < 50; i++) {
      if (typeof globalThis.scryAuditPage === 'function') return;
      await new Promise((r) => setTimeout(r, 10));
    }
    throw new Error('scryAuditPage never materialised after boot');
  }

  async version() {
    await this.boot();
    return parseEnvelope(globalThis.scryVersion!(), VersionSchema);
  }

  async listChecks() {
    await this.boot();
    return parseEnvelope(globalThis.scryListChecks!(), ChecksListSchema);
  }

  async auditPage(page: Page, body: string): Promise<AuditData | null> {
    await this.boot();
    const input = JSON.stringify({ page, body });
    return parseEnvelope(globalThis.scryAuditPage!(input), AuditDataSchema);
  }
}
