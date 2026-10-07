---
name: hypr-backup-timestamped
description: Create a timestamped, verified, copy-only backup of the Hyprland and Omarchy configuration in `~/repos/dotfiles`. Use when the user runs `/hypr` or asks to snapshot or back up their Hypr/Omarchy desktop setup.
disable-model-invocation: true
---

# Hypr Backup Timestamped

Snapshot both halves of the Omarchy desktop setup, `~/.config/hypr` and `~/.config/omarchy`, into one fresh backup.

## Safety

- Copy only: both sources stay read-only, and destructive commands (`rm`, `mv`, `git reset`) stay off the table.
- Every run writes a new backup directory; existing backups stay untouched.
- Leave the backup uncommitted unless the user separately asks to commit or push it.
- Stop and report on any failure.

## Workflow

1. Verify `~/.config/hypr` and `~/.config/omarchy` exist and resolve both real paths.
2. Verify `~/repos/dotfiles` is a git repository.
3. Run `git pull --ff-only` there, outside the Codex sandbox.
4. Create `~/repos/dotfiles/stow/arch-linux/other/BACKUP-hypr-config-<timestamp>`.
5. Copy `~/.config/hypr` to `<backup>/hypr` and `~/.config/omarchy` to `<backup>/omarchy`, preserving permissions, timestamps, and symlinks.
6. Write `README-BACKUP.txt` with the snapshot time, both resolved source paths, destination layout, and copy-only policy.
7. Verify each copied tree against its source with a recursive, non-dereferencing diff. Done when both are clean.

## Output

Print both resolved source paths, the `git pull` result, the final backup path, and each tree's verification result.
