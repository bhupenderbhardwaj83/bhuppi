#!/usr/bin/env python3
"""Extract command chapters from the owned epub into internal/atlas/data.json.

The binary embeds that JSON. Runtime does not open the epub.
"""

from __future__ import annotations

import html
import json
import re
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
EPUB = ROOT / "ebook" / "The-Story-of-Linux-part-6.epub"
OUT = ROOT / "internal" / "atlas" / "data.json"

# This callout is one nmap warning. The epub repeats it on every nmap use case.
# Keep a single page notice. Do not copy it onto other commands.
LAB_SCOPE = (
    "Authorization is required for every target. Discovery, version, timing, "
    "scripting, spoofing, and packet options can trigger monitoring or affect "
    "fragile systems; the target placeholders must be replaced only with an "
    "approved lab scope."
)


def textify(fragment: str) -> str:
    fragment = re.sub(r"<[^>]+>", " ", fragment)
    fragment = html.unescape(fragment)
    fragment = fragment.replace("\xa0", " ")
    return re.sub(r"\s+", " ", fragment).strip()


def li_parts(raw: str) -> tuple[str, str]:
    m = re.match(
        r"\s*<(?:code|strong)>(.*?)</(?:code|strong)>\s*(.*)\s*\Z",
        raw,
        re.S,
    )
    if not m:
        return "", textify(raw)
    label = textify(m.group(1)).rstrip(":").strip()
    return label, textify(m.group(2))


def parse_tool(name: str, doc: str) -> dict:
    body = re.search(r"<body>(.*)</body>", doc, re.S)
    chunk = body.group(1) if body else doc
    h1 = re.search(r"<h1[^>]*>(.*?)</h1>", chunk, re.S)
    title = textify(h1.group(1)) if h1 else Path(name).stem.removeprefix("atlas_tool_")
    syn = re.search(r'class="command-syntax">(.*?)</p>', chunk, re.S)
    syntax = ""
    if syn:
        syntax = textify(re.sub(r"<strong>\s*Syntax\s*</strong>", "", syn.group(1), flags=re.I))
    desc_m = re.search(r'class="command-logic">(.*?)</p>', chunk, re.S)
    description = textify(desc_m.group(1)) if desc_m else ""

    def section_lists(heading: str) -> list[tuple[str, str]]:
        m = re.search(
            rf"<h2>\s*{re.escape(heading)}\s*</h2>\s*<ul[^>]*>(.*?)</ul>",
            chunk,
            re.S,
        )
        if not m:
            return []
        items = []
        for li in re.findall(r"<li>(.*?)</li>", m.group(1), re.S):
            label, text = li_parts(li)
            if text or label:
                items.append({"label": label, "text": text})
        return items

    cases = []
    for block in re.findall(
        r'<h3 class="use-case-title">(.*?)</h3>(.*?)(?=<h3 class="use-case-title">|</body>)',
        chunk,
        re.S,
    ):
        title_c, rest = block
        code = re.search(r"<code>(.*?)</code>", rest, re.S)
        what = re.search(r'class="command-purpose">(.*?)</p>', rest, re.S)
        parts = [textify(li) for li in re.findall(r'class="command-breakdown"[^>]*>(.*?)</ul>', rest, re.S)]
        # parts captured the ul; split lis from the first ul
        ul = re.search(r'class="command-breakdown"[^>]*>(.*?)</ul>', rest, re.S)
        bullets = []
        if ul:
            bullets = [textify(li) for li in re.findall(r"<li>(.*?)</li>", ul.group(1), re.S)]
            bullets = [b for b in bullets if b]
        note = ""
        call = re.search(r'class="callout[^"]*">(.*?)</p>', rest, re.S)
        if call:
            note = textify(call.group(1))
        cases.append(
            {
                "title": textify(title_c),
                "command": textify(code.group(1)) if code else "",
                "what": textify(re.sub(r"<strong>\s*What it does:\s*</strong>", "", what.group(1), flags=re.I))
                if what
                else "",
                "parts": bullets,
                "note": note,
            }
        )
    page = {
        "name": title,
        "syntax": syntax,
        "description": description,
        "anatomy": section_lists("Anatomy & Mechanics") or section_lists("Anatomy &amp; Mechanics"),
        "origins": section_lists("Origins and purpose"),
        "cases": cases,
    }
    lift_lab_scope(page)
    return page


def lab_scope_note(note: str) -> bool:
    low = re.sub(r"\s+", " ", note).strip().lower()
    if not low:
        return False
    if "approved lab scope" in low:
        return True
    return "authorization is required for every target" in low and "fragile" in low


def lift_lab_scope(page: dict) -> None:
    """Drop the repeated nmap authorization callout from each use case.

    When the command is nmap, keep that sentence once as page["notice"].
    """
    found = False
    for case in page["cases"]:
        if lab_scope_note(case.get("note", "")):
            found = True
            case["note"] = ""
    if found and page["name"].lower() == "nmap":
        page["notice"] = LAB_SCOPE


def main() -> None:
    book = []
    with zipfile.ZipFile(EPUB) as z:
        names = sorted(n for n in z.namelist() if re.search(r"atlas_tool_.+\.xhtml$", n))
        for name in names:
            page = parse_tool(name, z.read(name).decode("utf-8"))
            if page["name"]:
                book.append(page)
    book.sort(key=lambda p: p["name"].lower())
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(json.dumps({"commands": book}, ensure_ascii=False, indent=1) + "\n", encoding="utf-8")
    awk = next(p for p in book if p["name"] == "awk")
    cmds = " ".join(c["command"] for c in awk["cases"][:3])
    if "awk [OPTION]" not in awk["syntax"] and "OPTION" not in awk["syntax"]:
        raise SystemExit(f"awk syntax missing: {awk['syntax']!r}")
    if "/etc/passwd" not in cmds or "ERROR" not in cmds or "printf" not in cmds:
        raise SystemExit("awk use cases 1-3 missing expected commands")
    print(f"commands={len(book)} awk_cases={len(awk['cases'])} wrote {OUT}")


if __name__ == "__main__":
    main()
