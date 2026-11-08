# 游戏协议 v1

本地 Protobuf 实验增加可选 WebSocket 子协议 `drawguess.pb.v1`，定义见 [drawing.proto](drawing.proto)，测量与回退方式见 [实验报告](../docs/protobuf-experiment.md)。下文的 JSON 消息仍是完整兼容通道。

协商成功后，笔迹命令/广播及完整快照使用二进制 `DrawingPacket`；其余消息仍为文本 JSON。`STROKE` 的 `xy` 为 packed 数字数组，每两个值为一个点；命令最多 96 点，快照共计最多 24,000 点。二进制命令只接受 `STROKE`，携带当前 turn，禁止附带服务端 seq、快照状态或其他笔画。服务器仍校验玩家权限和阶段。

`SNAPSHOT` 的 `state_json` 是按接收者身份过滤后的原状态 JSON，不向猜词者发送候选词或答案。clear/undo 继续使用 JSON 并推进相同的画板 seq。未协商子协议时，整个通道保持 JSON。

生产环境前端、HTTP 和 WebSocket 使用同一站点入口。会话使用 24 小时 HttpOnly、SameSite=Strict cookie；生产开启 Secure。访客身份持久化到 SQLite，在内存缓存。状态的 `version` 字段固定为 `1`。开发由 Vite 代理 `/api`、`/ws`。

## HTTP

| 方法 | 路径 | 请求 / 响应 |
| --- | --- | --- |
| GET | `/healthz` | `{ "status": "ok" }` |
| POST | `/api/session` | `{ "name": "昵称" }` → `{ "id", "name" }`，设置 cookie |
| GET | `/api/session` | 当前会话，无效返回 401 |
| POST | `/api/rooms` | `{ "rounds": 2, "drawSeconds": 90 }` → `{ "id": "8位房间码" }` |
| GET | `/api/rooms/{id}` | 校验会话后查询房间是否存在，不存在返回 404 |
| GET | `/api/history` | 当前会话参与并完成的最近 10 场对局 |
| GET | `/ws?room=ABCDEFG2` | 校验会话、同源校验后升级 WebSocket |

HTTP 错误为 `{ "error": "面向用户的错误信息" }`。JSON 请求最大 4KB，昵称 1–16 字符。房间码是随机的 8 位 Base32 字符。每个来源 IP 的变更请求及连接请求共享令牌桶，容量 12、每秒恢复 2 个。

## 客户端消息

所有帧为 UTF-8 JSON，最大 16KB，服务端严格校验字段。

```json
{"type":"start"}
{"type":"settings","settings":{"rounds":2,"drawSeconds":90}}
{"type":"choose","turn":1,"index":0}
{"type":"guess","text":"大熊猫"}
{"type":"stroke","turn":1,"stroke":{"id":"本轮唯一笔画ID","color":"#292d2b","width":6,"tool":"pen","points":[{"x":1000,"y":2000},{"x":1200,"y":2300}]}}
{"type":"undo","turn":1}
{"type":"clear","turn":1}
{"type":"sync"}
{"type":"leave"}
```

`turn` 为本局画手次序，从 1 开始。绘画操作及选词必须携带当前 turn。坐标归一化为 0–10000 整数；逻辑画布为 1000×650；画笔粗细 1–40；`tool` 为 `pen` 或 `eraser`。橡皮擦使用白色笔迹。

每 40ms 合并一次笔迹，每帧最多 96 个点。同一连续笔画通过相同 ID 追加；只能追加最后一笔，不能修改其颜色、工具或宽度。每轮最多 600 笔 / 24000 点。达到上限可清空。所有输入均由服务端验证，客户端即时绘制仅用于减少本地延迟。

连接令牌桶为 160、每秒补充 80；聊天间隔至少 650ms，长度不超过 120 字符；同步请求最多每 2 秒一次。

## 服务端消息

- `snapshot`：`{ type, state, strokes }`，加入/重连/同步时完整恢复画板。
- `state`：`{ type, state }`，成员、得分、消息、阶段变化。每类接收者分别构造视图。
- `stroke`：`{ type, turn, seq, stroke }`，本房间权威笔迹增量，包含发送者回声。
- `undo` / `clear`：`{ type, turn, seq }`，撤销上一笔 / 清空。
- `error`：`{ type, message }`，可恢复的操作错误。
- `fatal`：`{ type, message }`，无法入房或身份被另一个页面接管，客户端停止重连。

状态字段见 `backend/internal/game/types.go` 的 `State`。`deadline`、`serverTime` 为 Unix 毫秒；前端用服务端时间差显示倒计时，服务端时钟裁决超时。`seq` 为本轮画板操作序号，从 0 开始，笔迹、撤销、清空都递增，序号不连续时请求 `sync`。重连以 snapshot 为准。

阶段为 `lobby → choosing → drawing → reveal → choosing / finished`。选词 12 秒、揭晓 6 秒；绘画 30–180 秒。每人每轮画一次，开局固定轮转名单，游戏中只允许原成员重连；掉线成员的后续画手轮次会跳过。

只有当前画手能收到 `choices` 和 `word`。猜中时发送系统提示，原答案不会作为聊天广播；`revealedWord` 只在揭晓/结束阶段出现。画手和已猜中玩家在绘画阶段不能发言。

发送队列每连接 128 帧、房间事件队列 256 条。队列满或写入超过 5 秒则断开慢连接，通过 snapshot 恢复。心跳每 20 秒，等待 pong 8 秒。掉线座位保留 45 秒，画手掉线 10 秒结束本轮；全部离线 2 分钟、未进入的新房间 5 分钟回收。
