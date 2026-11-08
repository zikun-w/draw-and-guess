import { computed, ref } from "vue";
import { PROTOBUF_PROTOCOL, encodeStroke, decodeDrawing } from "./wire";
import type { IncomingFrame } from "./wire";

export interface Point {
  x: number;
  y: number;
}
export interface Stroke {
  id: string;
  color: string;
  width: number;
  tool: string;
  points: Point[];
}
export interface Player {
  id: string;
  name: string;
  score: number;
  connected: boolean;
  guessed: boolean;
}
export interface Settings {
  rounds: number;
  drawSeconds: number;
}
export interface State {
  version: number;
  id: string;
  phase: "lobby" | "choosing" | "drawing" | "reveal" | "finished";
  hostId: string;
  drawerId: string;
  players: Player[];
  settings: Settings;
  turn: number;
  totalTurns: number;
  deadline: number;
  serverTime: number;
  wordLength: number;
  word?: string;
  choices?: string[];
  revealedWord?: string;
  seq: number;
  messages: { id: number; name: string; text: string; kind: string }[];
}
export interface Result {
  id: string;
  room: string;
  finishedAt: number;
  players: Player[];
}
export const session = ref<{ id: string; name: string } | null>(null);
export const state = ref<State | null>(null);
export const strokes = ref<Stroke[]>([]);
export const connection = ref<
  "offline" | "connecting" | "connected" | "reconnecting"
>("offline");
export const toast = ref("");
export const now = ref(Date.now());
export const clockOffset = ref(0);
let toastTimer: ReturnType<typeof setTimeout>;
export function notify(message: string) {
  toast.value = message;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => (toast.value = ""), 4000);
}
export async function api<T>(url: string, body?: unknown): Promise<T> {
  const response = await fetch(url, {
    method: body === undefined ? "GET" : "POST",
    headers: { "Content-Type": "application/json" },
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  });
  const data = await response.json();
  if (!response.ok)
    throw Object.assign(new Error(data.error || "连接失败，请稍后重试"), {
      status: response.status,
    });
  return data;
}
export const isDrawer = computed(
  () => !!state.value && state.value.drawerId === session.value?.id,
);
export const isHost = computed(
  () => !!state.value && state.value.hostId === session.value?.id,
);
export const remaining = computed(() =>
  Math.max(
    0,
    Math.ceil(
      ((state.value?.deadline || 0) - now.value - clockOffset.value) / 1000,
    ),
  ),
);
export const ranked = computed(() =>
  [...(state.value?.players || [])].sort((a, b) => b.score - a.score),
);
let socket: WebSocket | undefined;
let retry: ReturnType<typeof setTimeout> | undefined;
let wantedRoom = "";
let attempts = 0;
let receivedSeq = 0;
let awaitingSync = false;
let jsonFallback = false;
export function send(type: string, payload: Record<string, unknown> = {}) {
  if (socket?.readyState !== WebSocket.OPEN) return false;
  if (type === "stroke" && socket.protocol === PROTOBUF_PROTOCOL) {
    socket.send(encodeStroke(state.value?.turn || 0, payload.stroke as Stroke));
    return true;
  }
  socket.send(
    JSON.stringify({ type, turn: state.value?.turn || 0, ...payload }),
  );
  return true;
}
export function connectRoom(room: string) {
  disconnectRoom(false);
  wantedRoom = room;
  attempts = 0;
  jsonFallback = false;
  state.value = null;
  strokes.value = [];
  const url = new URL(location.href);
  url.searchParams.set("room", room);
  history.replaceState({}, "", url);
  openSocket();
}
async function openSocket() {
  if (!wantedRoom) return;
  connection.value = attempts ? "reconnecting" : "connecting";
  const room = wantedRoom;
  try {
    await api(`/api/rooms/${encodeURIComponent(room)}`);
  } catch (error) {
    if (room !== wantedRoom) return;
    const e = error as Error & { status?: number };
    if (e.status === 404 || e.status === 401) {
      notify(e.message);
      if (e.status === 401) session.value = null;
      disconnectRoom();
      return;
    }
  }
  if (room !== wantedRoom) return;
  const useProtobuf =
    import.meta.env.VITE_WIRE_FORMAT !== "json" && !jsonFallback;
  let opened = false;
  const current = new WebSocket(
    `${location.protocol === "https:" ? "wss:" : "ws:"}//${location.host}/ws?room=${encodeURIComponent(wantedRoom)}`,
    useProtobuf ? [PROTOBUF_PROTOCOL] : [],
  );
  current.binaryType = "arraybuffer";
  socket = current;
  current.onopen = () => {
    if (current !== socket) return;
    opened = true;
    connection.value = "connected";
    attempts = 0;
  };
  current.onmessage = (event) => {
    if (current !== socket) return;
    let frame: IncomingFrame;
    try {
      frame =
        typeof event.data === "string"
          ? JSON.parse(event.data)
          : decodeDrawing(event.data);
    } catch {
      notify("房间消息无法解析，正在重新连接");
      current.close(1002, "invalid drawing packet");
      return;
    }
    if (frame.type === "fatal") {
      notify(frame.message || "房间连接已结束");
      disconnectRoom();
      return;
    }
    if (frame.type === "error") {
      notify(frame.message || "操作失败，请稍后重试");
      return;
    }
    if (frame.type === "snapshot" || frame.type === "state") {
      const next = frame.state as State;
      const old = state.value;
      if (
        frame.type === "snapshot" ||
        !old ||
        next.turn !== old.turn ||
        next.phase === "lobby"
      ) {
        strokes.value = frame.strokes || [];
        receivedSeq = next.seq;
        awaitingSync = false;
      }
      clockOffset.value = next.serverTime - Date.now();
      state.value = next;
      return;
    }
    if (!state.value || frame.turn !== state.value.turn || awaitingSync) return;
    if (frame.seq !== receivedSeq + 1) {
      awaitingSync = true;
      send("sync");
      return;
    }
    receivedSeq = frame.seq;
    if (frame.type === "stroke") {
      const incoming = frame.stroke as Stroke;
      const last = strokes.value.at(-1);
      if (last?.id === incoming.id) last.points.push(...incoming.points);
      else strokes.value.push(incoming);
    } else if (frame.type === "clear") strokes.value = [];
    else if (frame.type === "undo") strokes.value.pop();
  };
  current.onclose = async () => {
    if (socket !== current || !wantedRoom) return;
    if (!opened && useProtobuf) {
      jsonFallback = true;
      retry = setTimeout(openSocket, 200);
      return;
    }
    connection.value = "reconnecting";
    attempts++;
    if (attempts > 8) {
      notify("房间连接已结束，请重新加入");
      disconnectRoom();
      return;
    }
    // Detect missing rooms and expired sessions before retrying an upgrade.
    if (attempts === 1) {
      try {
        await api("/api/session");
      } catch {
        notify("会话已过期，请重新进入");
        session.value = null;
        disconnectRoom();
        return;
      }
      if (socket !== current || !wantedRoom) return;
    }
    retry = setTimeout(openSocket, Math.min(8000, 600 * 2 ** (attempts - 1)));
  };
  current.onerror = () => current.close();
}
export function disconnectRoom(clearUrl = true) {
  wantedRoom = "";
  clearTimeout(retry);
  const old = socket;
  socket = undefined;
  old?.close();
  connection.value = "offline";
  state.value = null;
  strokes.value = [];
  awaitingSync = false;
  if (clearUrl) {
    const url = new URL(location.href);
    url.searchParams.delete("room");
    history.replaceState({}, "", url);
  }
}
