// Aggregates raw issues into the at-a-glance scores the Home tab renders.
// One implementation, consumed by every UI surface.
import type { Issue } from '@/schemas/audit';
import { categoryFor, CATEGORIES, CATEGORY_ORDER, type CategoryKey } from './categories';

const SEVERITY_WEIGHT = { critical: 14, warning: 5, info: 1 } as const;

export interface CategoryScore {
  key: CategoryKey;
  label: string;
  cssVar: string;
  score: number;         // 0-100, higher = better
  critical: number;
  warning: number;
  info: number;
  total: number;
}

export interface ScoreSummary {
  overall: number;        // 0-100
  grade: 'A' | 'B' | 'C' | 'D' | 'F';
  counts: { critical: number; warning: number; info: number; total: number };
  byCategory: CategoryScore[];
}

function toScore(critical: number, warning: number, info: number): number {
  const penalty =
    critical * SEVERITY_WEIGHT.critical +
    warning * SEVERITY_WEIGHT.warning +
    info * SEVERITY_WEIGHT.info;
  return Math.max(0, Math.min(100, 100 - penalty));
}

function grade(score: number): ScoreSummary['grade'] {
  if (score >= 90) return 'A';
  if (score >= 75) return 'B';
  if (score >= 60) return 'C';
  if (score >= 40) return 'D';
  return 'F';
}

export function summarize(issues: Issue[]): ScoreSummary {
  const counts = { critical: 0, warning: 0, info: 0, total: issues.length };
  const perCat = new Map<CategoryKey, CategoryScore>();

  for (const cat of CATEGORY_ORDER) {
    const meta = CATEGORIES[cat];
    perCat.set(cat, {
      key: cat,
      label: meta.label,
      cssVar: meta.cssVar,
      score: 100,
      critical: 0,
      warning: 0,
      info: 0,
      total: 0,
    });
  }

  for (const issue of issues) {
    counts[issue.severity]++;
    const cat = categoryFor(issue.check_name);
    const row = perCat.get(cat)!;
    row[issue.severity]++;
    row.total++;
  }

  for (const row of perCat.values()) {
    row.score = toScore(row.critical, row.warning, row.info);
  }

  const overall = toScore(counts.critical, counts.warning, counts.info);

  return {
    overall,
    grade: grade(overall),
    counts,
    byCategory: Array.from(perCat.values())
      .filter((c) => c.total > 0)
      .sort((a, b) => a.score - b.score),
  };
}
