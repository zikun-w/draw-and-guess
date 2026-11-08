# Debian 部署

本文适用于单台 Debian 服务器。已有 1Panel / OpenResty，或需要 FRP 转发时，参见 [双机部署](../deploy/split/README.md)。

## 准备发布包

在开发机安装 Node.js 22.12+ 和 Go 1.26+，从项目根目录执行：

```bash
bash scripts/build.sh amd64
```

Windows 使用：

```powershell
.\scripts\build.ps1 -Architecture amd64
```

ARM 服务器选择 `arm64`。打包前停止前端开发服务。产物位于 `dist/release-时间-架构/`。

## 安装

将 `game.example.com` 的 DNS A / AAAA 记录指向服务器，确保 80/443 可达。将发布目录上传服务器后执行，替换路径和域名：

```bash
sudo bash /path/to/release/deploy/install.sh /path/to/release game.example.com
```

脚本会安装 Debian 的 Caddy、创建受限的 `drawguess` 用户、安装 systemd 服务，并配置前端及 `/api`、`/ws` 的反向代理。已有 Caddy 配置会追加一次 import，并在应用前校验配置。

此脚本使用 80/443。服务器已有 Nginx、OpenResty 或其他入口时，应复用现有入口，按 [Caddyfile](../deploy/Caddyfile) 中的路径分流规则配置代理。

| 路径 | 作用 |
| --- | --- |
| `/opt/drawguess/releases/版本/` | 各发布版本 |
| `/opt/drawguess/current` | 当前版本符号链接 |
| `/etc/drawguess.env` | 后端环境变量 |
| `/var/lib/drawguess/game.db` | SQLite 数据库 |
| `/etc/caddy/Caddyfile.d/drawguess.caddy` | 站点配置 |

生产环境使用 `COOKIE_SECURE=true`、`TRUST_PROXY=true`，后端绑定 loopback，由同一域名提供前端、API 和 WebSocket。可按资源情况调小 `MAX_ROOMS`。

## 验证与维护

```bash
systemctl status drawguess caddy
journalctl -u drawguess -f
curl -f http://127.0.0.1:8080/healthz
```

再从公网浏览器验证 HTTPS、双人绘画和断线重连。域名刚添加时，客户端 DNS 或代理可能仍缓存旧解析结果，应单独检查解析与服务状态。

更新时重新运行安装脚本，环境文件和数据目录会保留。回滚时将 `/opt/drawguess/current` 指向旧版本，再重启游戏服务并重载 Caddy。更新、回滚或服务重启会结束正在进行的游戏。

备份时使用 SQLite 在线备份命令，或停服后复制数据库。运行中的 WAL 数据库不能只复制主 `.db` 文件；关联日志可能包含尚未合并的事务。

## 外部站点测试

在开发机的 `frontend` 目录执行：

```powershell
$env:E2E_BASE_URL = 'https://game.example.com'
npm run test:e2e
```

DNS 尚未生效时，可以仅对测试浏览器指定映射，示例地址应替换成自己的服务器 IP：

```powershell
$env:E2E_HOST_RULES = 'MAP game.example.com 203.0.113.10'
npm run test:e2e
```

设置 `E2E_HOST_RULES` 时测试浏览器会使用直连，不修改系统 hosts、DNS 或代理设置，也不会跳过 TLS 验证。完成后可移除该环境变量。
