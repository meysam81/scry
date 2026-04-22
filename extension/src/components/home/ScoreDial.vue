<script setup lang="ts">
/**
 * Big editorial serif score dial. Colour comes from a single token derived
 * from the numeric grade so other surfaces can depend on the same mapping.
 */
import { computed } from "vue";

const props = defineProps<{
  score: number;
  grade: "A" | "B" | "C" | "D" | "F";
  critical: number;
  warning: number;
  info: number;
}>();

const GRADE_TOKEN: Record<"A" | "B" | "C" | "D" | "F", string> = {
  A: "var(--color-success)",
  B: "var(--color-success)",
  C: "var(--color-warning)",
  D: "var(--color-warning)",
  F: "var(--color-critical)",
};

const accent = computed(() => GRADE_TOKEN[props.grade]);
const circumference = 2 * Math.PI * 62; // matches r=62 below
const dash = computed(() => (circumference * props.score) / 100);
</script>

<template>
  <div class="dial" :style="{ '--accent': accent }">
    <svg class="ring" viewBox="0 0 140 140" aria-hidden="true">
      <circle cx="70" cy="70" r="62" class="track" />
      <circle
        cx="70"
        cy="70"
        r="62"
        class="progress"
        :stroke-dasharray="`${dash} ${circumference}`"
      />
    </svg>
    <div class="center">
      <div class="number">{{ score }}</div>
      <div class="grade">Grade {{ grade }}</div>
    </div>
    <div class="legend">
      <div class="lg-row">
        <span class="lg-dot" data-kind="critical" />
        <span class="lg-label">Critical</span>
        <span class="lg-n">{{ critical }}</span>
      </div>
      <div class="lg-row">
        <span class="lg-dot" data-kind="warning" />
        <span class="lg-label">Warning</span>
        <span class="lg-n">{{ warning }}</span>
      </div>
      <div class="lg-row">
        <span class="lg-dot" data-kind="info" />
        <span class="lg-label">Info</span>
        <span class="lg-n">{{ info }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dial {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: var(--space-5);
  align-items: center;
}

.ring {
  position: relative;
  width: 140px;
  height: 140px;
  grid-row: 1 / 3;
  grid-column: 1;
}

.track {
  fill: none;
  stroke: var(--color-border-subtle);
  stroke-width: 8;
}

.progress {
  fill: none;
  stroke: var(--accent);
  stroke-width: 8;
  stroke-linecap: round;
  transform: rotate(-90deg);
  transform-origin: center;
  transition: stroke-dasharray var(--duration-slow) var(--ease-out);
}

.center {
  grid-row: 1;
  grid-column: 1;
  position: relative;
  text-align: center;
  align-self: center;
  justify-self: center;
  margin-left: -140px; /* overlay the ring */
  width: 140px;
}

.number {
  font-family: var(--font-display);
  font-size: var(--font-size-display);
  line-height: 1;
  color: var(--accent);
  letter-spacing: var(--letter-tight);
}

.grade {
  margin-top: var(--space-1);
  font-size: var(--font-size-xs);
  color: var(--color-text-faint);
  text-transform: uppercase;
  letter-spacing: var(--letter-wide);
}

.legend {
  grid-row: 1 / 3;
  grid-column: 2;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.lg-row {
  display: grid;
  grid-template-columns: 10px 1fr auto;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
}

.lg-dot {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
}
.lg-dot[data-kind="critical"] {
  background: var(--color-critical);
}
.lg-dot[data-kind="warning"] {
  background: var(--color-warning);
}
.lg-dot[data-kind="info"] {
  background: var(--color-info);
}

.lg-label {
  color: var(--color-text-muted);
}

.lg-n {
  font-family: var(--font-mono);
  color: var(--color-text);
  font-variant-numeric: tabular-nums;
}
</style>
