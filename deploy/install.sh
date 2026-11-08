#!/usr/bin/env bash
set -euo pipefail
if [[ $EUID -ne 0 ]]; then echo '请使用 sudo bash deploy/install.sh 发布目录 域名' >&2; exit 1; fi
release_source="$(realpath -- "${1:?缺少发布目录}")"
domain="${2:?缺少域名}"
if [[ ! "$domain" =~ ^[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?$ ]] || [[ "$domain" != *.* ]]; then echo '域名格式有误' >&2; exit 1; fi
[[ -f "$release_source/drawguess" && -f "$release_source/frontend/index.html" ]] || { echo '发布目录不完整' >&2; exit 1; }
apt-get update
apt-get install -y caddy curl ca-certificates
if ! id drawguess &>/dev/null; then useradd --system --home-dir /var/lib/drawguess --shell /usr/sbin/nologin drawguess; fi
release_target="/opt/drawguess/releases/$(date +%Y%m%d-%H%M%S)"
install -d -m 0755 "$release_target" /etc/caddy/Caddyfile.d
install -m 0755 "$release_source/drawguess" "$release_target/drawguess"
cp -R "$release_source/frontend" "$release_target/frontend"
chmod -R a+rX "$release_target/frontend"
if [[ ! -f /etc/drawguess.env ]]; then install -m 0640 -o root -g drawguess "$release_source/deploy/game.env.example" /etc/drawguess.env; fi
install -m 0644 "$release_source/deploy/drawguess.service" /etc/systemd/system/drawguess.service
sed "s/game.example.com/$domain/g" "$release_source/deploy/Caddyfile" > /etc/caddy/Caddyfile.d/drawguess.caddy
if ! grep -Fqx 'import /etc/caddy/Caddyfile.d/*.caddy' /etc/caddy/Caddyfile; then
  printf '\nimport /etc/caddy/Caddyfile.d/*.caddy\n' >> /etc/caddy/Caddyfile
fi
caddy validate --config /etc/caddy/Caddyfile
ln -sfn "$release_target" /opt/drawguess/current
systemctl daemon-reload
systemctl enable --now drawguess
systemctl restart drawguess
for attempt in {1..20}; do if curl -fsS http://127.0.0.1:8080/healthz; then break; fi; sleep 1; done
curl -fsS http://127.0.0.1:8080/healthz
systemctl enable --now caddy
systemctl reload caddy
printf '\n已安装到 %s\n访问：https://%s\n日志：journalctl -u drawguess -f\n' "$release_target" "$domain"
