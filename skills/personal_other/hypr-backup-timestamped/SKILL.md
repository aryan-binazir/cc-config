---
name: hypr-backup-timestamped
description: Create a timestamped, verified, copy-only backup of Hyprland, Omarchy, Ghostty, and keyd configuration in ~/repos/dotfiles, including keyboard-remapper restore metadata.
disable-model-invocation: true
---

# Hypr Backup Timestamped

Snapshot the desktop, terminal, and keyboard-remapping setup into one fresh backup.

| Source | Destination inside the backup |
| --- | --- |
| `~/.config/hypr` | `hypr/` |
| `~/.config/omarchy` | `omarchy/` |
| `~/.config/ghostty` | `ghostty/` |
| `/etc/keyd` | `keyd/` |

## Safety

- Copy only: all sources stay read-only, and destructive commands (`rm`, `mv`, `git reset`) stay off the table. Record service state without installing packages, reloading configs, or changing services.
- Inspect the source inventory and review Ghostty/keyd files for credentials before copying. If secrets or private keys are present, stop and ask how to exclude them; report paths, not secret values. Capture only the service properties below, not logs or environment dumps.
- Every run writes a new backup directory; existing backups stay untouched.
- Leave the backup uncommitted unless the user separately asks to commit or push it.
- Stop and report on any failure.

## Workflow

1. Verify every source in the table exists and resolve all real paths. Complete the sensitivity check before writing a backup.
2. Verify `~/repos/dotfiles` is a git repository.
3. Run `git pull --ff-only` there, outside the Codex sandbox.
4. Create a new `~/repos/dotfiles/stow/arch-linux/other/BACKUP-hypr-config-<timestamp>` directory. Use a collision-resistant timestamp and refuse to reuse an existing destination.
5. Copy each source to its destination from the table, preserving permissions, timestamps, and symlinks without following symlinks. Use `cp -a --no-preserve=ownership` so root-owned keyd files can be saved as the current user. Request read privileges if needed rather than changing source permissions.
6. Write `KEYD-STATE.txt` with the output of `pacman -Q keyd` and `systemctl show keyd.service udevmon.service -p Id -p LoadState -p ActiveState -p UnitFileState`. This captures the package version and actual active/enabled states instead of assuming that copying `/etc/keyd` reproduces the service setup.
7. Write `README-BACKUP.txt` with the snapshot time and timezone, every resolved source path, destination layout, and copy-only policy. Include these restore notes as documentation only:
   - Restore the three user config trees to their original locations.
   - Install keyd using `sudo pacman -S --needed keyd`, restore `keyd/` into `/etc/keyd` with root ownership and preserved permissions, and validate with `sudo keyd check`.
   - Use `KEYD-STATE.txt` to restore keyd/udevmon active and enabled states. For a snapshot with keyd active/enabled and udevmon inactive/disabled, disable and stop udevmon before enabling and starting keyd. They must not grab the same keyboard concurrently.
   - Applying these restore steps changes the live system and requires a separate user request; backing up does not apply them.
8. Verify all four copied trees against their sources with `diff -r --no-dereference`. Done when every diff is clean and both metadata files contain the required fields.

## Output

Print every resolved source path, the sensitivity-check result, the `git pull` result, the final backup path, and each tree's verification result. Confirm that keyd package/service state and restore instructions were saved and that the backup remains uncommitted.
