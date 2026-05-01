#!/usr/bin/env bash
# plan-executor.sh — helper for the Planner-Executor workflow
#
# Usage:
#   ./scripts/plan-executor.sh new <feature-name>    scaffold a new plan folder
#   ./scripts/plan-executor.sh list                  list all plans and their status
#   ./scripts/plan-executor.sh status <folder>       show task completion for a plan
#   ./scripts/plan-executor.sh open <folder>         print plan files to stdout (paste into Copilot)
#   ./scripts/plan-executor.sh test <folder>         run the test command from config or tests.md

set -euo pipefail

PLANS_DIR=".plans"
TODAY=$(date +%Y-%m-%d)
CONFIG_FILE=".planner-executor/config.yaml"

# ── Config reader ─────────────────────────────────────────────────────────────
# Reads a key from config.yaml using python3 (available on all platforms).
# Falls back to empty string if config missing or key absent.

read_config() {
    local key="$1"
    if [[ ! -f "$CONFIG_FILE" ]]; then echo ""; return; fi
    python3 - "$CONFIG_FILE" "$key" <<'PY'
import sys, re
path, key = sys.argv[1], sys.argv[2]
for line in open(path):
    m = re.match(r'^\s*' + re.escape(key) + r'\s*:\s*"?(.*?)"?\s*$', line)
    if m:
        val = m.group(1).strip().strip('"')
        print(val)
        sys.exit(0)
print("")
PY
}

usage() {
    echo "Usage:"
    echo "  $0 new <feature-name>    Scaffold a new plan folder"
    echo "  $0 list                  List all plans and their completion status"
    echo "  $0 status <folder>       Show task completion for a plan"
    echo "  $0 open <folder>         Print plan files to stdout (paste into Copilot)"
    echo "  $0 test <folder>         Run build+test commands from config"
    exit 1
}

cmd_new() {
    local feature="${1:-}"
    if [[ -z "$feature" ]]; then
        echo "Error: feature name required"
        echo "  $0 new my-feature-name"
        exit 1
    fi

    local folder="${PLANS_DIR}/${TODAY}-${feature}"

    if [[ -d "$folder" ]]; then
        echo "Error: plan folder already exists: $folder"
        exit 1
    fi

    mkdir -p "$folder"

    cat > "$folder/spec.md" <<'SPEC'
# Spec: [Feature Name]

## Goal
[One paragraph: what problem does this solve, and why now?]

## Key discoveries
[Facts about the existing codebase that directly affect the design.]

## Design
[Describe the approach. Include ASCII diagrams if helpful.]

### Layout / structure
```
[diagram or pseudocode]
```

## Files changed
| File | Change |
|---|---|
| `path/to/file.ext` | [What changes and why] |

## Out of scope
- [Explicitly list what is NOT being done and why]
SPEC

    cat > "$folder/tasks.md" <<'TASKS'
# Tasks: [Feature Name]

Each task must be completed and its test (see `tests.md`) must pass before
moving to the next. Mark `[x]` when done.

---

## Task 1 — [Short description]

- [ ] **1.1** In `path/to/file.ext`, locate `[symbol]` at line ~N.
  [Exact instruction: what to add, remove, or replace.]

- [ ] **1.2** [Next atomic step — different file or different section.]

---

## Task 2 — [Short description]

- [ ] **2.1** [...]
TASKS

    cat > "$folder/tests.md" <<'TESTS'
# Tests: [Feature Name]

Run each test after completing the corresponding task. Stop and report on first
failure.

---

## T1 — [What is being verified]

```bash
[exact command to run]
```

**Pass:** [exact output or condition]
**Fail:** [what to look for — and what to report back]

---

## T2 — [...]
TESTS

    echo "Plan scaffolded: $folder"
    echo ""
    echo "Next steps:"
    echo "  1. Fill in $folder/spec.md"
    echo "  2. Fill in $folder/tasks.md"
    echo "  3. Fill in $folder/tests.md"
    echo "  4. Give Copilot: 'Implement the plan from $folder/'"
}

