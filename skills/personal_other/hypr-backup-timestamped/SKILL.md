---
name: hypr-backup-timestamped
description: Create a timestamped backup of `~/.config/hypr` inside `~/repos/dotfiles` after pulling the latest dotfiles changes, using a copy-only workflow with verification. Use when the user asks to snapshot or back up their Hypr config into the dotfiles repo.
---

# Hypr Backup Timestamped

Create a timestamped copy-only backup of the Hypr config.

## Hard Safety Rules

- Never move, delete, or edit anything under `.config`.
- Copy only.
- Do not run destructive commands such as `rm`, `mv`, or `git reset`.
- If any step fails, stop and report the failure.

## Workflow

1. Verify that `~/.config/hypr` exists and resolve its real path.
2. Verify that `~/repos/dotfiles` is a git repository.
3. Pull the latest dotfiles changes with a fast-forward-only strategy.
4. Create a timestamped backup directory at `~/repos/dotfiles/stow/linux-framework-arch/other/BACKUP-hypr-config-<timestamp>`.
5. Copy the Hypr config into that directory.
6. Write a short `README-BACKUP.txt` describing the snapshot time, source path, and copy-only policy.
7. Verify the backup by diffing source and destination while ignoring the README marker.

## Output

Print:
- the resolved source path
- the `git pull` result
- the final backup path
- the verification summary
