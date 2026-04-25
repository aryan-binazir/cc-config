#!/usr/bin/env python3
"""Convert a Notion page (page metadata + block tree) into Obsidian markdown.

Input:
  --page-json    path to a JSON file containing the Notion page object
                 (as returned by get_page / pages.retrieve)
  --blocks-json  path to a JSON file containing the recursive block tree,
                 where each block has a "children" array (empty if leaf).
                 The skill's walker is responsible for building this tree.
  --target-path  the intended Obsidian vault-relative path for this page,
                 e.g. "Notion/Projects/Acme rebrand/_index.md"
                 (affects wikilink resolution but doesn't write anywhere)
  --synced-pages-json (optional) path to a JSON file mapping
                 notion_page_id -> vault-relative path for pages already
                 synced. Used to decide whether a page mention becomes a
                 wikilink or an external Notion link.
  --out          (optional) write to this file; otherwise stdout

Output: a complete markdown document with frontmatter, printed to stdout
(or the --out path).

Stderr: a JSON blob summarizing unhandled block types, e.g.
{"unhandled": {"unsupported": 2, "breadcrumb": 1}}

This script is pure Python (stdlib only) so it runs anywhere.

The mapping logic follows references/block-mapping.md. When in doubt, that
reference file is authoritative — if behavior diverges, fix the script
rather than changing the reference.
"""
from __future__ import annotations

import argparse
import json
import sys
from collections import Counter
from pathlib import Path
from typing import Any

# --- Frontmatter -----------------------------------------------------------

FORBIDDEN_KEY_CHARS = set(" :/\\.")


def _yaml_scalar(value: Any) -> str:
    """Serialize a scalar for YAML frontmatter. Conservative quoting."""
    if value is None:
        return "null"
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, (int, float)):
        return str(value)
    s = str(value)
    # Quote if the string contains characters that would confuse YAML parsers,
    # or if it could be misread as a different type.
    if (
        s == ""
        or s != s.strip()
        or any(c in s for c in ":#\n\"'[]{}")
        or s.lower() in ("true", "false", "null", "yes", "no", "~")
        or s.startswith(("-", "?", "&", "*", "!", "|", ">", "%", "@", "`"))
    ):
        escaped = s.replace("\\", "\\\\").replace('"', '\\"')
        return f'"{escaped}"'
    return s


def _yaml_value(value: Any, indent: int = 0) -> str:
    """Serialize a value. Lists become block arrays, dicts become nested mappings."""
    if isinstance(value, list):
        if not value:
            return "[]"
        lines = []
        for item in value:
            if isinstance(item, (dict, list)):
                nested = _yaml_value(item, indent + 2)
                lines.append(" " * indent + "- " + nested.lstrip())
            else:
                lines.append(" " * indent + "- " + _yaml_scalar(item))
        return "\n" + "\n".join(lines)
    if isinstance(value, dict):
        if not value:
            return "{}"
        lines = []
        for k, v in value.items():
            key = _yaml_key(k)
            if isinstance(v, (dict, list)):
                lines.append(" " * indent + f"{key}:" + _yaml_value(v, indent + 2))
            else:
                lines.append(" " * indent + f"{key}: {_yaml_scalar(v)}")
        return "\n" + "\n".join(lines)
    return _yaml_scalar(value)


def _yaml_key(k: str) -> str:
    if any(c in FORBIDDEN_KEY_CHARS for c in k) or not k:
        return f'"{k}"'
    return k


def build_frontmatter(fields: dict[str, Any]) -> str:
    lines = ["---"]
    for k, v in fields.items():
        key = _yaml_key(k)
        if isinstance(v, (dict, list)):
            lines.append(f"{key}:" + _yaml_value(v, 2))
        else:
            lines.append(f"{key}: {_yaml_scalar(v)}")
    lines.append("---")
    return "\n".join(lines) + "\n\n"


# --- Rich text -------------------------------------------------------------


def render_rich_text(rt: list[dict[str, Any]], synced: dict[str, str]) -> str:
    out = []
    for span in rt or []:
        out.append(render_one_rich_text(span, synced))
    return "".join(out)


def render_one_rich_text(span: dict[str, Any], synced: dict[str, str]) -> str:
    t = span.get("type", "text")
    if t == "equation":
        expr = span.get("equation", {}).get("expression", "")
        return f"${expr}$"
    if t == "mention":
        return _render_mention(span, synced)

    # default: plain text with annotations + optional href
    text_obj = span.get("text") or {}
    content = text_obj.get("content", span.get("plain_text", ""))
    href = text_obj.get("link", {}).get("url") if isinstance(text_obj.get("link"), dict) else span.get("href")
    s = _apply_annotations(content, span.get("annotations", {}))
    if href:
        s = f"[{s}]({href})"
    return s


