#!/usr/bin/env bash
set -euo pipefail

# sync-claude-plugins.sh
#
# Copies portable markdown from Claude Code plugins (~/.claude/plugins/cache/)
# into Codex (~/.codex/prompts/), OpenCode (~/.config/opencode/commands/),
# and Cursor (~/.cursor/commands/).
#
# Portable: commands/*.md, agents/*.md, skills/** (SKILL.md + examples + reference)
# Skipped:  hooks, scripts, .mcp.json, LSP-only and hook-only plugins
#
# Single-file plugins get promoted to a root SKILL.md.
# Multi-file plugins keep their tree and get a generated index SKILL.md.
#
# Idempotent — target dirs are wiped and recreated each run.
# Won't create tool config dirs if they don't exist.
# Requires: jq, rsync

PLUGINS_JSON="$HOME/.claude/plugins/installed_plugins.json"
CODEX_SKILLS="$HOME/.codex/prompts"
OPENCODE_SKILLS="$HOME/.config/opencode/commands"
CURSOR_SKILLS="$HOME/.cursor/commands"

# Plugins to skip entirely (no portable markdown content)
SKIP_PLUGINS=("security-guidance" "gopls-lsp" "typescript-lsp")

# --- Dependency checks ---

if ! command -v jq &>/dev/null; then
    echo "ERROR: jq is required but not installed. Install with: brew install jq" >&2
    exit 1
fi

if ! command -v rsync &>/dev/null; then
    echo "ERROR: rsync is required but not installed." >&2
    exit 1
fi

# --- Source check ---

if [[ ! -f "$PLUGINS_JSON" ]]; then
    echo "ERROR: installed_plugins.json not found at $PLUGINS_JSON" >&2
    exit 1
fi

# --- Determine active targets ---

targets=()
if [[ -d "$HOME/.codex" ]]; then
    targets+=("codex")
    mkdir -p "$CODEX_SKILLS"
else
    echo "[skip] Codex — ~/.codex/ does not exist"
fi

if [[ -d "$HOME/.config/opencode" ]]; then
    targets+=("opencode")
    mkdir -p "$OPENCODE_SKILLS"
else
    echo "[skip] OpenCode — ~/.config/opencode/ does not exist"
fi

if [[ -d "$HOME/.cursor" ]]; then
    targets+=("cursor")
    mkdir -p "$CURSOR_SKILLS"
else
    echo "[skip] Cursor — ~/.cursor/ does not exist"
fi

if [[ ${#targets[@]} -eq 0 ]]; then
    echo "No targets available. Nothing to sync."
    exit 0
fi

# --- Helper: get target root dir for a tool ---

target_dir() {
    local tool="$1"
    case "$tool" in
        codex)    echo "$CODEX_SKILLS" ;;
        opencode) echo "$OPENCODE_SKILLS" ;;
        cursor)   echo "$CURSOR_SKILLS" ;;
    esac
}

# --- Helper: check if plugin should be skipped ---

is_skipped() {
    local name="$1"
    for skip in "${SKIP_PLUGINS[@]}"; do
        if [[ "$name" == "$skip" ]]; then
            return 0
        fi
    done
    return 1
}

# --- Helper: validate plugin name is safe for filesystem paths ---
# Allows scoped names like "@scope/name" but blocks traversal/absolute paths.

