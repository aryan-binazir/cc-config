---
name: hypr-backup-timestamped
description: Create a timestamped, verified backup of the user-owned Hyprland and Omarchy configuration inside `~/repos/dotfiles`. Use when the user asks to run `/hypr`, snapshot their Hypr setup, or back up their Omarchy desktop configuration.
disable-model-invocation: true
---

# Hypr Backup Timestamped

Create one copy-only snapshot containing both sides of the Omarchy desktop setup:

- `~/.config/hypr` contains the active Hyprland Lua configuration.
- `~/.config/omarchy` contains shell, idle/lock, bar, plugin, hook, branding, and custom theme configuration.

## Hard Safety Rules

- Treat both source directories as read-only.
- Copy only.
- Do not run destructive commands such as `rm`, `mv`, or `git reset`.
- Do not overwrite or reuse an existing backup directory.
- Leave the new backup uncommitted unless the user separately asks to commit or push it.
- If any step fails, stop and report the failure.

## Workflow

1. Verify that `~/.config/hypr` and `~/.config/omarchy` exist and resolve both real paths.
2. Verify that `~/repos/dotfiles` is a git repository.
3. Pull the latest dotfiles changes with a fast-forward-only strategy.
4. Create a timestamped backup directory at `~/repos/dotfiles/stow/arch-linux/other/BACKUP-hypr-config-<timestamp>`.
5. Copy `~/.config/hypr` to `<backup>/hypr` and `~/.config/omarchy` to `<backup>/omarchy`, preserving permissions, timestamps, and symlinks.
6. Write `README-BACKUP.txt` with the snapshot time, both resolved source paths, destination layout, and copy-only policy.
7. Verify each copied tree against its source with recursive, non-dereferencing diffs. Completion requires both diffs to be clean.

## Output

Print:
- both resolved source paths
- the `git pull` result
- the final backup path
- the verification result for each copied tree
