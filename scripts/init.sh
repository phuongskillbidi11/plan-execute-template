#!/usr/bin/env bash
# init.sh — initialize the Planner-Executor workflow in any project.
#
# Usage (run from your project root):
#   bash path/to/planner-executor-template/scripts/init.sh
#   bash path/to/planner-executor-template/scripts/init.sh --upgrade
#
# Normal mode:  copies all template files, detects project type, writes config.
# --upgrade:    only refreshes scripts/ and .claude/planner-instructions.md.
#               Never overwrites CLAUDE.md, copilot-instructions.md, or config.yaml.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TARGET_DIR="$(pwd)"
UPGRADE=false

# ── Parse args ────────────────────────────────────────────────────────────────

for arg in "$@"; do
    case "$arg" in
        --upgrade) UPGRADE=true ;;
        --help|-h)
            echo "Usage: init.sh [--upgrade]"
            echo ""
            echo "  (no flags)   Full init: copy all template files + detect project + write config"
            echo "  --upgrade    Refresh scripts and planner instructions only (preserves customizations)"
            exit 0
            ;;
        *) echo "Unknown argument: $arg"; exit 1 ;;
    esac
done

# ── Helpers ───────────────────────────────────────────────────────────────────

info()    { echo "  [OK]  $*"; }
skip()    { echo " [SKIP] $*"; }
warn()    { echo " [WARN] $*" >&2; }

copy_if_missing() {
    local src="$1" dst="$2"
    if [[ -f "$dst" ]]; then
        skip "$dst (already exists — not overwritten)"
    else
        mkdir -p "$(dirname "$dst")"
        cp "$src" "$dst"
        info "Created $dst"
    fi
}

copy_always() {
    local src="$1" dst="$2"
    mkdir -p "$(dirname "$dst")"
    cp "$src" "$dst"
    info "Updated $dst"
}

# ── Main ──────────────────────────────────────────────────────────────────────

echo ""
echo "Planner-Executor init${UPGRADE:+ (upgrade mode)}"
echo "Target: $TARGET_DIR"
echo ""

# -- Scripts (always refreshed) -----------------------------------------------
echo "→ Refreshing scripts..."
mkdir -p "$TARGET_DIR/scripts"
for f in plan-executor.sh detect-project.sh init.sh; do
    src="$TEMPLATE_DIR/scripts/$f"
    [[ -f "$src" ]] && copy_always "$src" "$TARGET_DIR/scripts/$f"
done
chmod +x "$TARGET_DIR/scripts/"*.sh

# -- Planner instructions (always refreshed) ----------------------------------
echo "→ Refreshing .claude/planner-instructions.md..."
copy_always \
    "$TEMPLATE_DIR/.claude/planner-instructions.md" \
    "$TARGET_DIR/.claude/planner-instructions.md"

if $UPGRADE; then
    echo ""
    echo "Upgrade complete. User-owned files were not touched."
    echo ""
    exit 0
fi

# -- User-owned files (copy only if missing) ----------------------------------
echo "→ Copying template files (skip if already present)..."

copy_if_missing \
    "$TEMPLATE_DIR/CLAUDE.md" \
    "$TARGET_DIR/CLAUDE.md"

copy_if_missing \
    "$TEMPLATE_DIR/.github/copilot-instructions.md" \
    "$TARGET_DIR/.github/copilot-instructions.md"

copy_if_missing \
    "$TEMPLATE_DIR/ROADMAP.md" \
    "$TARGET_DIR/ROADMAP.md"

# -- Project detection --------------------------------------------------------
echo ""
echo "→ Detecting project type..."

# Source detect-project.sh to populate DETECTED_* variables
# shellcheck source=detect-project.sh
source "$TEMPLATE_DIR/scripts/detect-project.sh" || true

echo "   Detected: ${DETECTED_TYPE:-unknown}"
[[ -n "${DETECTED_BUILD:-}" ]] && echo "   Build:    $DETECTED_BUILD"
[[ -n "${DETECTED_TEST:-}"  ]] && echo "   Test:     $DETECTED_TEST"
[[ -n "${DETECTED_RUN:-}"   ]] && echo "   Run:      $DETECTED_RUN"
[[ -n "${DETECTED_LINT:-}"  ]] && echo "   Lint:     $DETECTED_LINT"

# -- Write config.yaml --------------------------------------------------------
CONFIG_DIR="$TARGET_DIR/.planner-executor"
CONFIG_FILE="$CONFIG_DIR/config.yaml"

if [[ -f "$CONFIG_FILE" ]]; then
    skip ".planner-executor/config.yaml (already exists — not overwritten)"
else
    mkdir -p "$CONFIG_DIR"
    # Extract project name from directory name
    PROJECT_NAME="$(basename "$TARGET_DIR")"

    sed \
        -e "s|\[PROJECT_NAME\]|$PROJECT_NAME|g" \
        -e "s|\[PROJECT_TYPE\]|${DETECTED_TYPE:-unknown}|g" \
        -e "s|\[BUILD_COMMAND\]|${DETECTED_BUILD:-}|g" \
        -e "s|\[TEST_COMMAND\]|${DETECTED_TEST:-}|g" \
        -e "s|\[RUN_COMMAND\]|${DETECTED_RUN:-}|g" \
        -e "s|\[LINT_COMMAND\]|${DETECTED_LINT:-}|g" \
        "$TEMPLATE_DIR/.planner-executor/config.yaml.template" \
        > "$CONFIG_FILE"

    info "Created .planner-executor/config.yaml"
fi

# -- Create .plans/ directory -------------------------------------------------
mkdir -p "$TARGET_DIR/.plans"
info "Ensured .plans/ directory exists"

# -- Populate CLAUDE.md placeholders (best-effort sed) -----------------------
CLAUDE_FILE="$TARGET_DIR/CLAUDE.md"
if [[ -f "$CLAUDE_FILE" ]] && grep -q '\[BUILD_COMMAND\]' "$CLAUDE_FILE" 2>/dev/null; then
    PROJECT_NAME="$(basename "$TARGET_DIR")"
    sed -i \
        -e "s|\[PROJECT_NAME\]|$PROJECT_NAME|g" \
        -e "s|\[PROJECT_TYPE\]|${DETECTED_TYPE:-unknown}|g" \
        -e "s|\[BUILD_COMMAND\]|${DETECTED_BUILD:-}|g" \
        -e "s|\[TEST_COMMAND\]|${DETECTED_TEST:-}|g" \
        -e "s|\[RUN_COMMAND\]|${DETECTED_RUN:-}|g" \
        -e "s|\[LINT_COMMAND\]|${DETECTED_LINT:-}|g" \
        -e "s|\[MAIN_LANGUAGE\]|${DETECTED_TYPE:-unknown}|g" \
        "$CLAUDE_FILE"
    info "Populated placeholders in CLAUDE.md"
fi

# -- Summary ------------------------------------------------------------------
echo ""
echo "Done. Next steps:"
echo ""
echo "  1. Review .planner-executor/config.yaml — correct any wrong commands"
echo "  2. Fill in CLAUDE.md: architecture, key files, coding conventions"
echo "  3. (Optional) /plugin install andrej-karpathy-skills@karpathy-skills"
echo "  4. Ask Claude: 'Create a plan for <feature>. Save to .plans/'"
echo ""

if [[ "${DETECTED_TYPE:-unknown}" == "unknown" ]]; then
    warn "Project type was not detected automatically."
    warn "Edit .planner-executor/config.yaml to set build_cmd and test_cmd."
    echo ""
fi
