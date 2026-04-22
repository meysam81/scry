// Content script: runs in the active tab, captures DOM + extracts metadata,
// ships a PageSnapshot to the service worker. Pure data collection; never
// calls into WASM (that lives in the SW, so one WASM instance serves all tabs).
import type { PageSnapshot } from '@/schemas/page';
import { detectTech, type DetectInput } from '@/lib/tech-detect';

function readMeta(name: string): string {
  return (
    document
      .querySelector<HTMLMetaElement>(`meta[name="${name}" i]`)
      ?.content?.trim() ?? ''
  );
}

function collectOg(): Record<string, string> {
  const out: Record<string, string> = {};
  document
    .querySelectorAll<HTMLMetaElement>('meta[property^="og:" i]')
    .forEach((el) => {
      const key = el.getAttribute('property')?.slice(3);
      if (key) out[key] = el.content.trim();
    });
  return out;
}

function collectTwitter(): Record<string, string> {
  const out: Record<string, string> = {};
  document
    .querySelectorAll<HTMLMetaElement>('meta[name^="twitter:" i]')
    .forEach((el) => {
      const key = el.getAttribute('name')?.slice(8);
      if (key) out[key] = el.content.trim();
    });
  return out;
}

function collectLinks(): string[] {
  const seen = new Set<string>();
  document.querySelectorAll<HTMLAnchorElement>('a[href]').forEach((a) => {
    try {
      const abs = new URL(a.href, window.location.href).toString();
      seen.add(abs);
    } catch {
      /* invalid href, skip */
    }
  });
  return Array.from(seen).slice(0, 500);
}

function collectAssets(): string[] {
  const seen = new Set<string>();
  document
    .querySelectorAll<HTMLImageElement | HTMLScriptElement | HTMLLinkElement>(
      'img[src], script[src], link[rel="stylesheet"][href]',
    )
    .forEach((el) => {
      const src =
        (el as HTMLImageElement).src ||
        (el as HTMLScriptElement).src ||
        (el as HTMLLinkElement).href;
      if (src) seen.add(src);
    });
  return Array.from(seen).slice(0, 500);
}

function countWords(text: string): number {
  return text.split(/\s+/).filter(Boolean).length;
}

function collectMetaTags(): Array<{
  name?: string;
  property?: string;
  content?: string;
}> {
  return Array.from(document.querySelectorAll('meta')).map((m) => ({
    name: m.getAttribute('name') ?? undefined,
    property: m.getAttribute('property') ?? undefined,
    content: m.getAttribute('content') ?? undefined,
  }));
}

function collectScripts(): string[] {
  return Array.from(document.querySelectorAll<HTMLScriptElement>('script[src]'))
    .map((s) => s.src)
    .filter(Boolean);
}

function buildSnapshot(): PageSnapshot {
  const html = document.documentElement.outerHTML;
  const bodyText = document.body?.innerText ?? '';

  const imgs = document.querySelectorAll<HTMLImageElement>('img');
  const imgWithoutAlt = Array.from(imgs).filter(
    (i) => !i.alt?.trim(),
  ).length;

  const origin = window.location.origin;
  const links = Array.from(document.querySelectorAll<HTMLAnchorElement>('a[href]'))
    .map((a) => a.href)
    .filter(Boolean);
  const externalLinks = links.filter((l) => {
    try {
      return new URL(l).origin !== origin;
    } catch {
      return false;
    }
  });

  const detectInput: DetectInput = {
    html,
    headers: {}, // headers are added by the SW before forwarding to WASM
    scripts: collectScripts(),
    metaTags: collectMetaTags(),
  };
  const technologies = detectTech(detectInput).map(
    (t) => `${t.category}:${t.name}`,
  );

  return {
    page: {
      url: window.location.href,
      status_code: 200, // refined by the SW from webRequest
      content_type: document.contentType || 'text/html',
      redirect_chain: [],
      headers: {},
      links: collectLinks(),
      assets: collectAssets(),
      depth: 0,
      fetched_at: new Date().toISOString(),
      fetch_duration: 0,
      in_sitemap: false,
    },
    body: html,
    html_meta: {
      title: document.title.trim(),
      description: readMeta('description'),
      lang: document.documentElement.lang || '',
      canonical:
        document
          .querySelector<HTMLLinkElement>('link[rel="canonical" i]')
          ?.href ?? '',
      og: collectOg(),
      twitter: collectTwitter(),
      json_ld_count: document.querySelectorAll(
        'script[type="application/ld+json"]',
      ).length,
      h1_count: document.querySelectorAll('h1').length,
      h2_count: document.querySelectorAll('h2').length,
      img_count: imgs.length,
      img_without_alt: imgWithoutAlt,
      link_count: links.length,
      external_link_count: externalLinks.length,
      word_count: countWords(bodyText),
    },
    technologies,
  };
}

// Respond on demand when the SW asks for a snapshot. Using request-response
// instead of fire-and-forget means the SW only collects when a UI is open.
chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  if (msg?.kind === 'bg:request-snapshot') {
    try {
      const snapshot = buildSnapshot();
      sendResponse({ kind: 'content:snapshot', snapshot });
    } catch (e) {
      sendResponse({ kind: 'content:error', error: String(e) });
    }
    return true; // keep channel open for async response
  }
  // Drop others silently. Do not `return true` for messages we don't handle.
  return false;
});