def _apply_annotations(text: str, ann: dict[str, Any]) -> str:
    if not text:
        return text
    if ann.get("code"):
        text = f"`{text}`"
    if ann.get("strikethrough"):
        text = f"~~{text}~~"
    if ann.get("underline"):
        text = f"<u>{text}</u>"
    if ann.get("italic"):
        text = f"*{text}*"
    if ann.get("bold"):
        text = f"**{text}**"
    return text


def _render_mention(span: dict[str, Any], synced: dict[str, str]) -> str:
    m = span.get("mention", {})
    mtype = m.get("type")
    plain = span.get("plain_text", "")
    if mtype == "user":
        return plain or "@user"
    if mtype == "page":
        page_id = m.get("page", {}).get("id")
        if page_id and page_id in synced:
            title = Path(synced[page_id]).stem
            return f"[[{title}]]"
        href = span.get("href", "")
        return f"[{plain}]({href})" if href else plain
    if mtype == "database":
        db_id = m.get("database", {}).get("id")
        if db_id and db_id in synced:
            idx = synced[db_id]
            stem = Path(idx).stem
            return f"[[{stem}]]"
        href = span.get("href", "")
        return f"[{plain}]({href})" if href else plain
    if mtype == "date":
        d = m.get("date", {})
        start = d.get("start", "")
        end = d.get("end")
        return f"{start} → {end}" if end else start
    if mtype in ("link_mention", "link_preview"):
        href = span.get("href") or m.get(mtype, {}).get("href", "")
        title = plain or href
        return f"[{title}]({href})"
    # template_mention and anything else: fall back to plain text
    return plain


# --- Callouts --------------------------------------------------------------

CALLOUT_BY_EMOJI = {
    "ℹ️": "info", "💡": "info",
    "⚠️": "warning", "⚠": "warning",
    "🚨": "danger", "🛑": "danger", "❗": "danger",
    "✅": "success", "👍": "success",
    "❌": "failure", "👎": "failure",
    "📝": "note", "📌": "note", "🗒️": "note",
    "❓": "question", "❔": "question",
    "💬": "quote",
    "📋": "todo",
    "🎯": "tip", "⭐": "tip",
}


def _callout_type(block_data: dict[str, Any]) -> tuple[str, str]:
    """Return (obsidian_callout_type, emoji_prefix_or_empty)."""
    icon = block_data.get("icon") or {}
    if icon.get("type") == "emoji":
        emoji = icon.get("emoji", "")
        if emoji in CALLOUT_BY_EMOJI:
            return CALLOUT_BY_EMOJI[emoji], ""
        return "note", emoji + " " if emoji else ""
    return "note", ""


# --- Blocks ----------------------------------------------------------------


def render_blocks(
    blocks: list[dict[str, Any]],
    synced: dict[str, str],
    unhandled: Counter,
    indent: int = 0,
) -> list[str]:
    """Render a list of Notion blocks to markdown lines.

    Returns a list of markdown line chunks; caller joins with "\n".
    """
    out: list[str] = []
    i = 0
    while i < len(blocks):
        block = blocks[i]
        t = block.get("type")

        # List items are grouped to avoid inserting blank lines between siblings.
        if t in ("bulleted_list_item", "numbered_list_item", "to_do"):
            run = [blocks[i]]
            while i + 1 < len(blocks) and blocks[i + 1].get("type") == t:
                i += 1
                run.append(blocks[i])
            out.append(_render_list_run(t, run, synced, unhandled, indent))
            i += 1
            continue

        out.append(_render_block(block, synced, unhandled, indent))
        i += 1

    return [chunk for chunk in out if chunk]


