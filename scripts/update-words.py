"""Append one weekly batch of drawable Chinese words from Unicode's public data."""

import argparse
from collections import defaultdict
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import re
import unicodedata
from urllib.request import urlopen


ROOT = Path(__file__).resolve().parents[1] / "backend/internal/game"
WORDS = ROOT / "words.txt"
LAST_WEEK = ROOT / "words.sync-week"
EMOJI = "https://www.unicode.org/Public/emoji/latest/emoji-test.txt"
CLDR = "https://raw.githubusercontent.com/unicode-org/cldr-json/main/cldr-json/cldr-annotations-full/annotations/zh/annotations.json"
BATCH = 12

# Allow only concrete subjects with a visible shape.
GROUPS = {
    "动物": "animal-mammal animal-bird animal-reptile animal-amphibian animal-marine animal-bug",
    "植物": "plant-flower plant-other",
    "食物": "food-fruit food-vegetable food-prepared food-sweet food-asian",
    "交通": "transport-water transport-air",
    "用品": "household tool",
    "穿戴": "clothing",
    "乐器": "musical-instrument",
}
SKIP = set("""
黑色的鸟 蜥蜴类 白花 开花 圆形花饰 根菜 枯萎的花 大野牛 美洲鸵 细菌 服务犬 猴头 狗脸 猫脸
老虎头 马头 奶牛头 猪头 猪鼻子 老鼠头 兔子头 龙头 空巢 有蛋的巢 风吹叶落 无叶树 药草 落叶树
肉块 家禽的腿 一锅食物 碗勺 奶黄 绿色沙拉 绿叶蔬菜 褐色蘑菇 锤子与镐 锤子与扳手 链接 断链
女装 男鞋 女帽 女靴 手袋 乳液瓶 活塞 气泡 沙发和灯 抄网 夹钳 盲杖 螺母与螺栓
座位 空轨 白十字头盔 三角裤 纱丽 蠕虫 炸豆丸子 墨西哥粽子 匕首 炸弹 念珠
""".split())
# Prefer the familiar spelling shared by the game and the source.
ALIASES = {
    "熊猫": "大熊猫", "耗子": "老鼠", "汉堡": "汉堡包", "圆筒冰激凌": "冰淇淋",
    "莲花": "荷花", "煎蛋": "荷包蛋", "盆栽植物": "盆栽", "萨克斯管": "萨克斯",
    "豌豆荚": "豌豆", "夹趾凉鞋": "人字拖", "圆筒冰淇淋": "冰淇淋",
    "青苹果": "苹果", "红苹果": "苹果", "双峰骆驼": "骆驼", "索道": "缆车",
}


def canonical(word):
    name = unicodedata.normalize("NFKC", word).strip()
    return ALIASES.get(name, name)


def candidates(emoji_text, annotations):
    groups = {sub: group for group, subs in GROUPS.items() for sub in subs.split()}
    names = {cp.replace("\ufe0f", ""): entry["tts"][0]
             for cp, entry in annotations.items() if entry.get("tts")}
    result = defaultdict(set)
    group = None
    for line in emoji_text.splitlines():
        if line.startswith("# subgroup: "):
            group = groups.get(line.partition(": ")[2].strip())
        elif group and ";" in line:
            points, status = line.split(";", 1)
            if status.partition("#")[0].strip() != "fully-qualified":
                continue
            cp = "".join(chr(int(p, 16)) for p in points.split()).replace("\ufe0f", "")
            word = canonical(names.get(cp, ""))
            if re.fullmatch(r"[\u4e00-\u9fff]{2,5}", word) and word not in SKIP:
                result[group].add(word)
    if set(result) != set(GROUPS):
        raise ValueError("incomplete source data: a required subject category is missing")
    return result


def select(pool, existing, week):
    def rank(word):
        return hashlib.sha256((week + "/" + word).encode()).digest()

    seen = {canonical(word) for word in existing}
    buckets = [sorted(pool[group] - seen, key=rank, reverse=True)
               for group in sorted(pool, key=rank)]
    chosen = []
    while len(chosen) < BATCH and any(buckets):
        for bucket in buckets:
            while bucket and bucket[-1] in seen:
                bucket.pop()
            if bucket:
                word = bucket.pop()
                seen.add(word)
                chosen.append(word)
                if len(chosen) == BATCH:
                    break
    return chosen


def fetch(url):
    with urlopen(url, timeout=30) as response:
        data = response.read(8 * 1024 * 1024 + 1)
    if len(data) > 8 * 1024 * 1024:
        raise ValueError("unexpectedly large word source")
    return data.decode("utf-8")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()
    year, number, _ = datetime.now(timezone.utc).date().isocalendar()
    week = f"{year}-W{number:02d}"
    if LAST_WEEK.exists() and LAST_WEEK.read_text().strip() == week:
        print(f"{week} already added")
        return
    original = WORDS.read_text(encoding="utf-8")
    existing = original.splitlines()
    if len(existing) < 3 or len(existing) != len(set(existing)) or not all(
            re.fullmatch(r"[\u4e00-\u9fff]{2,8}", word) for word in existing):
        raise ValueError("fix invalid or duplicate entries in the existing word bank first")
    names = json.loads(fetch(CLDR))["annotations"]["annotations"]
    additions = select(candidates(fetch(EMOJI), names), existing, week)
    print(f"{week} +{len(additions)}", " ".join(additions))
    if additions and not args.dry_run:
        WORDS.write_bytes((original.rstrip("\n") + "\n" + "\n".join(additions) + "\n").encode())
        LAST_WEEK.write_bytes((week + "\n").encode())


if __name__ == "__main__":
    main()
