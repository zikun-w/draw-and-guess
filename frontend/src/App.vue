<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";
import {
  ArrowUpRight,
  ArrowRight,
  Pencil,
  Sparkles,
  Users,
  Copy,
  LogOut,
  Clock3,
  Trophy,
  Send,
  Crown,
  Check,
  X,
  Download,
  Wifi,
  HelpCircle,
  Palette,
  Smile,
  ChevronRight,
  LoaderCircle,
  Plus,
} from "@lucide/vue";
import DrawingBoard from "./components/DrawingBoard.vue";
import {
  api,
  connectRoom,
  connection,
  disconnectRoom,
  isDrawer,
  isHost,
  notify,
  now,
  ranked,
  remaining,
  send,
  session,
  state,
  toast,
} from "./game";
import type { Result, Settings } from "./game";

const nickname = ref(localStorage.getItem("dg_name") || "");
const roomCode = ref(new URLSearchParams(location.search).get("room") || "");
const tab = ref<"create" | "join">(roomCode.value ? "join" : "create");
const busy = ref(false),
  initial = ref(true),
  help = ref(false),
  showHistory = ref(false);
const results = ref<Result[]>([]),
  historyBusy = ref(false);
const chat = ref(""),
  feed = ref<HTMLElement>(),
  board = ref<InstanceType<typeof DrawingBoard>>();
const settings = ref<Settings>({ rounds: 2, drawSeconds: 90 });
let clock: ReturnType<typeof setInterval>;
const inRoom = computed(
  () => state.value !== null || connection.value !== "offline",
);
const drawerName = computed(
  () =>
    state.value?.players.find((p) => p.id === state.value?.drawerId)?.name ||
    "画手",
);
const me = computed(() =>
  state.value?.players.find((p) => p.id === session.value?.id),
);
const canChat = computed(
  () =>
    connection.value === "connected" &&
    !(
      state.value?.phase === "drawing" &&
      (isDrawer.value || me.value?.guessed)
    ),
);
const phaseLabel = computed(
  () =>
    ({
      lobby: "灵感集合中",
      choosing: "挑选灵感",
      drawing: "脑洞进行中",
      reveal: "答案揭晓",
      finished: "本局圆满结束",
    })[state.value?.phase || "lobby"],
);
const connectedPlayers = computed(
  () => state.value?.players.filter((p) => p.connected).length || 0,
);
const avatars = [
  "#e7eedf",
  "#f7e2d5",
  "#e7dff1",
  "#deeaf4",
  "#f7edc9",
  "#f5dfe6",
  "#d9eeea",
  "#e9e4db",
];

async function enter() {
  if (!nickname.value.trim()) {
    notify("先给自己取个昵称吧");
    return;
  }
  if (
    tab.value === "join" &&
    !/^[A-Z2-7]{8}$/.test(roomCode.value.trim().toUpperCase())
  ) {
    notify("请输入 8 位房间码");
    return;
  }
  busy.value = true;
  try {
    session.value = await api("/api/session", { name: nickname.value.trim() });
    localStorage.setItem("dg_name", nickname.value.trim());
    let code = roomCode.value.trim().toUpperCase();
    if (tab.value === "create") {
      const result = await api<{ id: string }>("/api/rooms", settings.value);
      code = result.id;
    }
    connectRoom(code);
  } catch (e) {
    notify((e as Error).message);
  } finally {
    busy.value = false;
  }
}
function leave() {
  send("leave");
  disconnectRoom();
  roomCode.value = "";
  chat.value = "";
}
async function invite() {
  try {
    await navigator.clipboard.writeText(location.href);
    notify("邀请链接已复制，发给朋友一起玩！");
  } catch {
    notify(`房间码：${state.value?.id}，复制后分享给朋友`);
  }
}
function submitGuess() {
  if (chat.value.trim() && send("guess", { text: chat.value.trim() }))
    chat.value = "";
}
async function loadHistory() {
  showHistory.value = true;
  historyBusy.value = true;
  try {
    results.value = await api<Result[]>("/api/history");
  } catch (e) {
    notify((e as Error).message);
  } finally {
    historyBusy.value = false;
  }
}
function updateSettings() {
  send("settings", { settings: settings.value });
}
function onEscape(e: KeyboardEvent) {
  if (e.key === "Escape") {
    help.value = false;
    showHistory.value = false;
  }
}
watch(
  () => state.value?.messages.at(-1)?.id,
  async () => {
    await nextTick();
    feed.value?.scrollTo({ top: feed.value.scrollHeight, behavior: "smooth" });
  },
);
watch(
  () => [state.value?.settings.rounds, state.value?.settings.drawSeconds],
  () => {
    if (state.value) settings.value = { ...state.value.settings };
  },
);
onMounted(async () => {
  clock = setInterval(() => (now.value = Date.now()), 200);
  document.addEventListener("keydown", onEscape);
  try {
    session.value = await api("/api/session");
    nickname.value = session.value!.name;
    if (roomCode.value) connectRoom(roomCode.value.trim().toUpperCase());
  } catch {
    /* New guests enter their nickname. */
  } finally {
    initial.value = false;
  }
});
onBeforeUnmount(() => {
  clearInterval(clock);
  document.removeEventListener("keydown", onEscape);
  disconnectRoom(false);
});
</script>