def _render_block(
    block: dict[str, Any],
    synced: dict[str, str],
    unhandled: Counter,
    indent: int,
) -> str:
    t = block.get("type")
    data = block.get(t, {}) if t else {}
    children = block.get("children") or []
    pad = " " * indent

    if t == "paragraph":
        return pad + render_rich_text(data.get("rich_text", []), synced)

    if t == "heading_1":
        return pad + "# " + render_rich_text(data.get("rich_text", []), synced)
    if t == "heading_2":
        return pad + "## " + render_rich_text(data.get("rich_text", []), synced)
    if t == "heading_3":
        return pad + "### " + render_rich_text(data.get("rich_text", []), synced)

    if t == "quote":
        body = render_rich_text(data.get("rich_text", []), synced)
        lines = body.splitlines() or [""]
        return "\n".join(pad + "> " + ln for ln in lines)

    if t == "callout":
        ctype, emoji_prefix = _callout_type(data)
        title = emoji_prefix + render_rich_text(data.get("rich_text", []), synced)
        lines = [pad + f"> [!{ctype}] {title}".rstrip()]
        if children:
            child_md = "\n".join(render_blocks(children, synced, unhandled, 0))
            for ln in child_md.splitlines():
                lines.append(pad + "> " + ln)
        return "\n".join(lines)

    if t == "toggle":
        summary = render_rich_text(data.get("rich_text", []), synced)
        child_md = "\n\n".join(render_blocks(children, synced, unhandled, 0))
        return f"{pad}<details>\n{pad}<summary>{summary}</summary>\n\n{child_md}\n\n{pad}</details>"

    if t == "divider":
        return pad + "---"

    if t == "code":
        lang = data.get("language", "")
        if lang == "plain text":
            lang = ""
        content = "".join(span.get("plain_text", "") for span in data.get("rich_text", []))
        caption = render_rich_text(data.get("caption", []), synced)
        lines = [pad + f"```{lang}"]
        for ln in content.splitlines() or [""]:
            lines.append(pad + ln)
        lines.append(pad + "```")
        if caption:
            lines.append(pad + f"_{caption}_")
        return "\n".join(lines)

    if t == "equation":
        return pad + f"$${data.get('expression', '')}$$"

    if t == "image":
        return _render_media(block, data, synced, kind="image")
    if t in ("video", "audio", "pdf", "file"):
        return _render_media(block, data, synced, kind=t)

    if t == "bookmark":
        url = data.get("url", "")
        caption = render_rich_text(data.get("caption", []), synced)
        line = pad + f"[{url}]({url})"
        if caption:
            line += f"\n{pad}_{caption}_"
        return line

    if t in ("embed", "link_preview"):
        url = data.get("url", "")
        return pad + f"<!-- Notion {t} -->\n{pad}[{url}]({url})"

    if t == "table":
        return _render_table(block, synced)

    if t == "column_list":
        md = "\n\n".join(render_blocks(children, synced, unhandled, 0))
        return f"<!-- Notion columns flattened ({len(children)} columns in original) -->\n\n{md}"

    if t == "column":
        # columns only appear inside column_list and are handled via their children
        return "\n\n".join(render_blocks(children, synced, unhandled, 0))

    if t == "synced_block":
        synced_id = data.get("synced_from", {}).get("block_id") if data.get("synced_from") else block.get("id")
        comment = f"<!-- Notion synced block (id: {synced_id}) -->"
        md = "\n\n".join(render_blocks(children, synced, unhandled, 0))
        return f"{comment}\n\n{md}"

    if t == "child_page":
        title = data.get("title", "")
        return pad + f"[[{title}]]"

    if t == "child_database":
        title = data.get("title", "")
        return pad + f"[[{title}/_index|{title}]]"

    if t == "link_to_page":
        ref = data.get("page_id") or data.get("database_id")
        if ref and ref in synced:
            stem = Path(synced[ref]).stem
            return pad + f"[[{stem}]]"
        return pad + "<!-- link_to_page: unresolved -->"

    if t in ("table_of_contents", "breadcrumb", "unsupported", "template"):
        return ""

    # Unknown block type: emit a warning callout and track it.
    unhandled[t or "unknown"] += 1
    raw = json.dumps(block, indent=2, ensure_ascii=False)
    lines = [
        pad + "> [!warning] Unhandled Notion block type",
        pad + f"> Type: `{t}`",
        pad + "> ```json",
    ]
    for ln in raw.splitlines():
        lines.append(pad + "> " + ln)
    lines.append(pad + "> ```")
    return "\n".join(lines)


def _render_list_run(
    list_type: str,
    items: list[dict[str, Any]],
    synced: dict[str, str],
    unhandled: Counter,
    indent: int,
) -> str:
    pad = " " * indent
    lines = []
    for item in items:
        data = item.get(list_type, {})
        body = render_rich_text(data.get("rich_text", []), synced)
        if list_type == "bulleted_list_item":
            bullet = "- "
        elif list_type == "numbered_list_item":
            bullet = "1. "
        else:  # to_do
            checked = data.get("checked", False)
            bullet = "- [x] " if checked else "- [ ] "
        lines.append(pad + bullet + body)
        # Recurse into children with 4-space indent per Obsidian convention.
        children = item.get("children") or []
        if children:
            child_chunks = render_blocks(children, synced, unhandled, indent + 4)
            for c in child_chunks:
                lines.append(c)
    return "\n".join(lines)


def _render_media(
    block: dict[str, Any],
    data: dict[str, Any],
    synced: dict[str, str],
    kind: str,
) -> str:
    caption = render_rich_text(data.get("caption", []), synced)
    # Internal (Notion-hosted) files have type "file"; external have type "external".
    ftype = data.get("type")
    if ftype == "external":
        url = data.get("external", {}).get("url", "")
        if kind == "image":
            body = f"![{caption}]({url})"
        else:
            body = f"[{caption or url}]({url})"
        return body
    # Internal file. The orchestrator downloads these and rewrites the URL;
    # here we emit a stable placeholder keyed by the block id so the caller
    # can find and replace it after downloading.
    block_id = block.get("id", "unknown-block")
    placeholder = f"![[ATTACHMENT::{block_id}]]"
    if caption:
        placeholder += f"\n_{caption}_"
    return placeholder


