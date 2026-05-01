#!/usr/bin/env bash
# detect-project.sh — scan the current directory and output build/test/run
# commands for the detected project type.
#
# Usage:
#   source scripts/detect-project.sh        # populates shell variables
#   scripts/detect-project.sh               # prints YAML fragment to stdout
#
# Outputs (when sourced):
#   DETECTED_TYPE, DETECTED_BUILD, DETECTED_TEST, DETECTED_RUN, DETECTED_LINT
#
# Detection runs top-to-bottom; the first match wins.

set -euo pipefail

# ── Helpers ───────────────────────────────────────────────────────────────────

has() { [[ -f "$1" ]] || [[ -d "$1" ]]; }

# ── Detection table ───────────────────────────────────────────────────────────
# Each block: set TYPE, BUILD, TEST, RUN, LINT, then break.

_detect() {
    local TYPE="" BUILD="" TEST="" RUN="" LINT=""

    # ESP-IDF (check before generic CMake)
    if has "sdkconfig" || has "idf_component.yml" || \
       grep -qsE "idf_component_register|esp_idf_component_register" CMakeLists.txt 2>/dev/null; then
        TYPE="esp-idf"
        BUILD=". ~/esp/esp-idf/export.sh && idf.py build"
        TEST=". ~/esp/esp-idf/export.sh && idf.py build 2>&1 | tail -5"
        RUN=". ~/esp/esp-idf/export.sh && idf.py flash monitor"
        LINT=""

    # Rust
    elif has "Cargo.toml"; then
        TYPE="rust"
        BUILD="cargo build"
        TEST="cargo test"
        RUN="cargo run"
        LINT="cargo clippy -- -D warnings"

    # Node.js / TypeScript — prefer pnpm > yarn > npm
    elif has "package.json"; then
        TYPE="nodejs"
        local PM="npm"
        has "pnpm-lock.yaml" && PM="pnpm"
        has "yarn.lock"      && PM="yarn"
        # Read test/build scripts from package.json if present
        local HAS_BUILD HAS_TEST HAS_LINT HAS_START
        HAS_BUILD=$(python3 -c "import json,sys; d=json.load(open('package.json')); print('yes' if 'build' in d.get('scripts',{}) else '')" 2>/dev/null || echo "")
        HAS_TEST=$(python3  -c "import json,sys; d=json.load(open('package.json')); print('yes' if 'test'  in d.get('scripts',{}) else '')" 2>/dev/null || echo "")
        HAS_LINT=$(python3  -c "import json,sys; d=json.load(open('package.json')); print('yes' if 'lint'  in d.get('scripts',{}) else '')" 2>/dev/null || echo "")
        HAS_START=$(python3 -c "import json,sys; d=json.load(open('package.json')); print('yes' if 'start' in d.get('scripts',{}) else '')" 2>/dev/null || echo "")
        BUILD="${HAS_BUILD:+$PM run build}"
        TEST="${HAS_TEST:+$PM test}"
        RUN="${HAS_START:+$PM start}"
        LINT="${HAS_LINT:+$PM run lint}"

    # Python — pyproject.toml (modern) > setup.py > requirements.txt
    elif has "pyproject.toml" || has "setup.py" || has "requirements.txt"; then
        TYPE="python"
        BUILD="pip install -e ."
        # prefer pytest; fall back to unittest
        if python3 -c "import pytest" 2>/dev/null; then
            TEST="pytest -x"
        else
            TEST="python -m unittest discover"
        fi
        RUN="python main.py"
        # prefer ruff, then flake8, then pylint
        if python3 -c "import ruff" 2>/dev/null || command -v ruff &>/dev/null; then
            LINT="ruff check ."
        elif command -v flake8 &>/dev/null; then
            LINT="flake8 ."
        fi

    # Go
    elif has "go.mod"; then
        TYPE="go"
        BUILD="go build ./..."
        TEST="go test ./..."
        RUN="go run ."
        LINT="$(command -v golangci-lint &>/dev/null && echo 'golangci-lint run' || echo '')"

    # Generic CMake (non-ESP-IDF)
    elif has "CMakeLists.txt"; then
        TYPE="c-cpp"
        BUILD="cmake -B build && cmake --build build"
        TEST="ctest --test-dir build"
        RUN=""
        LINT="$(command -v clang-tidy &>/dev/null && echo 'clang-tidy src/**/*.cpp' || echo '')"

    # Generic Makefile
    elif has "Makefile" || has "makefile"; then
        TYPE="make"
        BUILD="make"
        TEST="make test"
        RUN="make run"
        LINT=""

    # Unknown
    else
        TYPE="unknown"
        BUILD=""
        TEST=""
        RUN=""
        LINT=""
    fi

    # Export for sourcing
    DETECTED_TYPE="$TYPE"
    DETECTED_BUILD="$BUILD"
    DETECTED_TEST="$TEST"
    DETECTED_RUN="$RUN"
    DETECTED_LINT="$LINT"
}

_detect

# ── Output ────────────────────────────────────────────────────────────────────

# When sourced, variables are already set — just return
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    return 0
fi

# When run directly, print a YAML fragment
cat <<YAML
project_type: "$DETECTED_TYPE"
build_cmd:    "$DETECTED_BUILD"
test_cmd:     "$DETECTED_TEST"
run_cmd:      "$DETECTED_RUN"
lint_cmd:     "$DETECTED_LINT"
YAML

if [[ "$DETECTED_TYPE" == "unknown" ]]; then
    echo "" >&2
    echo "WARNING: project type not detected. Edit .planner-executor/config.yaml manually." >&2
    exit 1
fi
