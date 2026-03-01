#!/bin/bash
branch=$(git branch --show-current 2>/dev/null | tr '/' '-')
[ -z "$branch" ] && exit 0

# Mainline branches intentionally do not auto-load context.
case "$branch" in
  main|master) exit 0 ;;
esac

f="_context/CONTEXT-${branch}.md"
if [ -f "$f" ]; then
  echo "[Context loaded: $f]"
  cat "$f"
else
  echo "[No context file: $f]"
fi

exit 0
