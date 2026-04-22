// Maps a check's canonical name (e.g. "seo/missing-title") to the audit
// category it belongs to. One source of truth for both sidebar grouping and
// category chip colours.
export type CategoryKey =
  | 'seo'
  | 'performance'
  | 'security'
  | 'accessibility'
  | 'images'
  | 'structured-data'
  | 'health'
  | 'links'
  | 'tls'
  | 'hreflang'
  | 'external-links'
  | 'other';

export interface CategoryMeta {
  key: CategoryKey;
  label: string;
  cssVar: string; // e.g. "--color-cat-seo"
  description: string;
}

export const CATEGORIES: Record<CategoryKey, CategoryMeta> = {
  seo:              { key: 'seo',              label: 'SEO',             cssVar: '--color-cat-seo',           description: 'Titles, meta, OG, Twitter, canonical, lang' },
  performance:      { key: 'performance',      label: 'Performance',     cssVar: '--color-cat-performance',   description: 'Cache, compression, HTTP/2, resource hints' },
  security:         { key: 'security',         label: 'Security',        cssVar: '--color-cat-security',      description: 'CSP, HSTS, XFO, Permissions-Policy, security.txt' },
  accessibility:    { key: 'accessibility',    label: 'Accessibility',   cssVar: '--color-cat-accessibility', description: 'ARIA landmarks, skip-nav, lang attribute' },
  images:           { key: 'images',           label: 'Images',          cssVar: '--color-cat-images',        description: 'Alt text, dimensions, loading attributes' },
  'structured-data':{ key: 'structured-data',  label: 'Structured Data', cssVar: '--color-cat-structured',    description: 'JSON-LD, microdata, Schema.org coverage' },
  health:           { key: 'health',           label: 'Health',          cssVar: '--color-cat-health',        description: 'Status codes, charset, content-type' },
  links:            { key: 'links',            label: 'Links',           cssVar: '--color-cat-links',         description: 'Anchor text, broken internal links' },
  tls:              { key: 'tls',              label: 'TLS',             cssVar: '--color-cat-tls',           description: 'Certificate strength, HTTPS posture' },
  hreflang:         { key: 'hreflang',         label: 'Hreflang',        cssVar: '--color-cat-hreflang',      description: 'Internationalisation annotations' },
  'external-links': { key: 'external-links',   label: 'External Links',  cssVar: '--color-cat-links',         description: 'rel=noopener, safe target attributes' },
  other:            { key: 'other',            label: 'Other',           cssVar: '--color-cat-links',         description: 'Miscellaneous findings' },
};

const CHECK_PREFIX_TO_CATEGORY: Array<[string, CategoryKey]> = [
  ['seo/',               'seo'],
  ['performance/',       'performance'],
  ['security/',          'security'],
  ['accessibility/',     'accessibility'],
  ['images/',            'images'],
  ['structured-data/',   'structured-data'],
  ['schema/',            'structured-data'],
  ['health/',            'health'],
  ['links/',             'links'],
  ['tls/',               'tls'],
  ['hreflang/',          'hreflang'],
  ['external-links/',    'external-links'],
];

export function categoryFor(checkName: string): CategoryKey {
  for (const [prefix, cat] of CHECK_PREFIX_TO_CATEGORY) {
    if (checkName.startsWith(prefix)) return cat;
  }
  return 'other';
}

export const CATEGORY_ORDER: CategoryKey[] = [
  'seo',
  'performance',
  'security',
  'accessibility',
  'images',
  'structured-data',
  'health',
  'links',
  'tls',
  'hreflang',
  'external-links',
  'other',
];
