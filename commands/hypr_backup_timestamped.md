---
name: hypr_backup_timestamped
description: Find ~/.config/hypr, pull dotfiles, and create a timestamped backup copy
version: "1.0"
---

Create a timestamped backup of `~/.config/hypr` into `~/repos/dotfiles` after pulling latest dotfiles changes.

## Hard Safety Rules

- Do not move, delete, or edit anything under any `.config` path.
- Copy only.
- Do not run destructive commands like `rm`, `mv`, or `git reset`.
- If any step fails, stop and report the failure.

## Steps

1. Resolve and verify source config directory:

```bash
SOURCE="$HOME/.config/hypr"
if [ ! -d "$SOURCE" ]; then
  echo "ERROR: $SOURCE does not exist or is not a directory"
  exit 1
fi
REAL_SOURCE="$(readlink -f "$SOURCE")"
echo "Using source: $REAL_SOURCE"
```

2. Pull latest dotfiles:

```bash
DOTFILES="$HOME/repos/dotfiles"
if [ ! -d "$DOTFILES/.git" ]; then
  echo "ERROR: $DOTFILES is not a git repository"
  exit 1
fi
git -C "$DOTFILES" pull --ff-only
```

3. Create timestamped backup directory and copy Hypr config:

```bash
TS="$(date +%Y%m%d-%H%M%S)"
DEST="$DOTFILES/stow/linux-framework-arch/other/BACKUP-hypr-config-$TS"
mkdir -p "$DEST"
cp -a "$SOURCE"/. "$DEST"/
cat > "$DEST/README-BACKUP.txt" <<EOM
Manual backup snapshot of ~/.config/hypr
Created: $TS
Source: $REAL_SOURCE
Policy: copy-only backup; no .config files moved/deleted/edited by this command
EOM
```

4. Verify backup contains same config files (ignoring README marker):

```bash
diff -qr "$SOURCE" "$DEST" | rg -v 'README-BACKUP.txt' || true
echo "Backup complete: $DEST"
```

## Output

- Print:
  - Resolved source path
  - `git pull` result
  - Final backup path
  - Verification summary
