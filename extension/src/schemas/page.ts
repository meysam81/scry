// The page snapshot the content script collects and hands to the background.
// Shape intentionally mirrors core/model/model.go so the Go side can consume
// it with a single json.Unmarshal.
import { z } from "zod";

export const HeadersSchema = z.record(z.string(), z.array(z.string()));
export type Headers = z.infer<typeof HeadersSchema>;

export const PageSchema = z.object({
  url: z.string(),
  status_code: z.number().int(),
  content_type: z.string().default(""),
  redirect_chain: z.array(z.string()).optional().default([]),
  headers: HeadersSchema.default({}),
  links: z.array(z.string()).default([]),
  assets: z.array(z.string()).default([]),
  depth: z.number().int().default(0),
  fetched_at: z.string(),
  fetch_duration: z.number().default(0),
  in_sitemap: z.boolean().default(false),
});
export type Page = z.infer<typeof PageSchema>;

export const PageSnapshotSchema = z.object({
  page: PageSchema,
  body: z.string(),
  html_meta: z
    .object({
      title: z.string().optional().default(""),
      description: z.string().optional().default(""),
      lang: z.string().optional().default(""),
      canonical: z.string().optional().default(""),
      og: z.record(z.string(), z.string()).default({}),
      twitter: z.record(z.string(), z.string()).default({}),
      json_ld_count: z.number().default(0),
      h1_count: z.number().default(0),
      h2_count: z.number().default(0),
      img_count: z.number().default(0),
      img_without_alt: z.number().default(0),
      link_count: z.number().default(0),
      external_link_count: z.number().default(0),
      word_count: z.number().default(0),
    })
    .optional(),
  technologies: z.array(z.string()).default([]),
});
export type PageSnapshot = z.infer<typeof PageSnapshotSchema>;
