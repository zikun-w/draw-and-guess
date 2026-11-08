<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { Undo2, Eraser, Pencil, Trash2, Minus, Plus } from "@lucide/vue";
import { connection, isDrawer, send, state, strokes, notify } from "../game";
import type { Point, Stroke } from "../game";

const canvas = ref<HTMLCanvasElement>();
const colors = [
  "#292d2b",
  "#ef675b",
  "#f6b84b",
  "#79b985",
  "#5c9cde",
  "#9d7ac9",
  "#ee9ab5",
  "#ffffff",
];
const color = ref(colors[0]!);
const width = ref(6);
const tool = ref("pen");
const enabled = computed(
  () =>
    isDrawer.value &&
    state.value?.phase === "drawing" &&
    connection.value === "connected",
);
let current: Stroke | null = null;
let pending: Point[] = [];
let local: Stroke | null = null;
let timer: ReturnType<typeof setInterval>;
let resize: ResizeObserver;
let paintRequest = 0;
let activePointer: number | null = null;

function paintStroke(ctx: CanvasRenderingContext2D, s: Stroke) {
  if (!s.points.length) return;
  ctx.strokeStyle = s.tool === "eraser" ? "#ffffff" : s.color;
  ctx.fillStyle = ctx.strokeStyle;
  ctx.lineWidth = s.width;
  ctx.lineCap = "round";
  ctx.lineJoin = "round";
  ctx.beginPath();
  s.points.forEach((p, i) => {
    const x = (p.x / 10000) * 1000,
      y = (p.y / 10000) * 650;
    if (i === 0) ctx.moveTo(x, y);
    else ctx.lineTo(x, y);
  });
  if (s.points.length === 1) {
    const p = s.points[0]!;
    ctx.arc(
      (p.x / 10000) * 1000,
      (p.y / 10000) * 650,
      s.width / 2,
      0,
      Math.PI * 2,
    );
    ctx.fill();
  } else ctx.stroke();
}
function draw() {
  paintRequest = 0;
  const el = canvas.value,
    ctx = el?.getContext("2d");
  if (!el || !ctx) return;
  const dpr = Math.min(devicePixelRatio || 1, 2),
    w = Math.round(el.clientWidth * dpr),
    h = Math.round(el.clientHeight * dpr);
  if (el.width !== w || el.height !== h) {
    el.width = w;
    el.height = h;
  }
  ctx.setTransform(1, 0, 0, 1, 0, 0);
  ctx.fillStyle = "#fff";
  ctx.fillRect(0, 0, w, h);
  ctx.setTransform(w / 1000, 0, 0, h / 650, 0, 0);
  for (const s of strokes.value) {
    if (!local || s.id !== local.id) paintStroke(ctx, s);
  }
  if (local) {
    const echoed = strokes.value.find((s) => s.id === local!.id);
    if (
      !enabled.value ||
      (!current && echoed && echoed.points.length >= local.points.length)
    )
      local = null;
    else paintStroke(ctx, local);
    if (!local && echoed) paintStroke(ctx, echoed);
  }
}
function requestPaint() {
  if (!paintRequest) paintRequest = requestAnimationFrame(draw);
}
function point(e: PointerEvent): Point {
  const rect = canvas.value!.getBoundingClientRect();
  return {
    x: Math.round(
      Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width)) * 10000,
    ),
    y: Math.round(
      Math.max(0, Math.min(1, (e.clientY - rect.top) / rect.height)) * 10000,
    ),
  };
}
function down(e: PointerEvent) {
  if (!enabled.value || activePointer !== null || e.button !== 0) return;
  e.preventDefault();
  canvas.value!.setPointerCapture(e.pointerId);
  activePointer = e.pointerId;
  // The previous stroke is already in transit in the same ordered connection.
  flush();
  const p = point(e);
  current = {
    id: crypto.randomUUID(),
    color: color.value!,
    width: width.value,
    tool: tool.value,
    points: [p],
  };
  local = current;
  pending = [p];
  requestPaint();
}
function move(e: PointerEvent) {
  if (!current || activePointer !== e.pointerId) return;
  const p = point(e),
    last = current.points.at(-1)!;
  if (Math.abs(p.x - last.x) + Math.abs(p.y - last.y) < 8) return;
  current.points.push(p);
  pending.push(p);
  if (pending.length >= 80) flush();
  requestPaint();
}
function flush() {
  if (!local || !pending.length) return;
  const points = pending.splice(0, 96);
  if (!send("stroke", { stroke: { ...local, points } })) {
    current = null;
    local = null;
    pending = [];
    requestPaint();
  }
}
function up(e?: PointerEvent) {
  if (e && activePointer !== e.pointerId) return;
  flush();
  current = null;
  activePointer = null;
  requestPaint();
}
function action(type: string) {
  up();
  local = null;
  if (type === "clear" && !confirm("清空当前画板？")) return;
  send(type);
  requestPaint();
}
function savePreview() {
  if (!canvas.value) return;
  const a = document.createElement("a");
  a.href = canvas.value.toDataURL("image/png");
  a.download = `画猜-${state.value?.id || "作品"}.png`;
  a.click();
  notify("作品已保存");
}
defineExpose({ savePreview });
watch(strokes, requestPaint, { deep: true });
watch(
  () => [state.value?.turn, state.value?.phase, connection.value],
  () => {
    if (!enabled.value || state.value?.phase === "choosing") {
      current = null;
      local = null;
      pending = [];
      activePointer = null;
    }
    requestPaint();
  },
);
onMounted(() => {
  timer = setInterval(flush, 40);
  resize = new ResizeObserver(requestPaint);
  resize.observe(canvas.value!);
  requestPaint();
});
onBeforeUnmount(() => {
  clearInterval(timer);
  resize.disconnect();
  cancelAnimationFrame(paintRequest);
});
</script>

