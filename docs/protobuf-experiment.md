# Protobuf 本地实验

结论：保留在本地 `experiment/protobuf` 分支。带宽节省明确，密集笔迹的 CPU 成本下降；常驻内存变化很小。未部署线上，也未推送此实验分支。

基线为提交 `cf81408`。服务器、前端和负载程序在同一开发机验证，未经过 FRP、公网、TLS 或生产数据库。

## 方案

- 通过 WebSocket 子协议 `drawguess.pb.v1` 协商。
- 高频笔迹与重连画板快照使用 Protobuf 二进制帧，状态与聊天等控制消息保留 JSON。
- 坐标使用 packed `uint32` 数组，顺序为 `x0,y0,x1,y1,...`。
- 快照的身份视图先由原房间逻辑过滤，再编码；状态部分继续以 JSON 携带。
- 服务端支持 JSON 客户端及混合房间；新客户端协商失败时自动尝试旧 JSON 通道。
- 每种编码只为公开笔迹广播编码一次，房间内复用同一字节缓冲。

协议定义为 [drawing.proto](../protocol/drawing.proto)。生成的 Go、JavaScript 与类型声明随代码保存，正常构建不需要安装编译器。改动 schema 后需安装 `protoc`、`protoc-gen-go`，运行：

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12
cd frontend
npm ci
npm run proto:generate
```

确保 `protoc` 和 `protoc-gen-go` 在 PATH 中。生成脚本也支持仓库内 `.tools` 的本地工具，以及 `PROTOC` 指定的编译器。

## 对照负载

50 个房间、400 个连接，每房间一名画手；每 40ms 一批，共 200 批，分别测试每批 3 点和 24 点。每种场景独立运行 3 次，交替运行 JSON 与 Protobuf，表中取中位数。每次预期 80,000 次广播，全部送达，业务错误为 0，连接清理后 goroutine 回到 1。

两种格式均完整解码笔迹。CPU、堆内存和分配量包含同进程的服务端、客户端及生成器开销；以下不是生产服务的独立 RSS 测量。

| 指标 | 3 点 JSON | 3 点 Protobuf | 24 点 JSON | 24 点 Protobuf |
| --- | ---: | ---: | ---: | ---: |
| 笔迹广播总字节 | 15,974,800 | 5,697,600 | 49,336,000 | 12,497,600 |
| CPU 时间，秒 | 5.12 | 4.65 | 6.80 | 4.65 |
| P95 回环延迟，ms | 3.78 | 3.70 | 4.30 | 3.81 |
| 强制 GC 后保留堆，MiB | 27.20 | 26.99 | 30.46 | 30.37 |
| 测量期间累计分配，MiB | 181.29 | 195.87 | 385.72 | 257.79 |

笔迹流量分别下降 64.3% / 74.7%，CPU 时间下降 9.2% / 31.7%。小批次分配量增加约 8%，保留堆基本相同。低负载 P95 的差异很小，不应外推为公网延迟改善。

复现命令，在 `backend` 目录执行；对比时不要同时运行其他压测：

```bash
go run ./cmd/loadtest -rooms 50 -players 8 -batches 200 -interval 40ms -points 3 -format json
go run ./cmd/loadtest -rooms 50 -players 8 -batches 200 -interval 40ms -points 3 -format protobuf
# 将 -points 改为 24 测试密集笔迹；各自重复 3 次
```

## 编解码与前端成本

Go 微基准采用 `BenchmarkWire`、`-benchtime=300ms -count=3`，下表取中位数，包含领域对象与 Protobuf 对象的转换成本。

| 操作 | JSON | Protobuf |
| --- | ---: | ---: |
| Go 24 点广播编码 | 2.60 µs | 0.47 µs |
| Go 24 点命令解码 | 7.33 µs | 0.48 µs |
| Go 24,000 点快照编码 | 1.89 ms | 0.19 ms |
| Go 24,000 点快照解码 | 4.62 ms | 0.23 ms |

Edge 浏览器另外测了真实前端函数，7 次循环取中位数：24 点命令编码约从 1.04 µs 增至 1.49 µs，24 点广播解码从 1.41 µs 降至 0.63 µs；24,000 点快照解码从 1.78ms 降至 0.78ms。数据仅代表这台开发机，未包含绘画渲染。

前端 JS 生产包 gzip 体积约从 44.69 KB 增至 58.71 KB，增加约 14 KB。生成代码与运行库也增加了协议维护成本。

## 验证与回退

- Go 测试、Debian 竞态检查、`go vet`、TypeScript 检查和前端生产构建通过。
- Protobuf 模式下通过完整双人对局、手机触摸、刷新重连、答案隔离和战绩等 4 项浏览器测试，断言实际收发了二进制帧。
- 另用旧版 JSON 后端完成双人对局及重连测试，确认协商失败后的 JSON 回退。
- 混合 JSON / Protobuf 客户端的广播、快照及答案隔离通过 Go WebSocket 集成测试。
- 校验奇数坐标数组、越界坐标、超大笔迹批次、不支持的命令类型和损坏消息。

当前实验仅保存在本地分支。完整回退可在工作区干净时切回 `main`，然后在前端执行 `npm ci`、`npm run build` 恢复基线依赖与构建产物，并从基线源码重新启动后端。仅强制客户端使用 JSON 时，在启动 Vite / 构建前设置 `VITE_WIRE_FORMAT=json`。

原始负载 JSON、Go 微基准和浏览器数据保存在本机忽略目录 `.local/experiments/protobuf/`。
