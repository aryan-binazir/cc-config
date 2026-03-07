#!/bin/bash
branch=$(git branch --show-current 2>/dev/null | tr '/' '-')
[ -z "$branch" ] && exit 0

f="_scratch/_context/${branch}.md"
if [ -f "$f" ]; then
  echo "[Context loaded: $f]"
  cat "$f"
else
  echo "[No context file: $f]"
fi

exit 0