is_safe_plugin_name() {
    local name="$1"
    [[ -n "$name" ]] || return 1
    [[ "$name" != /* ]] || return 1
    [[ "$name" != *"//"* ]] || return 1
    [[ "$name" != *"/./"* ]] || return 1
    [[ "$name" != "./"* ]] || return 1
    [[ "$name" != *"/." ]] || return 1
    [[ "$name" != *"/../"* ]] || return 1
    [[ "$name" != "../"* ]] || return 1
    [[ "$name" != *"/.." ]] || return 1
    [[ "$name" != *$'\n'* ]] || return 1
    [[ "$name" =~ ^[@A-Za-z0-9._/-]+$ ]] || return 1

    local segment
    IFS='/' read -r -a segments <<< "$name"
    for segment in "${segments[@]}"; do
        [[ -n "$segment" ]] || return 1
        [[ "$segment" != "." && "$segment" != ".." ]] || return 1
        [[ "$segment" =~ ^[@A-Za-z0-9._-]+$ ]] || return 1
    done

    return 0
}

# --- Helper: check if an array contains a given value ---

array_contains() {
    local needle="$1"
    shift
    local value
    for value in "$@"; do
        if [[ "$value" == "$needle" ]]; then
            return 0
        fi
    done
    return 1
}

# --- Helper: prune plugins removed since the previous sync ---

prune_removed_plugins() {
    local tool="$1"
    shift
    local current_plugins=("$@")

    local skills_root
    skills_root="$(target_dir "$tool")"
    local manifest="$skills_root/.claude-plugin-sync.manifest"

    if [[ -f "$manifest" ]]; then
        while IFS= read -r old_plugin; do
            [[ -z "$old_plugin" ]] && continue
            if ! array_contains "$old_plugin" "${current_plugins[@]-}"; then
                local old_dir="$skills_root/$old_plugin"
                if [[ -d "$old_dir" ]]; then
                    rm -rf "$old_dir"
                    echo "  [$tool] pruned removed plugin: $old_plugin"
                fi
            fi
        done < "$manifest"
    fi

    : > "$manifest"
    local plugin_name
    for plugin_name in "${current_plugins[@]-}"; do
        echo "$plugin_name" >> "$manifest"
    done
}

# --- Helper: escape a value for inline double-quoted YAML ---

yaml_escape_inline() {
    local text="$1"
    text="${text//$'\r'/ }"
    text="${text//$'\n'/ }"
    text="${text//\\/\\\\}"
    text="${text//\"/\\\"}"
    printf '%s' "$text"
}

# --- Helper: extract description from YAML frontmatter ---
# Reads a .md file, extracts the description field from --- fenced frontmatter.

extract_description() {
    local file="$1"
    local in_frontmatter=false
    while IFS= read -r line; do
        if [[ "$line" == "---" ]]; then
            if $in_frontmatter; then
                break
            else
                in_frontmatter=true
                continue
            fi
        fi
        if $in_frontmatter; then
            if [[ "$line" =~ ^description:\ *(.*) ]]; then
                local desc="${BASH_REMATCH[1]}"
                # Strip surrounding quotes
                desc="${desc#\"}"
                desc="${desc%\"}"
                desc="${desc#\'}"
                desc="${desc%\'}"
                echo "$desc"
                return
            fi
        fi
    done < "$file"
}

# --- Helper: extract name from plugin.json ---

plugin_description() {
    local install_path="$1"
    local pjson="$install_path/.claude-plugin/plugin.json"
    if [[ -f "$pjson" ]]; then
        jq -r '.description // empty' "$pjson" 2>/dev/null
    fi
}

# --- Helper: check if plugin has portable content ---
# Returns 0 if plugin has commands/, agents/, or skills/ with .md files.

has_portable_content() {
    local install_path="$1"
    local count=0
    for dir in commands agents skills; do
        if [[ -d "$install_path/$dir" ]]; then
            local md_count
            md_count=$(find "$install_path/$dir" -name "*.md" \
                -not -name "README.md" -not -name "LICENSE*" 2>/dev/null | wc -l)
            count=$((count + md_count))
        fi
    done
    [[ $count -gt 0 ]]
}

# --- Helper: count portable .md files ---

count_portable_md() {
    local install_path="$1"
    local count=0
    for dir in commands agents skills; do
        if [[ -d "$install_path/$dir" ]]; then
            local md_count
            md_count=$(find "$install_path/$dir" -name "*.md" \
                -not -name "README.md" -not -name "LICENSE*" 2>/dev/null | wc -l)
            count=$((count + md_count))
        fi
    done
    echo "$count"
}

# --- Helper: sync a single-file plugin ---
# For plugins with exactly one command or agent .md and no skills dir.
# Promotes the content to SKILL.md at the plugin root.

sync_single_file() {
    local plugin_name="$1"
    local source_file="$2"
    local plugin_desc="$3"

    local file_desc
    file_desc=$(extract_description "$source_file")
    local desc="${file_desc:-$plugin_desc}"
    local escaped_desc
    escaped_desc=$(yaml_escape_inline "$desc")

    # Read the file content (everything after frontmatter)
    local content=""
    local in_frontmatter=false
    local past_frontmatter=false
    while IFS= read -r line; do
        if [[ "$past_frontmatter" == true ]]; then
            content+="$line"$'\n'
            continue
        fi
        if [[ "$line" == "---" ]]; then
            if $in_frontmatter; then
                past_frontmatter=true
                continue
            else
                in_frontmatter=true
                continue
            fi
        fi
        if ! $in_frontmatter; then
            # No frontmatter — entire file is content
            past_frontmatter=true
            content+="$line"$'\n'
        fi
    done < "$source_file"

    for tool in "${targets[@]}"; do
        local dest_dir
        dest_dir="$(target_dir "$tool")/$plugin_name"
        # Clean previous sync (may have been multi-file before)
        rm -rf "$dest_dir"
        mkdir -p "$dest_dir"

        # Write SKILL.md
        cat > "$dest_dir/SKILL.md" <<EOF
---
name: $plugin_name
description: "$escaped_desc"
---

$content
EOF
        echo "  [$tool] $plugin_name → $dest_dir/SKILL.md"
    done
}

# --- Helper: sync a multi-file plugin ---
# Copies commands/, agents/, skills/ subdirs and generates an index SKILL.md.

sync_multi_file() {
    local plugin_name="$1"
    local install_path="$2"
    local plugin_desc="$3"

    for tool in "${targets[@]}"; do
        local dest_dir
        dest_dir="$(target_dir "$tool")/$plugin_name"
        # Clean entire plugin dir so stale subdirs don't persist
        rm -rf "$dest_dir"
        mkdir -p "$dest_dir"

        # Sync each portable directory
        for dir in commands agents skills; do
            if [[ -d "$install_path/$dir" ]]; then
                # Check for .md files (excluding README, LICENSE)
                local md_count
                md_count=$(find "$install_path/$dir" -name "*.md" \
                    -not -name "README.md" -not -name "LICENSE*" 2>/dev/null | wc -l)
                if [[ $md_count -gt 0 ]]; then
                    # Clean target dir to remove stale files from previous syncs
                    rm -rf "$dest_dir/$dir"
                    mkdir -p "$dest_dir/$dir"
                    rsync -a --prune-empty-dirs \
                        --exclude='README.md' \
                        --exclude='LICENSE*' \
                        --exclude='.git' \
                        --exclude='.mcp.json' \
                        --exclude='*.py' \
                        --exclude='*.sh' \
                        --exclude='*.json' \
                        --include='*/' \
                        --include='*.md' \
                        --exclude='*' \
                        "$install_path/$dir/" "$dest_dir/$dir/"
                fi
            fi
        done

        # Generate index SKILL.md if one doesn't exist from skills/
        # (skills/ dirs have their own SKILL.md files, but the plugin root needs one too)
        local index="$dest_dir/SKILL.md"
        local escaped_plugin_desc
        escaped_plugin_desc=$(yaml_escape_inline "$plugin_desc")
        {
            echo "---"
            echo "name: $plugin_name"
            echo "description: \"$escaped_plugin_desc\""
            echo "---"
            echo ""
            echo "# $plugin_name"
            echo ""
            if [[ -n "$plugin_desc" ]]; then
                echo "$plugin_desc"
                echo ""
            fi

            # List commands
            if [[ -d "$dest_dir/commands" ]]; then
                local cmd_files
                cmd_files=$(find "$dest_dir/commands" -name "*.md" 2>/dev/null | sort)
                if [[ -n "$cmd_files" ]]; then
                    echo "## Commands"
                    echo ""
                    while IFS= read -r cmd_file; do
                        local cmd_name
                        cmd_name=$(basename "$cmd_file" .md)
                        local cmd_desc
                        cmd_desc=$(extract_description "$cmd_file")
                        if [[ -n "$cmd_desc" ]]; then
                            echo "- **$cmd_name**: $cmd_desc"
                        else
                            echo "- **$cmd_name**"
                        fi
                    done <<< "$cmd_files"
                    echo ""
                fi
            fi

            # List agents
            if [[ -d "$dest_dir/agents" ]]; then
                local agent_files
                agent_files=$(find "$dest_dir/agents" -name "*.md" 2>/dev/null | sort)
                if [[ -n "$agent_files" ]]; then
                    echo "## Agents"
                    echo ""
                    while IFS= read -r agent_file; do
                        local agent_name
                        agent_name=$(basename "$agent_file" .md)
                        local agent_desc
                        agent_desc=$(extract_description "$agent_file")
                        if [[ -n "$agent_desc" ]]; then
                            echo "- **$agent_name**: $agent_desc"
                        else
                            echo "- **$agent_name**"
                        fi
                    done <<< "$agent_files"
                    echo ""
                fi
            fi

            # List skills
            if [[ -d "$dest_dir/skills" ]]; then
                local skill_dirs
                skill_dirs=$(find "$dest_dir/skills" -name "SKILL.md" -not -path "$dest_dir/SKILL.md" 2>/dev/null | sort)
                if [[ -n "$skill_dirs" ]]; then
                    echo "## Skills"
                    echo ""
                    while IFS= read -r skill_md; do
                        local skill_name
                        skill_name=$(basename "$(dirname "$skill_md")")
                        local skill_desc
                        skill_desc=$(extract_description "$skill_md")
                        if [[ -n "$skill_desc" ]]; then
                            echo "- **$skill_name**: $skill_desc"
                        else
                            echo "- **$skill_name**"
                        fi
                    done <<< "$skill_dirs"
                    echo ""
                fi
            fi
        } > "$index"

        echo "  [$tool] $plugin_name → $dest_dir/ ($(count_portable_md "$install_path") files)"
    done
}

# --- Main ---

echo "Syncing Claude Code plugins to: ${targets[*]}"
echo ""

# Parse installed_plugins.json — get unique plugin names and their install paths.
# Take the first installPath for each plugin (handles multiple scopes).
plugin_entries=$(jq -r '
    .plugins
    | to_entries
    | map({
        key: .key,
        path: (.value[0].installPath // "")
      })
    | map(
        . + {
          name: (
            .key
            | if test("^.+@[^@]+$")
              then sub("@[^@]+$"; "")
              else .
              end
          )
        }
      )
    | sort_by(.name)
    | group_by(.name)
    | map(.[0])
    | .[]
    | "\(.name)\t\(.path)"
' "$PLUGINS_JSON")

synced=0
skipped=0
synced_plugins=()

if [[ -z "$plugin_entries" ]]; then
    echo "No plugins found in $PLUGINS_JSON"
else
    while IFS=$'\t' read -r plugin_name install_path; do
        [[ -z "$plugin_name" ]] && continue

        if ! is_safe_plugin_name "$plugin_name"; then
            echo "[skip] $plugin_name — unsafe plugin name"
            skipped=$((skipped + 1))
            continue
        fi

        # Skip non-portable plugins
        if is_skipped "$plugin_name"; then
            echo "[skip] $plugin_name — no portable content"
            skipped=$((skipped + 1))
            continue
        fi

        # Verify install path exists
        if [[ ! -d "$install_path" ]]; then
            echo "[skip] $plugin_name — install path not found: $install_path"
            skipped=$((skipped + 1))
            continue
        fi

        # Check for portable content
        if ! has_portable_content "$install_path"; then
            echo "[skip] $plugin_name — no portable markdown found"
            skipped=$((skipped + 1))
            continue
        fi

        # Get plugin description
        desc=$(plugin_description "$install_path")

        # Count portable directories with content
        portable_dirs=0
        portable_files=0
        single_file=""
        for dir in commands agents skills; do
            if [[ -d "$install_path/$dir" ]]; then
                local_count=$(find "$install_path/$dir" -name "*.md" \
                    -not -name "README.md" -not -name "LICENSE*" 2>/dev/null | wc -l)
                local_count=$((local_count + 0))  # trim whitespace
                if [[ $local_count -gt 0 ]]; then
                    portable_dirs=$((portable_dirs + 1))
                    portable_files=$((portable_files + local_count))
                    if [[ $local_count -eq 1 && "$dir" != "skills" ]]; then
                        single_file=$(find "$install_path/$dir" -name "*.md" \
                            -not -name "README.md" -not -name "LICENSE*" 2>/dev/null | head -1)
                    fi
                fi
            fi
        done

        echo "[sync] $plugin_name ($portable_files portable files)"

        # Single command/agent with no skills → promote to SKILL.md
        if [[ $portable_files -eq 1 && $portable_dirs -eq 1 && -n "$single_file" ]]; then
            sync_single_file "$plugin_name" "$single_file" "$desc"
        else
            sync_multi_file "$plugin_name" "$install_path" "$desc"
        fi

        synced=$((synced + 1))
        synced_plugins+=("$plugin_name")

    done <<< "$plugin_entries"
fi

for tool in "${targets[@]}"; do
    prune_removed_plugins "$tool" "${synced_plugins[@]-}"
done

echo ""
echo "Done. Synced $synced plugins, skipped $skipped."
