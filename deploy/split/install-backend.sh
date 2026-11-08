#!/usr/bin/env bash
set -euo pipefail
[[ $EUID -eq 0 ]] || { echo 'Run with sudo' >&2; exit 1; }
stage="$(realpath -- "${1:?stage directory required}")"
version="${2:?release identifier required}"
[[ "$version" =~ ^[a-zA-Z0-9._-]+$ ]] || exit 1
[[ -f "$stage/drawguess" && -f "$stage/drawguess.service" ]] || exit 1
if ! id drawguess >/dev/null 2>&1; then
    useradd --system --home-dir /var/lib/drawguess --shell /usr/sbin/nologin drawguess
fi
target="/opt/drawguess/releases/$version"
install -d -m 0755 "$target" /etc/systemd/system/drawguess.service.d
install -m 0755 "$stage/drawguess" "$target/drawguess"
install -m 0644 "$stage/drawguess.service" /etc/systemd/system/drawguess.service
install -m 0644 "$stage/limits.conf" /etc/systemd/system/drawguess.service.d/limits.conf
if [[ ! -f /etc/drawguess.env ]]; then
    install -m 0640 -o root -g drawguess "$stage/drawguess.env.example" /etc/drawguess.env
fi
ln -sfn "$target" /opt/drawguess/current
systemctl daemon-reload
systemctl enable drawguess
systemctl restart drawguess
for attempt in {1..20}; do
    if curl -fsS http://127.0.0.1:8080/healthz; then
        printf '\nGame backend is healthy\n'
        exit 0
    fi
    sleep 1
done
journalctl -u drawguess -n 20 --no-pager
exit 1