<template>
  <div class="drawing-surface" :class="{ drawable: enabled }">
    <canvas
      ref="canvas"
      aria-label="绘画画板"
      @pointerdown="down"
      @pointermove="move"
      @pointerup="up"
      @pointercancel="up"
      @lostpointercapture="up"
    />
    <div v-if="state?.phase === 'lobby'" class="board-placeholder">
      <div class="mini-doodle">
        <Pencil :size="42" :stroke-width="1.6" /><span class="spark">✳</span>
      </div>
      <h2>画笔已就位，脑洞准备好了吗？</h2>
      <p>邀请朋友加入，开启一场灵魂画作大赏</p>
      <span class="handwritten">画得开心，猜得尽兴。</span>
    </div>
    <div v-if="state?.phase === 'choosing'" class="board-overlay">
      <template v-if="isDrawer"
        ><span class="eyebrow">轮到你大显身手</span>
        <h2>这次，画点什么？</h2>
        <p>选一个词，准备开画</p>
        <div class="word-options">
          <button
            v-for="(word, i) in state.choices"
            :key="word"
            @click="send('choose', { index: i })"
          >
            {{ word }}<span>选这个 ↗</span>
          </button>
        </div></template
      >
      <template v-else
        ><div class="waiting-pencils">✎</div>
        <h2>画手正在挑选灵感…</h2>
        <p>准备好你的想象力，马上开画</p></template
      >
    </div>
    <div v-if="state?.phase === 'reveal'" class="answer-banner">
      <span>本轮答案</span><strong>{{ state.revealedWord }}</strong
      ><small>下一轮即将开始</small>
    </div>
  </div>
  <div class="drawing-toolbar" :class="{ 'tools-disabled': !enabled }">
    <div class="tool-group">
      <button
        class="icon-button"
        :class="{ selected: tool === 'pen' }"
        title="画笔"
        aria-label="画笔"
        :disabled="!enabled"
        @click="tool = 'pen'"
      >
        <Pencil :size="19" /></button
      ><button
        class="icon-button"
        :class="{ selected: tool === 'eraser' }"
        title="橡皮擦"
        aria-label="橡皮擦"
        :disabled="!enabled"
        @click="tool = 'eraser'"
      >
        <Eraser :size="19" />
      </button>
    </div>
    <div class="palette">
      <button
        v-for="c in colors"
        :key="c"
        :style="{ '--swatch': c }"
        :class="{ active: color === c && tool === 'pen' }"
        :aria-label="`颜色 ${c}`"
        :disabled="!enabled"
        @click="
          color = c;
          tool = 'pen';
        "
      />
    </div>
    <div class="brush-size">
      <button
        aria-label="减小画笔"
        :disabled="!enabled || width <= 2"
        @click="width = Math.max(2, width - 2)"
      >
        <Minus :size="13" /></button
      ><span>{{ width }}</span
      ><button
        aria-label="增大画笔"
        :disabled="!enabled || width >= 40"
        @click="width = Math.min(40, width + 2)"
      >
        <Plus :size="13" />
      </button>
    </div>
    <div class="tool-group tool-end">
      <button
        class="icon-button"
        aria-label="撤销"
        title="撤销上一笔"
        :disabled="!enabled"
        @click="action('undo')"
      >
        <Undo2 :size="19" /></button
      ><button
        class="icon-button"
        aria-label="清空画板"
        title="清空画板"
        :disabled="!enabled"
        @click="action('clear')"
      >
        <Trash2 :size="18" />
      </button>
    </div>
  </div>
</template>
