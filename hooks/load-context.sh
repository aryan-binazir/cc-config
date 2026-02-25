#!/bin/bash
branch=$(git branch --show-current 2>/dev/null | tr '/' '-')
[ -z "$branch" ] && exit 0
f="_context/CONTEXT-${branch}.md"
[ -f "$f" ] && echo "[Context loaded: $f]" && cat "$f"
exit 0