<template>
  <div class="app-shell">
    <header class="site-header">
      <button
        class="brand"
        aria-label="画猜俱乐部首页"
        @click="inRoom ? leave() : undefined"
      >
        <span class="brand-icon"><Pencil :size="24" :stroke-width="2.3" /></span
        ><span
          >画猜<span class="brand-light">俱乐部</span
          ><small>DRAW & GUESS</small></span
        >
      </button>
      <nav>
        <button @click="help = true">
          <HelpCircle :size="16" /><span>怎么玩</span></button
        ><button @click="loadHistory">
          <Trophy :size="16" /><span>我的战绩</span></button
        ><span class="header-separator"></span
        ><span class="guest-badge"
          ><span class="status-dot"></span
          >{{ session?.name || "快乐，不设门槛" }}</span
        >
      </nav>
    </header>

    <main v-if="!inRoom" class="home">
      <div class="home-main">
        <section class="home-story">
          <span class="edition"
            ><span></span> 好友局，随时开场 <Sparkles :size="14"
          /></span>
          <h1>
            你的灵魂画作，<br />朋友<span class="underline-word">猜得到。</span
            ><svg class="title-spark" viewBox="0 0 50 60" aria-hidden="true">
              <path
                d="m25 5-2 17M43 16 31 28M47 38l-17-2"
                fill="none"
                stroke="currentColor"
                stroke-width="4"
                stroke-linecap="round"
              />
            </svg>
          </h1>
          <p class="hero-description">
            一点画技，一点默契，一大堆意想不到。<br />拉上朋友，用画笔交换你们的奇思妙想。
          </p>
          <div class="hero-tags">
            <span><Check :size="14" /> 无需注册</span
            ><span><Users :size="15" /> 2–8 人同乐</span
            ><span><Pencil :size="14" /> 手机也能画</span>
          </div>
          <div class="doodle-card">
            <span class="tape"></span
            ><span class="doodle-label">今日份灵感练习</span>
            <svg
              viewBox="0 0 340 135"
              role="img"
              aria-label="手绘小猫和猜词气泡"
            >
              <g
                fill="none"
                stroke="#454b40"
                stroke-width="2.5"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <path
                  d="M64 93C44 87 39 67 47 49L43 23l25 16q15-6 32 0l24-18-3 32c11 25-2 43-22 44-12 2-26 2-35-4z"
                  fill="#f4f1dc"
                />
                <path
                  d="M66 60v5m34-6v5m-21 8 5 4 5-4m-5 4v7m0 0q-7 5-11 0m11 0q6 5 11-1M36 67l24 3M35 80l24-4m49-6 26-7m-26 15 25 4"
                />
                <path
                  d="M146 59q11-5 24 0m-7-6 8 6-8 5"
                  stroke-dasharray="3 5"
                />
                <path
                  d="M190 38q0-12 12-12h99q13 0 13 13v43q0 12-12 12h-68l-21 15 4-15h-14q-13 0-13-12z"
                  fill="#fff"
                />
              </g>
              <text
                x="251"
                y="65"
                text-anchor="middle"
                fill="#454b40"
                font-size="18"
                font-family="sans-serif"
              >
                这是…老虎？
              </text>
              <text
                x="251"
                y="83"
                text-anchor="middle"
                fill="#8a8e81"
                font-size="10"
                font-family="sans-serif"
              >
                自信猜错，也是种天赋。
              </text>
              <path
                d="m143 26 3-10m7 14 8-6"
                fill="none"
                stroke="#c78755"
                stroke-width="2.5"
                stroke-linecap="round"
              />
            </svg>
            <span class="doodle-note">画得像不像，快乐都一样。</span>
          </div>
        </section>

        <section class="entry-card">
          <div class="entry-top">
            <span class="section-index">LET’S PLAY</span
            ><span class="tiny-stars">✳</span>
          </div>
          <h2>快乐，就差你一笔。</h2>
          <p>开个房间，或者加入朋友的脑洞现场。</p>
          <div class="entry-tabs" role="tablist" aria-label="游戏方式">
            <button
              role="tab"
              :aria-selected="tab === 'create'"
              :class="{ active: tab === 'create' }"
              @click="tab = 'create'"
            >
              <Plus :size="16" /> 创建房间</button
            ><button
              role="tab"
              :aria-selected="tab === 'join'"
              :class="{ active: tab === 'join' }"
              @click="tab = 'join'"
            >
              <Users :size="16" /> 加入房间
            </button>
          </div>
          <form @submit.prevent="enter">
            <label for="nickname">你的昵称 <span>让朋友一眼认出你</span></label>
            <div class="input-wrap">
              <Smile :size="19" /><input
                id="nickname"
                v-model="nickname"
                maxlength="16"
                placeholder="给自己取个有趣的名字"
                autocomplete="nickname"
                required
              />
            </div>
            <template v-if="tab === 'join'"
              ><label for="room-code">房间码</label>
              <div class="input-wrap">
                <span class="hash-symbol">#</span
                ><input
                  id="room-code"
                  v-model="roomCode"
                  class="code-input"
                  maxlength="8"
                  placeholder="输入 8 位房间码"
                  autocomplete="off"
                  required
                /></div
            ></template>
            <div v-else class="settings-row">
              <div>
                <label for="rounds">游戏轮数</label
                ><select id="rounds" v-model="settings.rounds">
                  <option v-for="n in 5" :key="n" :value="n">{{ n }} 轮</option>
                </select>
              </div>
              <div>
                <label for="seconds">每次绘画</label
                ><select id="seconds" v-model="settings.drawSeconds">
                  <option
                    v-for="n in [30, 60, 90, 120, 180]"
                    :key="n"
                    :value="n"
                  >
                    {{ n }} 秒
                  </option>
                </select>
              </div>
            </div>
            <button
              class="button primary enter-button"
              :disabled="busy || initial"
              type="submit"
            >
              <LoaderCircle
                v-if="busy || initial"
                class="spin"
                :size="19"
              /><template v-else
                >{{ tab === "create" ? "创建我的快乐房间" : "加入朋友的房间"
                }}<ArrowUpRight :size="21"
              /></template>
            </button>
          </form>
          <div class="entry-foot">
            <span class="status-dot"></span> 私人好友房 · 分享链接即可加入
          </div>
        </section>
      </div>
      <section class="how-strip">
        <div class="how-intro">
          <span class="eyebrow">简单三步，快乐加倍</span
          ><strong>不用会画，<br />尽管来玩。</strong>
        </div>
        <div class="how-step">
          <span class="step-number">01</span>
          <div>
            <h3>把朋友叫上</h3>
            <p>一个链接，让快乐集合。</p>
          </div>
        </div>
        <ChevronRight class="step-arrow" :size="18" />
        <div class="how-step">
          <span class="step-number">02</span>
          <div>
            <h3>画出你的脑洞</h3>
            <p>轮流出题，画技自由发挥。</p>
          </div>
        </div>
        <ChevronRight class="step-arrow" :size="18" />
        <div class="how-step">
          <span class="step-number">03</span>
          <div>
            <h3>猜出彼此的默契</h3>
            <p>手速与想象力，一起上线。</p>
          </div>
        </div>
      </section>
    </main>

    <main v-else-if="!state" class="connection-screen">
      <LoaderCircle class="spin" :size="32" />
      <h2>
        {{
          connection === "reconnecting"
            ? "正在重新连接房间…"
            : "正在推开快乐的大门…"
        }}
      </h2>
      <p>正在连接 {{ roomCode || "你的房间" }}</p>
      <button class="button secondary" @click="leave">返回首页</button>
    </main>

    <main v-else class="room-page">
      <div class="room-heading">
        <div>
          <div class="room-breadcrumb">
            好友的脑洞现场 <span>/</span> ROOM {{ state.id }}
          </div>
          <h1>
            {{ phaseLabel
            }}<span class="room-pill"
              ><Users :size="13" />{{ connectedPlayers }}/8</span
            >
          </h1>
        </div>
        <div class="room-heading-actions">
          <button class="button secondary" @click="invite">
            <Copy :size="16" /> 邀请朋友</button
          ><button class="leave-button" @click="leave">
            <LogOut :size="17" /><span>离开</span>
          </button>
        </div>
      </div>
      <div
        v-if="connection !== 'connected'"
        class="connection-banner"
        role="status"
      >
        <Wifi :size="16" /> 连接中断，正在重连。你的座位会保留 45 秒。
      </div>
      <div class="game-layout">
        <aside class="players-panel panel">
          <div class="panel-heading">
            <h2>本场灵魂画手</h2>
            <span>{{ state.players.length }} 位</span>
          </div>
          <div class="players-list">
            <div
              v-for="(player, index) in ranked"
              :key="player.id"
              class="player-row"
              :class="{
                'player-current':
                  player.id === state.drawerId &&
                  ['drawing', 'choosing'].includes(state.phase),
                disconnected: !player.connected,
              }"
            >
              <span class="rank-number">{{ index + 1 }}</span>
              <div
                class="avatar"
                :style="{
                  background:
                    avatars[
                      state.players.findIndex((p) => p.id === player.id) %
                        avatars.length
                    ],
                }"
              >
                {{ player.name.slice(0, 1)
                }}<Pencil
                  v-if="
                    player.id === state.drawerId &&
                    ['drawing', 'choosing'].includes(state.phase)
                  "
                  class="avatar-badge"
                  :size="13"
                /><Check
                  v-else-if="player.guessed"
                  class="avatar-badge guessed"
                  :size="13"
                />
              </div>
              <div class="player-info">
                <strong
                  >{{ player.name
                  }}<small v-if="player.id === session?.id">你</small
                  ><Crown
                    v-if="player.id === state.hostId"
                    :size="12"
                    class="crown" /></strong
                ><span>{{
                  !player.connected
                    ? "暂时离线"
                    : player.guessed
                      ? "已经猜中！"
                      : player.id === state.drawerId &&
                          ["drawing", "choosing"].includes(state.phase)
                        ? "正在挥洒灵感"
                        : "脑洞蓄力中"
                }}</span>
              </div>
              <b class="player-score">{{ player.score }}<small>分</small></b>
            </div>
            <div
              v-if="state.phase === 'lobby'"
              v-for="n in Math.max(0, 4 - state.players.length)"
              :key="`empty-${n}`"
              class="empty-player"
            >
              <span>＋</span><small>虚位以待，等一个脑洞</small>
            </div>
          </div>
          <div class="room-settings">
            <span class="eyebrow">本场设定</span>
            <div>
              <span>游戏轮数</span
              ><select
                v-if="isHost && ['lobby', 'finished'].includes(state.phase)"
                v-model="settings.rounds"
                aria-label="游戏轮数"
                @change="updateSettings"
              >
                <option v-for="n in 5" :value="n" :key="n">
                  {{ n }} 轮
                </option></select
              ><strong v-else>{{ state.settings.rounds }} 轮</strong>
            </div>
            <div>
              <span>每次绘画</span
              ><select
                v-if="isHost && ['lobby', 'finished'].includes(state.phase)"
                v-model="settings.drawSeconds"
                aria-label="绘画时间"
                @change="updateSettings"
              >
                <option v-for="n in [30, 60, 90, 120, 180]" :value="n" :key="n">
                  {{ n }} 秒
                </option></select
              ><strong v-else>{{ state.settings.drawSeconds }} 秒</strong>
            </div>
            <button
              v-if="isHost && ['lobby', 'finished'].includes(state.phase)"
              class="button primary start-button"
              :disabled="connectedPlayers < 2 || connection !== 'connected'"
              @click="send('start')"
            >
              {{ state.phase === "finished" ? "再来一局" : "开始游戏"
              }}<ArrowRight :size="17" />
            </button>
            <p
              v-if="['lobby', 'finished'].includes(state.phase)"
              class="room-hint"
            >
              {{
                connectedPlayers < 2
                  ? "至少 2 人，就能开画"
                  : isHost
                    ? "每位玩家每轮都会当一次画手"
                    : "朋友已就位，等待房主开始"
              }}
            </p>
          </div>
        </aside>
        <section class="canvas-panel panel">
          <div class="canvas-heading">
            <div>
              <span
                class="live-indicator"
                :class="{ live: state.phase === 'drawing' }"
              ></span
              ><strong>{{
                state.phase === "lobby"
                  ? "一块画板，无限可能"
                  : state.phase === "finished"
                    ? "今天的快乐，满分收官"
                    : `${drawerName} 的画板`
              }}</strong
              ><span
                v-if="state.turn && state.phase !== 'lobby'"
                class="turn-chip"
                >{{ Math.min(state.turn, state.totalTurns) }} /
                {{ state.totalTurns }}</span
              >
            </div>
            <button
              class="icon-button"
              title="保存画作"
              aria-label="保存画作"
              @click="board?.savePreview()"
            >
              <Download :size="17" />
            </button>
          </div>
          <div v-if="state.phase === 'drawing'" class="word-bar">
            <span>{{ isDrawer ? "请画出" : "猜猜画的是什么" }}</span
            ><strong v-if="isDrawer" class="secret-word">{{
              state.word
            }}</strong>
            <div v-else class="word-blanks">
              <span v-for="n in state.wordLength" :key="n">?</span>
            </div>
            <div class="timer" :class="{ urgent: remaining <= 15 }">
              <Clock3 :size="16" />{{ remaining }}<small>秒</small>
            </div>
          </div>
          <div
            v-else-if="['choosing', 'reveal'].includes(state.phase)"
            class="word-bar"
          >
            <span>{{
              state.phase === "choosing"
                ? "灵感正在加载，请稍候"
                : "下一位画手准备就绪"
            }}</span>
            <div class="timer">
              <Clock3 :size="16" />{{ remaining }}<small>秒</small>
            </div>
          </div>
          <DrawingBoard ref="board" />
          <div class="canvas-caption">
            <span
              ><Palette :size="14" />{{
                isDrawer && state.phase === "drawing"
                  ? "发挥想象力，尽量用画面表达，别写出答案哦"
                  : state.phase === "lobby"
                    ? "画技不设限，想象力无上限"
                    : "仔细看，答案就藏在画里"
              }}</span
            ><span class="connection-state"
              ><span
                class="status-dot"
                :class="{ offline: connection !== 'connected' }"
              ></span
              >{{ connection === "connected" ? "已连接" : "重连中" }}</span
            >
          </div>
          <div v-if="state.phase === 'finished'" class="finished-overlay">
            <div class="trophy-disc"><Trophy :size="36" /></div>
            <span class="eyebrow">本场最佳脑洞</span>
            <h2>{{ ranked[0]?.name }}</h2>
            <p>以 {{ ranked[0]?.score }} 分收获本场掌声</p>
            <div class="podium">
              <div v-for="(p, i) in ranked.slice(0, 3)" :key="p.id">
                <span>{{ ["冠军", "亚军", "季军"][i] }}</span
                ><strong>{{ p.name }}</strong
                ><b>{{ p.score }}</b>
              </div>
            </div>
            <button v-if="isHost" class="button primary" @click="send('start')">
              再开一局 <ArrowRight :size="18" />
            </button>
            <p v-else>等待房主开启下一局</p>
            <button class="text-button" @click="board?.savePreview()">
              保存最后一幅画作
            </button>
          </div>
        </section>
        <aside class="chat-panel panel">
          <div class="panel-heading">
            <h2>脑洞交流区</h2>
            <span class="chat-heading-icon"><Sparkles :size="16" /></span>
          </div>
          <div class="chat-tip">
            <span>小提示</span
            >猜中越早，得分越高。<br />大胆猜，离谱一点也没关系。
          </div>
          <div ref="feed" class="chat-feed" role="log" aria-label="房间消息">
            <div class="welcome-message">
              <span>✳</span>
              <p>欢迎来到画猜俱乐部<br /><small>今天，也要玩得开心。</small></p>
            </div>
            <div
              v-for="message in state.messages"
              :key="message.id"
              class="message"
              :class="message.kind"
            >
              <template v-if="message.kind === 'chat'"
                ><strong
                  >{{ message.name
                  }}<small v-if="message.name === session?.name"
                    >你</small
                  ></strong
                >
                <p>{{ message.text }}</p></template
              ><template v-else
                ><Check v-if="message.kind === 'correct'" :size="14" /><span>{{
                  message.text
                }}</span></template
              >
            </div>
          </div>
          <form class="chat-form" @submit.prevent="submitGuess">
            <div>
              <input
                v-model="chat"
                maxlength="120"
                :disabled="!canChat"
                :placeholder="
                  !canChat
                    ? me?.guessed
                      ? '猜中了！休息一下…'
                      : '专心画画，让朋友来猜…'
                    : state.phase === 'drawing'
                      ? '输入你的答案…'
                      : '和大家打个招呼…'
                "
                aria-label="猜词或聊天"
                autocomplete="off"
              /><button
                type="submit"
                :disabled="!chat.trim() || !canChat"
                aria-label="发送"
              >
                <Send :size="17" />
              </button>
            </div>
            <small>按 Enter 发送 · 友好交流，快乐加倍</small>
          </form>
        </aside>
      </div>
    </main>

    <footer class="site-footer">
      <span
        >画猜俱乐部 <span class="footer-dot">·</span> 每一笔，都是快乐。</span
      ><span>MADE FOR GOOD TIMES <span>✳</span></span>
    </footer>
    <Transition name="toast"
      ><div v-if="toast" class="toast-message" role="status">
        <Check :size="17" />{{ toast }}
      </div></Transition
    >
    <div
      v-if="help || showHistory"
      class="modal-backdrop"
      @click.self="
        help = false;
        showHistory = false;
      "
    >
      <section
        class="modal"
        role="dialog"
        aria-modal="true"
        :aria-label="help ? '怎么玩' : '我的战绩'"
      >
        <button
          class="modal-close icon-button"
          aria-label="关闭"
          @click="
            help = false;
            showHistory = false;
          "
        >
          <X :size="21" /></button
        ><template v-if="help"
          ><span class="eyebrow">HOW TO PLAY</span>
          <h2>画不在好，懂你就行。</h2>
          <ol class="rules">
            <li>
              <strong>朋友集合</strong>
              <p>创建房间，把邀请链接或 8 位房间码发给朋友。2–8 人就能开玩。</p>
            </li>
            <li>
              <strong>轮流开画</strong>
              <p>
                每个人每轮画一次，从三个词中选一个。12
                秒未选择会自动选词，用画面表达，避免直接写答案。
              </p>
            </li>
            <li>
              <strong>大胆猜词</strong>
              <p>
                在聊天框输入答案。猜中获得 100–500
                分，越快越高；每有一人猜中，画手获得 100 分。
              </p>
            </li>
            <li>
              <strong>快乐继续</strong>
              <p>
                全部猜中或时间结束后公布答案。短暂掉线可在 45
                秒内重连，画手离线超过 10 秒会跳过。
              </p>
            </li>
          </ol></template
        ><template v-else
          ><span class="eyebrow">YOUR LITTLE WINS</span>
          <h2>每一局，都值得记住。</h2>
          <p class="muted">当前浏览器身份最近的 10 场已完成对局</p>
          <div v-if="historyBusy" class="history-empty">
            <LoaderCircle class="spin" :size="25" />
          </div>
          <div v-else-if="!results.length" class="history-empty">
            <Trophy :size="36" />
            <h3>你的高光时刻，正在路上。</h3>
            <p>完成一局游戏，战绩就会出现在这里。</p>
          </div>
          <div v-else class="history-list">
            <article v-for="result in results" :key="result.id">
              <div>
                <strong>{{ result.room }}</strong
                ><time>{{
                  new Date(result.finishedAt).toLocaleString("zh-CN")
                }}</time>
              </div>
              <p
                v-for="p in [...result.players].sort(
                  (a, b) => b.score - a.score,
                )"
                :key="p.id"
              >
                <span
                  >{{ p.name }}{{ p.id === session?.id ? "（你）" : "" }}</span
                ><strong>{{ p.score }} 分</strong>
              </p>
            </article>
          </div></template
        >
      </section>
    </div>
  </div>
</template>
