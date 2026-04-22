// Typed contracts for every chrome.runtime.sendMessage exchange between
// service worker, content script, and UI surfaces.
import { z } from "zod";
import { PageSnapshotSchema } from "./page";
import { IssueSchema } from "./audit";

export const MsgContentSnapshotSchema = z.object({
  kind: z.literal("content:snapshot"),
  snapshot: PageSnapshotSchema,
});

export const MsgRequestAuditSchema = z.object({
  kind: z.literal("ui:request-audit"),
  tabId: z.number(),
});

export const MsgAuditResultSchema = z.object({
  kind: z.literal("bg:audit-result"),
  tabId: z.number(),
  url: z.string(),
  issues: z.array(IssueSchema),
  snapshot: PageSnapshotSchema.nullable(),
  ran_at: z.string(),
  duration_ms: z.number(),
});

export const MsgAuditErrorSchema = z.object({
  kind: z.literal("bg:audit-error"),
  tabId: z.number(),
  error: z.string(),
});

export const MsgRequestRefreshSchema = z.object({
  kind: z.literal("ui:refresh"),
  tabId: z.number(),
});

export const AnyMessageSchema = z.discriminatedUnion("kind", [
  MsgContentSnapshotSchema,
  MsgRequestAuditSchema,
  MsgAuditResultSchema,
  MsgAuditErrorSchema,
  MsgRequestRefreshSchema,
]);
export type AnyMessage = z.infer<typeof AnyMessageSchema>;

export type MsgContentSnapshot = z.infer<typeof MsgContentSnapshotSchema>;
export type MsgRequestAudit = z.infer<typeof MsgRequestAuditSchema>;
export type MsgAuditResult = z.infer<typeof MsgAuditResultSchema>;
export type MsgAuditError = z.infer<typeof MsgAuditErrorSchema>;
export type MsgRequestRefresh = z.infer<typeof MsgRequestRefreshSchema>;

export function parseMessage(raw: unknown): AnyMessage | null {
  const r = AnyMessageSchema.safeParse(raw);
  return r.success ? r.data : null;
}
