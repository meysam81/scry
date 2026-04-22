// Zod schemas at the WASM ↔ JS boundary. Every value crossing this line is
// parsed, not cast — invalid data becomes `null`, never a runtime explosion.
// These mirror core/model/model.go exactly.
import { z } from "zod";

export const SeveritySchema = z.enum(["critical", "warning", "info"]);
export type Severity = z.infer<typeof SeveritySchema>;

export const IssueSchema = z.object({
  check_name: z.string(),
  severity: SeveritySchema,
  message: z.string(),
  url: z.string(),
  detail: z.string().optional().default(""),
});
export type Issue = z.infer<typeof IssueSchema>;

export const AuditDataSchema = z.object({
  issues: z.array(IssueSchema),
  url: z.string(),
});
export type AuditData = z.infer<typeof AuditDataSchema>;

export const WasmEnvelopeSchema = z.object({
  ok: z.boolean(),
  data: z.unknown().optional(),
  error: z.unknown().optional(),
});
export type WasmEnvelope = z.infer<typeof WasmEnvelopeSchema>;

export const VersionSchema = z.object({
  engine: z.string(),
  api: z.number(),
});
export type Version = z.infer<typeof VersionSchema>;

export const ChecksListSchema = z.object({
  checks: z.array(z.string()),
});
export type ChecksList = z.infer<typeof ChecksListSchema>;

/**
 * Parse a WASM envelope string and narrow its `data` using the supplied
 * schema. Returns null on ANY failure (invalid JSON, envelope not ok,
 * data shape mismatch). The null-on-failure contract is why callers can
 * trust-but-verify without try/catch.
 */
export function parseEnvelope<T extends z.ZodTypeAny>(
  raw: unknown,
  dataSchema: T,
): z.infer<T> | null {
  if (typeof raw !== "string") return null;

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }

  const env = WasmEnvelopeSchema.safeParse(parsed);
  if (!env.success || !env.data.ok) return null;

  const data = dataSchema.safeParse(env.data.data);
  return data.success ? data.data : null;
}