def _render_table(block: dict[str, Any], synced: dict[str, str]) -> str:
    data = block.get("table", {})
    has_header = data.get("has_column_header", False)
    width = data.get("table_width", 0)
    children = block.get("children") or []
    rows = []
    for row_block in children:
        if row_block.get("type") != "table_row":
            continue
        cells = row_block.get("table_row", {}).get("cells", [])
        rendered_cells = []
        for cell in cells:
            s = render_rich_text(cell, synced).replace("|", "\\|").replace("\n", "<br>")
            rendered_cells.append(s)
        # Pad if fewer cells than table_width
        while len(rendered_cells) < width:
            rendered_cells.append("")
        rows.append(rendered_cells)

    if not rows:
        return ""

    if has_header:
        header = rows[0]
        body = rows[1:]
    else:
        header = [""] * (width or len(rows[0]))
        body = rows

    sep = ["---"] * len(header)
    lines = [
        "| " + " | ".join(header) + " |",
        "| " + " | ".join(sep) + " |",
    ]
    for r in body:
        lines.append("| " + " | ".join(r) + " |")
    return "\n".join(lines)


# --- Page frontmatter ------------------------------------------------------


def page_frontmatter(
    page: dict[str, Any],
    target_path: str,
    last_synced_at: str,
    extra: dict[str, Any] | None = None,
) -> dict[str, Any]:
    fm: dict[str, Any] = {
        "notion_id": page.get("id", ""),
        "notion_url": page.get("url", ""),
        "last_notion_edited_time": page.get("last_edited_time", ""),
        "last_synced_at": last_synced_at,
    }
    parent = page.get("parent") or {}
    parent_type = parent.get("type")
    if parent_type == "page_id":
        fm["notion_parent_id"] = parent.get("page_id", "")
    elif parent_type == "database_id":
        fm["db_parent"] = parent.get("database_id", "")
    if extra:
        fm.update(extra)
    return fm


def extract_title(page: dict[str, Any]) -> str:
    """Best-effort extraction of a page's visible title."""
    props = page.get("properties") or {}
    for prop in props.values():
        if prop.get("type") == "title":
            parts = prop.get("title") or []
            return "".join(p.get("plain_text", "") for p in parts).strip()
    # Fallback: some API responses expose a top-level title for workspace pages
    title = page.get("title")
    if isinstance(title, list):
        return "".join(t.get("plain_text", "") for t in title).strip()
    return ""


# --- Main ------------------------------------------------------------------


def convert(
    page: dict[str, Any],
    blocks: list[dict[str, Any]],
    target_path: str,
    synced: dict[str, str],
    last_synced_at: str,
    extra_frontmatter: dict[str, Any] | None = None,
) -> tuple[str, Counter]:
    unhandled: Counter = Counter()
    fm = page_frontmatter(page, target_path, last_synced_at, extra_frontmatter)
    title = extract_title(page)
    body_chunks = render_blocks(blocks, synced, unhandled)
    body = "\n\n".join(chunk for chunk in body_chunks if chunk)
    header = f"# {title}\n\n" if title else ""
    out = build_frontmatter(fm) + header + body
    if not out.endswith("\n"):
        out += "\n"
    return out, unhandled


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--page-json", required=True)
    p.add_argument("--blocks-json", required=True)
    p.add_argument("--target-path", required=True)
    p.add_argument("--synced-pages-json", default=None)
    p.add_argument("--last-synced-at", required=True)
    p.add_argument("--extra-frontmatter-json", default=None, help="JSON object merged into frontmatter (e.g. db row properties)")
    p.add_argument("--out", default=None)
    args = p.parse_args(argv)

    page = json.loads(Path(args.page_json).read_text(encoding="utf-8"))
    blocks = json.loads(Path(args.blocks_json).read_text(encoding="utf-8"))
    synced: dict[str, str] = {}
    if args.synced_pages_json:
        synced = json.loads(Path(args.synced_pages_json).read_text(encoding="utf-8"))
    extra = json.loads(args.extra_frontmatter_json) if args.extra_frontmatter_json else None

    out, unhandled = convert(page, blocks, args.target_path, synced, args.last_synced_at, extra)

    if args.out:
        Path(args.out).parent.mkdir(parents=True, exist_ok=True)
        Path(args.out).write_text(out, encoding="utf-8")
    else:
        sys.stdout.write(out)

    sys.stderr.write(json.dumps({"unhandled": dict(unhandled)}) + "\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
