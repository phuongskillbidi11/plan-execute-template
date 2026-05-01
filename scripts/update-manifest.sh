#!/usr/bin/env bash
# update-manifest.sh — scan skills/*/SKILL.md and regenerate skills/manifest.json.
#
# Extraction rules:
#   name        ← first "# Skill: <name>" heading in the file
#   description ← first non-empty sentence after the "## Purpose" heading
#   path        ← relative path to the SKILL.md file
#
# Usage:
#   ./scripts/update-manifest.sh

set -euo pipefail

SKILLS_DIR="skills"
MANIFEST="${SKILLS_DIR}/manifest.json"

if [[ ! -d "$SKILLS_DIR" ]]; then
    echo "Error: $SKILLS_DIR/ directory not found. Run from project root."
    exit 1
fi

# ── Extract name + description from one SKILL.md ─────────────────────────────

extract_skill() {
    local file="$1"

    # name: first line matching "# Skill: <name>"
    local name
    name=$(grep -m1 '^# Skill:' "$file" | sed 's/^# Skill: *//' | tr -d '\r' | xargs)

    if [[ -z "$name" ]]; then
        echo "SKIP: $file (no '# Skill:' heading found)" >&2
        return 1
    fi

    # description: first non-empty, non-heading line after "## Purpose"
    local desc
    desc=$(awk '
        /^## Purpose/ { in_purpose=1; next }
        in_purpose && /^## / { exit }
        in_purpose && /^[^[:space:]>]/ && NF > 0 {
            # Take up to the first period or end of line
            line = $0
            idx = index(line, ".")
            if (idx > 0) line = substr(line, 1, idx)
            gsub(/^[[:space:]]+|[[:space:]]+$/, "", line)
            if (length(line) > 0) { print line; exit }
        }
    ' "$file" | tr -d '\r' | xargs)

    if [[ -z "$desc" ]]; then
        desc="(no description)"
    fi

    # Escape double quotes for JSON
    name="${name//\"/\\\"}"
    desc="${desc//\"/\\\"}"

    echo "$name|$desc"
}

# ── Build manifest ────────────────────────────────────────────────────────────

entries=()

for skill_file in "$SKILLS_DIR"/*/SKILL.md; do
    [[ -f "$skill_file" ]] || continue

    result=$(extract_skill "$skill_file") || continue

    name="${result%%|*}"
    desc="${result#*|}"
    # Normalise path to use forward slashes (Windows safety)
    path="${skill_file//\\//}"

    entries+=("$(printf '    {\n      "name": "%s",\n      "description": "%s",\n      "path": "%s"\n    }' "$name" "$desc" "$path")")
done

if [[ ${#entries[@]} -eq 0 ]]; then
    echo "No SKILL.md files found in $SKILLS_DIR/*/"
    exit 1
fi

# Join with commas
joined=""
for i in "${!entries[@]}"; do
    if [[ $i -gt 0 ]]; then
        joined+=$',\n'
    fi
    joined+="${entries[$i]}"
done

printf '{\n  "version": 1,\n  "skills": [\n%s\n  ]\n}\n' "$joined" > "$MANIFEST"

echo "Updated $MANIFEST with ${#entries[@]} skill(s):"
for skill_file in "$SKILLS_DIR"/*/SKILL.md; do
    [[ -f "$skill_file" ]] || continue
    result=$(extract_skill "$skill_file" 2>/dev/null) || continue
    name="${result%%|*}"
    printf "  %-20s ← %s\n" "$name" "$skill_file"
done
