<script setup lang="ts">
/**
 * Single source for every icon in the app. Renders stroke-1.5 SVG paths from
 * a compact map so we never import Heroicons/Lucide at runtime. Add new icons
 * to ICONS below; keep every path 24x24, stroked (no fills).
 */
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    name: IconName;
    size?: number | string;
    stroke?: number;
  }>(),
  {
    size: 16,
    stroke: 1.6,
  },
);

const ICONS = {
  "alert-circle": "M12 8v4m0 4h.01M3 12a9 9 0 1 0 18 0 9 9 0 0 0-18 0Z",
  "alert-triangle":
    "M12 9v4m0 4h.01M4.7 19h14.6a2 2 0 0 0 1.73-3L13.7 4a2 2 0 0 0-3.5 0L3 16a2 2 0 0 0 1.7 3Z",
  info: "M12 8h.01M11 12h1v4m-3 0h6M3 12a9 9 0 1 0 18 0 9 9 0 0 0-18 0Z",
  check: "M4 12l5 5 11-11",
  "check-circle": "m9 12 2 2 4-4m-5 10a9 9 0 1 1 0-18 9 9 0 0 1 0 18Z",
  "x-circle": "M9 9l6 6m0-6-6 6m5 7a9 9 0 1 1 0-18 9 9 0 0 1 0 18Z",
  "chevron-right": "m9 6 6 6-6 6",
  "chevron-down": "m6 9 6 6 6-6",
  "chevron-up": "m18 15-6-6-6 6",
  refresh:
    "M3 12a9 9 0 0 1 15.5-6.3L21 8m0-5v5h-5M21 12a9 9 0 0 1-15.5 6.3L3 16m0 5v-5h5",
  "external-link":
    "M10 6H6a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-4m0-10h-6m6 0v6m0-6L10 14",
  link: "M10 13a5 5 0 0 0 7.5.5l3-3a5 5 0 0 0-7-7l-1.7 1.7m-3.8 6.8a5 5 0 0 0-7.5-.5l-3 3a5 5 0 0 0 7 7l1.7-1.7",
  image:
    "M4 16l4-4 4 4m0 0 5-5 3 3M4 20h16a1 1 0 0 0 1-1V5a1 1 0 0 0-1-1H4a1 1 0 0 0-1 1v14a1 1 0 0 0 1 1Zm10-11h.01",
  shield: "M12 22a9 9 0 0 0 8-9V6l-8-3-8 3v7a9 9 0 0 0 8 9Z",
  eye: "M12 5c-7 0-10 7-10 7s3 7 10 7 10-7 10-7-3-7-10-7Zm0 10a3 3 0 1 1 0-6 3 3 0 0 1 0 6Z",
  accessibility:
    "M12 4a2 2 0 1 1 0 4 2 2 0 0 1 0-4Zm0 6v9m-4-5 8-2m-8 2 8 2m-8-2-2 7m10-7 2 7",
  code: "m10 15-5-3 5-3m4 0 5 3-5 3",
  database:
    "M4 7c0 1.7 3.6 3 8 3s8-1.3 8-3-3.6-3-8-3-8 1.3-8 3Zm0 0v10c0 1.7 3.6 3 8 3s8-1.3 8-3V7M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3",
  zap: "m13 2-8 12h7l-1 8 8-12h-7l1-8Z",
  "trending-up": "m3 17 6-6 4 4 8-8m0 0h-6m6 0v6",
  search: "m21 21-5-5m2-5a7 7 0 1 1-14 0 7 7 0 0 1 14 0Z",
  copy: "M8 5V3a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-2M8 5H6a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2v-4",
  layers:
    "M12 3 2 8l10 5 10-5-10-5Zm0 13 10-5m-20 0 10 5m-10 5 10-5m-10 0 10 5m0-5 10-5m-10 10 10-5",
  globe:
    "M2 12h20M12 2a15 15 0 0 1 0 20M12 2a15 15 0 0 0 0 20M2 12a10 10 0 1 1 20 0 10 10 0 0 1-20 0Z",
  "at-sign":
    "M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0Zm0 0v1.5a2.5 2.5 0 0 0 5 0V12a9 9 0 1 0-4.5 7.8",
  list: "M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01",
} as const;

export type IconName = keyof typeof ICONS;

const path = computed(() => ICONS[props.name]);
</script>

<template>
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 24 24"
    :width="size"
    :height="size"
    fill="none"
    :stroke-width="stroke"
    stroke="currentColor"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path :d="path" />
  </svg>
</template>
