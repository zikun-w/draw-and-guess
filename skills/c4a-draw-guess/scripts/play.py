"""A small HTTP client for one draw-and-guess player. Python 3, standard library."""

import argparse
import http.cookiejar
import json
import os
import re
import sys
import tempfile
from pathlib import Path
from urllib import error, parse, request


def write_private(path, data):
    with tempfile.NamedTemporaryFile(dir=path.parent, delete=False) as handle:
        tmp = Path(handle.name)
        try:
            handle.write(data)
            handle.close()
            os.replace(tmp, path)
        finally:
            if tmp.exists():
                tmp.unlink()


class NoRedirect(request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--profile", default=os.getenv("DRAW_PROFILE", "default"))
    parser.add_argument("--site", default=os.getenv("DRAW_SITE", "https://draw.elysium.org.cn"))
    actions = parser.add_subparsers(dest="op", required=True)
    create = actions.add_parser("create")
    create.add_argument("name")
    create.add_argument("--rounds", type=int, default=2)
    create.add_argument("--seconds", type=int, default=90)
    join = actions.add_parser("join")
    join.add_argument("name")
    join.add_argument("room", help="room code or invitation link")
    for op in ("state", "start", "leave", "undo", "clear"):
        actions.add_parser(op)
    actions.add_parser("choose").add_argument("index", type=int)
    actions.add_parser("guess").add_argument("text")
    for op in ("sketch", "draw"):
        actions.add_parser(op).add_argument("shapes", help="JSON array of shapes")
    actions.add_parser("board").add_argument("file", type=Path)
    args = parser.parse_args()

    if not re.fullmatch(r"[A-Za-z0-9_-]{1,48}", args.profile):
        parser.error("profile 应使用 1–48 个英文字母、数字、下划线或连字符")
    site = args.site.rstrip("/")
    url = parse.urlsplit(site)
    if (not url.hostname or url.username or url.password or url.query or url.fragment or url.path
            or not (url.scheme == "https" or (url.scheme == "http" and url.hostname in ("localhost", "127.0.0.1", "::1")))):
        parser.error("site 需要 HTTPS 站点根地址，本机联调可用 HTTP")
    root = Path(os.getenv("XDG_STATE_HOME") or Path.home() / ".local/state") / "c4a-draw-guess" / args.profile
    root.mkdir(mode=0o700, parents=True, exist_ok=True)
    memo = root / "seat.json"
    seat = json.loads(memo.read_text(encoding="utf-8")) if memo.exists() else {"site": site}
    if seat["site"] != site:
        parser.error("不同站点请使用不同的 profile")
    jar = http.cookiejar.MozillaCookieJar(str(root / "cookies.txt"))
    if Path(jar.filename).exists():
        jar.load()
    opener = request.build_opener(request.HTTPCookieProcessor(jar), NoRedirect())

    def fetch(path, body=None, binary=False):
        payload = None if body is None else json.dumps(body, ensure_ascii=False).encode()
        req = request.Request(site + path, data=payload, headers={"Content-Type": "application/json"})
        with opener.open(req, timeout=10) as resp:
            data = resp.read()
            if binary:
                if resp.headers.get_content_type() != "image/png":
                    raise ValueError("接口没有返回 PNG，请检查服务端版本")
                return data
            return json.loads(data) if data else {}

    def save():
        write_private(memo, json.dumps(seat).encode())

    if args.op in ("join", "create"):
        room = None
        if args.op == "join":
            room = args.room.strip()
            if "://" in room:
                invite = parse.urlsplit(room)
                if (invite.scheme, invite.netloc) != (url.scheme, url.netloc):
                    parser.error("邀请链接和游戏站点不一致")
                room = parse.parse_qs(invite.query).get("room", [""])[0]
            room = room.upper()
            if not re.fullmatch(r"[A-Z2-7]{8}", room):
                parser.error("需要有效的 8 位房间码")
        fetch("/api/session", {"name": args.name})
        jar.save()
        if room is None:
            room = fetch("/api/rooms", {"rounds": args.rounds, "drawSeconds": args.seconds})["id"]
            print("邀请链接 " + site + "/?room=" + room, file=sys.stderr)
        base = "/api/agent/rooms/" + room
        view = fetch(base + "/join", {})
        seat["room"] = room
    else:
        if "room" not in seat:
            parser.error("先用 create 或 join 入座")
        base = "/api/agent/rooms/" + seat["room"]
        if args.op == "board":
            write_private(args.file, fetch(base + "/board.png", binary=True))
            print(args.file)
            return
        if args.op == "state":
            view = fetch(base + "/state")
        else:
            body = {"type": args.op, "roundKey": seat["roundKey"]}
            if args.op in ("sketch", "draw", "undo", "clear"):
                body["seq"] = seat["seq"]
            if args.op in ("sketch", "draw"):
                body["shapes"] = json.loads(args.shapes)
            if args.op == "choose":
                body["index"] = args.index
            if args.op == "guess":
                body["text"] = args.text
            view = fetch(base + "/actions", body)
            if args.op == "leave":
                seat.pop("room")
                save()
                print("已离开房间")
                return
    state = view["state"]
    seat.update(roundKey=view["roundKey"], seq=state["seq"])
    save()
    brief = {k: state[k] for k in ("phase", "turn", "totalTurns", "seq", "wordLength", "choices", "word", "revealedWord") if k in state}
    brief.update(room=seat["room"], next=view["next"], players=[
        {"name": p["name"], "me": p["id"] == view["me"], "score": p["score"], "online": p["connected"], "guessed": p["guessed"]}
        for p in state["players"]
    ])
    if state["deadline"]:
        brief["secondsLeft"] = max(0, (state["deadline"] - state["serverTime"]) // 1000)
    print(json.dumps(brief, ensure_ascii=False))


if __name__ == "__main__":
    try:
        main()
    except error.HTTPError as exc:
        try:
            reason = json.load(exc)["error"]
        except (ValueError, KeyError, TypeError):
            reason = exc.reason
        sys.exit("HTTP %s %s" % (exc.code, reason))
    except (OSError, ValueError) as exc:
        sys.exit(str(exc))
