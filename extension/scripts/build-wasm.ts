// Compiles cmd/wasm into extension/public/scry.wasm and copies Go's
// official wasm_exec.js shim alongside it. Run from the extension/ dir:
//   bun run build:wasm
import { $ } from "bun";
import { mkdir, cp } from "node:fs/promises";
import { existsSync } from "node:fs";
import { resolve } from "node:path";

const here = import.meta.dir;
const repoRoot = resolve(here, "..", "..");
const publicDir = resolve(here, "..", "public");

await mkdir(publicDir, { recursive: true });

const goroot = (await $`go env GOROOT`.text()).trim();
if (!goroot) throw new Error("GOROOT is empty — is Go installed?");

// Go 1.21+ ships the shim at $GOROOT/lib/wasm/wasm_exec.js. Older versions
// had it under misc/wasm. Support both so we do not break on older Go.
const shimCandidates = [
  resolve(goroot, "lib", "wasm", "wasm_exec.js"),
  resolve(goroot, "misc", "wasm", "wasm_exec.js"),
];
const shim = shimCandidates.find((p) => existsSync(p));
if (!shim) throw new Error(`wasm_exec.js not found under ${goroot}`);

console.log(`[scry] compiling WASM from ${repoRoot}/cmd/wasm …`);
await $`GOOS=js GOARCH=wasm go build -ldflags="-s -w" -trimpath -o ${publicDir}/scry.wasm ./cmd/wasm/`.cwd(
  repoRoot,
);

console.log(`[scry] copying wasm shim from ${shim}`);
await cp(shim, resolve(publicDir, "wasm_exec.js"));

console.log(`[scry] ok → ${publicDir}/scry.wasm`);