cmd_list() {
    if [[ ! -d "$PLANS_DIR" ]]; then
        echo "No .plans/ directory found."
        exit 0
    fi

    local found=0
    for folder in "$PLANS_DIR"/*/; do
        [[ -d "$folder" ]] || continue
        found=1
        local tasks_file="${folder}tasks.md"
        if [[ -f "$tasks_file" ]]; then
            local total done
            total=$(grep -c '^\- \[' "$tasks_file" 2>/dev/null || echo 0)
            done=$(grep -c '^\- \[x\]' "$tasks_file" 2>/dev/null || echo 0)
            printf "  %-45s  %s/%s tasks done\n" "$folder" "$done" "$total"
        else
            printf "  %-45s  (no tasks.md)\n" "$folder"
        fi
    done

    if [[ $found -eq 0 ]]; then
        echo "No plans found in $PLANS_DIR/"
    fi
}

cmd_status() {
    local folder="${1:-}"
    if [[ -z "$folder" ]]; then
        echo "Error: folder name required"
        echo "  $0 status .plans/YYYY-MM-DD-feature"
        exit 1
    fi

    # allow short name without .plans/ prefix
    if [[ ! -d "$folder" && -d "${PLANS_DIR}/${folder}" ]]; then
        folder="${PLANS_DIR}/${folder}"
    fi

    if [[ ! -d "$folder" ]]; then
        echo "Error: folder not found: $folder"
        exit 1
    fi

    local tasks_file="${folder}/tasks.md"
    if [[ ! -f "$tasks_file" ]]; then
        echo "Error: no tasks.md in $folder"
        exit 1
    fi

    echo "=== $folder ==="
    echo ""
    grep -E '^\- \[' "$tasks_file" | while IFS= read -r line; do
        if echo "$line" | grep -q '^\- \[x\]'; then
            echo "  DONE  $line"
        else
            echo "  TODO  $line"
        fi
    done
    echo ""

    local total done
    total=$(grep -c '^\- \[' "$tasks_file" 2>/dev/null || echo 0)
    done=$(grep -c '^\- \[x\]' "$tasks_file" 2>/dev/null || echo 0)
    echo "Progress: $done / $total tasks complete"
}

cmd_open() {
    local folder="${1:-}"
    if [[ -z "$folder" ]]; then
        echo "Error: folder name required"
        echo "  $0 open .plans/YYYY-MM-DD-feature"
        exit 1
    fi

    if [[ ! -d "$folder" && -d "${PLANS_DIR}/${folder}" ]]; then
        folder="${PLANS_DIR}/${folder}"
    fi

    if [[ ! -d "$folder" ]]; then
        echo "Error: folder not found: $folder"
        exit 1
    fi

    for file in spec.md tasks.md tests.md; do
        local path="${folder}/${file}"
        if [[ -f "$path" ]]; then
            echo "--- PLAN: ${file} ---"
            cat "$path"
            echo ""
        else
            echo "--- PLAN: ${file} --- (not found)"
        fi
    done
}

cmd_test() {
    # Run build + test commands from .planner-executor/config.yaml.
    # Falls back to prompting if config is missing.
    local env_setup build_cmd test_cmd
    env_setup=$(read_config "env_setup")
    build_cmd=$(read_config "build_cmd")
    test_cmd=$(read_config "test_cmd")

    if [[ -z "$build_cmd" && -z "$test_cmd" ]]; then
        echo "No commands found in $CONFIG_FILE"
        echo "Run scripts/init.sh first, or edit $CONFIG_FILE manually."
        exit 1
    fi

    if [[ -n "$env_setup" ]]; then
        echo "→ Environment setup: $env_setup"
        # shellcheck disable=SC1090
        eval "$env_setup"
    fi

    if [[ -n "$build_cmd" ]]; then
        echo "→ Build: $build_cmd"
        eval "$build_cmd"
    fi

    if [[ -n "$test_cmd" ]]; then
        echo "→ Test:  $test_cmd"
        eval "$test_cmd"
    fi
}

# ── main ──────────────────────────────────────────────────────────────────────

if [[ $# -eq 0 ]]; then
    usage
fi

COMMAND="$1"
shift

case "$COMMAND" in
    new)    cmd_new "$@" ;;
    list)   cmd_list ;;
    status) cmd_status "$@" ;;
    open)   cmd_open "$@" ;;
    test)   cmd_test ;;
    *)      echo "Unknown command: $COMMAND"; usage ;;
esac
